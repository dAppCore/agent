// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
)

// resp := monitor.CheckinResponse{Changed: []monitor.ChangedRepo{{Repo: "core-agent", Branch: "main", SHA: "abc123"}}, Timestamp: 1712345678}
type CheckinResponse struct {
	Changed   []ChangedRepo `json:"changed,omitempty"`
	Timestamp int64         `json:"timestamp"`
}

// repo := monitor.ChangedRepo{Repo: "core-agent", Branch: "main", SHA: "abc123"}
type ChangedRepo struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	SHA    string `json:"sha"`
}

func (m *Subsystem) syncRepos() string {
	agentName := agentic.AgentName()
	checkinURL := core.Sprintf("%s/v1/agent/checkin?agent=%s&since=%d", monitorAPIURL(), core.URLEncode(agentName), m.lastSyncTimestamp)

	brainKey := monitorBrainKey()
	httpResult := agentic.HTTPGet(context.Background(), checkinURL, brainKey, "Bearer")
	if !httpResult.OK {
		return ""
	}

	var checkin CheckinResponse
	if parseResult := core.JSONUnmarshalString(httpResult.Value.(string), &checkin); !parseResult.OK {
		return ""
	}

	if len(checkin.Changed) == 0 {
		unlock := m.monitorLock()
		m.lastSyncTimestamp = checkin.Timestamp
		unlock()
		return ""
	}

	basePath := core.Env("CODE_PATH")
	if basePath == "" {
		basePath = core.JoinPath(agentic.HomeDir(), "Code", "core")
	}

	var pulled []string
	for _, repo := range checkin.Changed {
		repoName := core.PathBase(core.Replace(repo.Repo, "\\", "/"))
		if repoName == "." || repoName == ".." || repoName == "" {
			continue
		}
		repoDir := core.JoinPath(basePath, repoName)
		if !fs.Exists(repoDir) || fs.IsFile(repoDir) {
			continue
		}

		current := m.gitOutput(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
		if current == "" {
			continue
		}

		targetBranch := repo.Branch
		if targetBranch == "" {
			targetBranch = current
		}

		if current != targetBranch {
			continue
		}

		status := m.gitOutput(repoDir, "status", "--porcelain")
		if len(status) > 0 {
			continue
		}

		if m.gitOK(repoDir, "pull", "--ff-only", "origin", targetBranch) {
			pulled = append(pulled, repo.Repo)
		}
	}

	skipped := len(checkin.Changed) - len(pulled)
	if skipped == 0 {
		unlock := m.monitorLock()
		m.lastSyncTimestamp = checkin.Timestamp
		unlock()
	}

	if len(pulled) == 0 {
		return ""
	}

	return core.Sprintf("Synced %d repo(s): %s", len(pulled), core.Join(", ", pulled...))
}

func (m *Subsystem) syncWorkspacePush(repo, branch, org string) bool {
	if m.ServiceRuntime == nil {
		return false
	}

	repoDir := localRepoDir(org, repo)
	if repoDir == "" || !fs.Exists(repoDir) || fs.IsFile(repoDir) {
		return false
	}

	targetBranch := core.Trim(branch)
	if targetBranch == "" {
		targetBranch = m.detectBranch(repoDir)
	}
	if targetBranch == "" {
		targetBranch = m.defaultBranch(repoDir)
	}
	if targetBranch == "" {
		return false
	}

	if !m.gitOK(repoDir, "fetch", "origin", targetBranch) {
		return false
	}

	currentBranch := m.detectBranch(repoDir)
	if currentBranch != "" && currentBranch != targetBranch {
		if !m.gitOK(repoDir, "checkout", "-B", targetBranch, core.Concat("origin/", targetBranch)) {
			return false
		}
	}

	return m.gitOK(repoDir, "reset", "--hard", core.Concat("origin/", targetBranch))
}

func localRepoDir(org, repo string) string {
	basePath := core.Env("CODE_PATH")
	if basePath == "" {
		basePath = core.JoinPath(agentic.HomeDir(), "Code")
	}

	normalisedRepo := core.Replace(repo, "\\", "/")
	repoName := core.PathBase(normalisedRepo)
	orgName := core.PathBase(core.Replace(org, "\\", "/"))
	repoParts := core.Split(normalisedRepo, "/")
	if orgName == "" && len(repoParts) > 1 {
		orgName = repoParts[0]
	}

	candidates := []string{}
	if orgName != "" {
		candidates = append(candidates, core.JoinPath(basePath, orgName, repoName))
	}
	candidates = append(candidates, core.JoinPath(basePath, repoName))

	for _, candidate := range candidates {
		if fs.Exists(candidate) && !fs.IsFile(candidate) {
			return candidate
		}
	}

	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func (m *Subsystem) initSyncTimestamp() {
	unlock := m.monitorLock()
	if m.lastSyncTimestamp == 0 {
		m.lastSyncTimestamp = time.Now().Unix()
	}
	unlock()
}
