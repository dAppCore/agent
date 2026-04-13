// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MirrorInput struct {
	Repo     string `json:"repo,omitempty"`
	DryRun   bool   `json:"dry_run,omitempty"`
	MaxFiles int    `json:"max_files,omitempty"`
}

type MirrorOutput struct {
	Success bool         `json:"success"`
	Synced  []MirrorSync `json:"synced"`
	Skipped []string     `json:"skipped,omitempty"`
	Count   int          `json:"count"`
}

type MirrorSync struct {
	Repo         string `json:"repo"`
	CommitsAhead int    `json:"commits_ahead"`
	FilesChanged int    `json:"files_changed"`
	PRURL        string `json:"pr_url,omitempty"`
	Pushed       bool   `json:"pushed"`
	Skipped      string `json:"skipped,omitempty"`
}

func (s *PrepSubsystem) registerMirrorTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_mirror",
		Description: "Sync Forge repos to GitHub mirrors. Pushes Forge main to GitHub dev branch and creates a PR. Respects file count limits for CodeRabbit review.",
	}, s.mirror)
}

func (s *PrepSubsystem) mirror(ctx context.Context, _ *mcp.CallToolRequest, input MirrorInput) (*mcp.CallToolResult, MirrorOutput, error) {
	maxFiles := input.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 50
	}
	process := s.Core().Process()

	basePath := s.codePath
	if basePath == "" {
		basePath = core.JoinPath(HomeDir(), "Code", "core")
	} else {
		basePath = core.JoinPath(basePath, "core")
	}

	var repos []string
	if input.Repo != "" {
		repos = []string{input.Repo}
	} else {
		repos = s.listLocalRepos(basePath)
	}

	var synced []MirrorSync
	var skipped []string

	for _, repo := range repos {
		repoDir := core.JoinPath(basePath, repo)

		if !s.hasRemote(repoDir, "github") {
			skipped = append(skipped, core.Concat(repo, ": no github remote"))
			continue
		}

		process.RunIn(ctx, repoDir, "git", "fetch", "github")

		localBase := s.DefaultBranch(repoDir)
		ahead := s.commitsAhead(repoDir, "github/main", localBase)
		if ahead == 0 {
			continue
		}

		files := s.filesChanged(repoDir, "github/main", localBase)

		sync := MirrorSync{
			Repo:         repo,
			CommitsAhead: ahead,
			FilesChanged: files,
		}

		if files > maxFiles {
			sync.Skipped = core.Sprintf("%d files exceeds limit of %d", files, maxFiles)
			synced = append(synced, sync)
			continue
		}

		if input.DryRun {
			sync.Skipped = "dry run"
			synced = append(synced, sync)
			continue
		}

		s.ensureDevBranch(repoDir)

		base := s.DefaultBranch(repoDir)
		if r := process.RunIn(ctx, repoDir, "git", "push", "github", core.Concat(base, ":refs/heads/dev"), "--force"); !r.OK {
			sync.Skipped = core.Sprintf("push failed: %s", r.Value)
			synced = append(synced, sync)
			continue
		}
		sync.Pushed = true

		pullRequestURL, err := s.createGitHubPR(ctx, repoDir, repo, ahead, files)
		if err != nil {
			sync.Skipped = core.Sprintf("PR creation failed: %v", err)
		} else {
			sync.PRURL = pullRequestURL
		}

		synced = append(synced, sync)
	}

	return nil, MirrorOutput{
		Success: true,
		Synced:  synced,
		Skipped: skipped,
		Count:   len(synced),
	}, nil
}

// url, err := s.createGitHubPR(ctx, repoDir, "go-io", 3, 12)
func (s *PrepSubsystem) createGitHubPR(ctx context.Context, repoDir, repo string, commits, files int) (string, error) {
	ghRepo := core.Sprintf("%s/%s", GitHubOrg(), repo)
	process := s.Core().Process()

	r := process.RunIn(ctx, repoDir, "gh", "pr", "list", "--repo", ghRepo, "--head", "dev", "--state", "open", "--json", "url", "--limit", "1")
	if r.OK {
		out := r.Value.(string)
		if core.Contains(out, "url") {
			if url := extractJSONField(out, "url"); url != "" {
				return url, nil
			}
		}
	}

	body := core.Sprintf(
		"## Forge → GitHub Sync\n\n**Commits:** %d\n**Files changed:** %d\n\nAutomated sync from Forge (forge.lthn.ai) to GitHub mirror.\nReview with CodeRabbit before merging.\n\n---\nCo-Authored-By: Virgil <virgil@lethean.io>",
		commits, files)

	title := core.Sprintf("[sync] %s: %d commits, %d files", repo, commits, files)

	r = process.RunIn(ctx, repoDir, "gh", "pr", "create",
		"--repo", ghRepo, "--head", "dev", "--base", "main",
		"--title", title, "--body", body)
	if !r.OK {
		return "", core.E("createGitHubPR", r.Value.(string), nil)
	}

	prOut := r.Value.(string)
	lines := core.Split(core.Trim(prOut), "\n")
	if len(lines) > 0 {
		return lines[len(lines)-1], nil
	}
	return "", nil
}

func (s *PrepSubsystem) ensureDevBranch(repoDir string) {
	s.Core().Process().RunIn(context.Background(), repoDir, "git", "push", "github", "HEAD:refs/heads/dev")
}

func (s *PrepSubsystem) hasRemote(repoDir, name string) bool {
	return s.Core().Process().RunIn(context.Background(), repoDir, "git", "remote", "get-url", name).OK
}

func (s *PrepSubsystem) commitsAhead(repoDir, base, head string) int {
	r := s.Core().Process().RunIn(context.Background(), repoDir, "git", "rev-list", core.Concat(base, "..", head), "--count")
	if !r.OK {
		return 0
	}
	out := core.Trim(r.Value.(string))
	return parseInt(out)
}

func (s *PrepSubsystem) filesChanged(repoDir, base, head string) int {
	r := s.Core().Process().RunIn(context.Background(), repoDir, "git", "diff", "--name-only", core.Concat(base, "..", head))
	if !r.OK {
		return 0
	}
	out := core.Trim(r.Value.(string))
	if out == "" {
		return 0
	}
	return len(core.Split(out, "\n"))
}

func (s *PrepSubsystem) listLocalRepos(basePath string) []string {
	paths := core.PathGlob(core.JoinPath(basePath, "*"))
	var repos []string
	for _, p := range paths {
		name := core.PathBase(p)
		if !fs.IsDir(p) {
			continue
		}
		if fs.IsDir(core.JoinPath(basePath, name, ".git")) {
			repos = append(repos, name)
		}
	}
	return repos
}

func extractJSONField(jsonStr, field string) string {
	if jsonStr == "" || field == "" {
		return ""
	}

	var list []map[string]any
	if r := core.JSONUnmarshalString(jsonStr, &list); r.OK {
		for _, item := range list {
			if value, ok := item[field].(string); ok {
				return value
			}
		}
	}

	var item map[string]any
	if r := core.JSONUnmarshalString(jsonStr, &item); !r.OK {
		return ""
	}

	value, _ := item[field].(string)
	return value
}
