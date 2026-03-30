// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"net/url"
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
	checkinURL := core.Sprintf("%s/v1/agent/checkin?agent=%s&since=%d", monitorAPIURL(), url.QueryEscape(agentName), m.lastSyncTimestamp)

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
		m.mu.Lock()
		m.lastSyncTimestamp = checkin.Timestamp
		m.mu.Unlock()
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
		m.mu.Lock()
		m.lastSyncTimestamp = checkin.Timestamp
		m.mu.Unlock()
	}

	if len(pulled) == 0 {
		return ""
	}

	return core.Sprintf("Synced %d repo(s): %s", len(pulled), core.Join(", ", pulled...))
}

func (m *Subsystem) initSyncTimestamp() {
	m.mu.Lock()
	if m.lastSyncTimestamp == 0 {
		m.lastSyncTimestamp = time.Now().Unix()
	}
	m.mu.Unlock()
}
