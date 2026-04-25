// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

const fetchLoopDefaultInterval = 5 * time.Minute

type fetchRepoRef struct {
	Org  string
	Repo string
}

// go s.runFetchLoop(ctx, 5*time.Minute)
func (s *PrepSubsystem) runFetchLoop(ctx context.Context, interval time.Duration) {
	if s == nil || s.ServiceRuntime == nil || ctx == nil || interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.runFetchLoopTicks(ctx, ticker.C)
}

func (s *PrepSubsystem) runFetchLoopTicks(ctx context.Context, ticks <-chan time.Time) {
	if s == nil || s.ServiceRuntime == nil || ctx == nil || ticks == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			s.fetchRegisteredRepos(ctx)
		}
	}
}

func (s *PrepSubsystem) fetchLoopInterval() time.Duration {
	if s != nil && s.ServiceRuntime != nil {
		if result := s.Core().Config().Get("agents.fetch_interval"); result.OK {
			if interval := fetchLoopDuration(result.Value); interval > 0 {
				return interval
			}
		}
	}

	for _, path := range s.fetchLoopConfigPaths() {
		raw := fetchLoopReadConfig(path)
		if interval := fetchLoopDuration(raw["fetch_interval"]); interval > 0 {
			return interval
		}
		if dispatch, ok := raw["dispatch"].(map[string]any); ok {
			if interval := fetchLoopDuration(dispatch["fetch_interval"]); interval > 0 {
				return interval
			}
		}
	}

	return fetchLoopDefaultInterval
}

func (s *PrepSubsystem) fetchRegisteredRepos(ctx context.Context) {
	if s == nil || s.ServiceRuntime == nil || ctx == nil {
		return
	}

	seen := map[string]bool{}
	for _, ref := range s.fetchLoopRepoRefs() {
		if ctx.Err() != nil {
			return
		}

		name := fetchLoopRepoName(ref)
		repoDir := s.localRepoDir(ref.Org, ref.Repo)
		if repoDir == "" || !fs.IsDir(core.JoinPath(repoDir, ".git")) {
			core.Warn("agentic fetch loop skipped repo", "repo", name, "path", repoDir)
			continue
		}
		if seen[repoDir] {
			continue
		}
		seen[repoDir] = true

		branch := s.DefaultBranch(repoDir)
		args := []string{"git", "fetch", "origin"}
		if branch != "" {
			args = append(args, branch)
		}

		result := s.Core().Process().RunIn(ctx, repoDir, args...)
		if !result.OK {
			core.Warn("agentic fetch loop failed", "repo", name, "branch", branch, "reason", result.Value)
			continue
		}
		core.Info("agentic fetch loop fetched repo", "repo", name, "branch", branch)
	}
}

func (s *PrepSubsystem) fetchLoopRepoRefs() []fetchRepoRef {
	seen := map[string]bool{}
	refs := []fetchRepoRef{}

	add := func(org, repo string) {
		orgName, ok := validateName(org)
		if !ok {
			return
		}
		repoName, ok := validateName(repo)
		if !ok {
			return
		}
		key := core.Concat(orgName, "/", repoName)
		if seen[key] {
			return
		}
		seen[key] = true
		refs = append(refs, fetchRepoRef{Org: orgName, Repo: repoName})
	}

	if s != nil && s.ServiceRuntime != nil {
		if result := s.Core().Config().Get("agents.fetch_repos"); result.OK {
			fetchLoopCollectRepoRefs(result.Value, add)
		}
	}

	for _, path := range s.fetchLoopConfigPaths() {
		raw := fetchLoopReadConfig(path)
		fetchLoopCollectRepoRefs(raw["repos"], add)
		if agents, ok := raw["agents"].(map[string]any); ok {
			for _, value := range agents {
				agent, ok := value.(map[string]any)
				if !ok {
					continue
				}
				fetchLoopCollectRepoRefs(agent["repos"], add)
			}
		}
	}

	for _, repoDir := range core.PathGlob(core.JoinPath(WorkspaceRoot(), "*", "*")) {
		if !fs.IsDir(repoDir) {
			continue
		}
		org := core.PathBase(core.PathDir(repoDir))
		repo := core.PathBase(repoDir)
		add(org, repo)
	}

	return refs
}

func (s *PrepSubsystem) fetchLoopConfigPaths() []string {
	paths := []string{}
	seen := map[string]bool{}
	add := func(path string) {
		clean := core.Trim(path)
		if clean == "" || seen[clean] {
			return
		}
		seen[clean] = true
		paths = append(paths, clean)
	}

	if s != nil && s.ServiceRuntime != nil {
		if result := s.Core().Config().Get("agents.config_path"); result.OK {
			if path, ok := result.Value.(string); ok {
				add(path)
			}
		}
	}

	add(core.JoinPath(CoreRoot(), "agents.yaml"))
	if s != nil {
		add(core.JoinPath(s.codePath, "core", "agent", "config", "agents.yaml"))
	}

	return paths
}

func fetchLoopReadConfig(path string) map[string]any {
	readResult := fs.Read(path)
	if !readResult.OK {
		return map[string]any{}
	}

	var raw map[string]any
	if err := yaml.Unmarshal([]byte(readResult.Value.(string)), &raw); err != nil {
		return map[string]any{}
	}

	return raw
}

func fetchLoopCollectRepoRefs(value any, add func(org, repo string)) {
	appendRepo := func(raw string) {
		org, repo, ok := fetchLoopParseRepo(raw)
		if !ok {
			return
		}
		add(org, repo)
	}

	switch typed := value.(type) {
	case string:
		appendRepo(typed)
	case []string:
		for _, raw := range typed {
			appendRepo(raw)
		}
	case []any:
		for _, item := range typed {
			if raw, ok := item.(string); ok {
				appendRepo(raw)
			}
		}
	case map[string]any:
		for raw := range typed {
			appendRepo(raw)
		}
	}
}

func fetchLoopParseRepo(raw string) (string, string, bool) {
	value := core.Trim(raw)
	if value == "" {
		return "", "", false
	}

	parts := core.Split(value, "/")
	switch len(parts) {
	case 1:
		org, ok := validateName("core")
		if !ok {
			return "", "", false
		}
		repo, ok := validateName(parts[0])
		return org, repo, ok
	case 2:
		org, ok := validateName(parts[0])
		if !ok {
			return "", "", false
		}
		repo, ok := validateName(parts[1])
		return org, repo, ok
	default:
		return "", "", false
	}
}

func fetchLoopDuration(value any) time.Duration {
	switch typed := value.(type) {
	case time.Duration:
		if typed > 0 {
			return typed
		}
	case string:
		parsed, err := time.ParseDuration(core.Trim(typed))
		if err == nil && parsed > 0 {
			return parsed
		}
	case int:
		if typed > 0 {
			return time.Duration(typed) * time.Second
		}
	case int64:
		if typed > 0 {
			return time.Duration(typed) * time.Second
		}
	case float64:
		if typed > 0 {
			return time.Duration(typed * float64(time.Second))
		}
	}

	return 0
}

func fetchLoopRepoName(ref fetchRepoRef) string {
	return core.Concat(ref.Org, "/", ref.Repo)
}
