// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
)

// CheckinResponse is what the API returns for an agent checkin.
//
//	resp := monitor.CheckinResponse{Changed: []monitor.ChangedRepo{{Repo: "core-agent", Branch: "main", SHA: "abc123"}}, Timestamp: 1712345678}
type CheckinResponse struct {
	// Repos that have new commits since the agent's last checkin.
	Changed []ChangedRepo `json:"changed,omitempty"`
	// Server timestamp — use as "since" on next checkin.
	Timestamp int64 `json:"timestamp"`
}

// ChangedRepo is a repo that has new commits.
//
//	repo := monitor.ChangedRepo{Repo: "core-agent", Branch: "main", SHA: "abc123"}
type ChangedRepo struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	SHA    string `json:"sha"`
}

// syncRepos calls the checkin API and pulls any repos that changed.
// Returns a human-readable message if repos were updated, empty string otherwise.
func (m *Subsystem) syncRepos() string {
	agentName := agentic.AgentName()
	checkinURL := core.Sprintf("%s/v1/agent/checkin?agent=%s&since=%d", monitorAPIURL(), core.Replace(agentName, " ", "%20"), m.lastSyncTimestamp)

	brainKey := monitorBrainKey()
	hr := agentic.HTTPGet(context.Background(), checkinURL, brainKey, "Bearer")
	if !hr.OK {
		return ""
	}

	var checkin CheckinResponse
	if r := core.JSONUnmarshalString(hr.Value.(string), &checkin); !r.OK {
		return ""
	}

	if len(checkin.Changed) == 0 {
		// No changes — safe to advance timestamp
		m.mu.Lock()
		m.lastSyncTimestamp = checkin.Timestamp
		m.mu.Unlock()
		return ""
	}

	// Pull changed repos
	basePath := core.Env("CODE_PATH")
	if basePath == "" {
		basePath = core.JoinPath(agentic.HomeDir(), "Code", "core")
	}

	var pulled []string
	for _, repo := range checkin.Changed {
		// Sanitise repo name to prevent path traversal from API response
		repoName := core.PathBase(core.Replace(repo.Repo, "\\", "/"))
		if repoName == "." || repoName == ".." || repoName == "" {
			continue
		}
		repoDir := core.JoinPath(basePath, repoName)
		if !fs.Exists(repoDir) || fs.IsFile(repoDir) {
			continue
		}

		// Check if on the default branch and clean
		current := m.gitOutput(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
		if current == "" {
			continue
		}

		// Determine which branch to pull — use server-reported branch,
		// fall back to current if server didn't specify
		targetBranch := repo.Branch
		if targetBranch == "" {
			targetBranch = current
		}

		// Only pull if we're on the target branch (or it's a default branch)
		if current != targetBranch {
			continue // On a different branch — skip
		}

		status := m.gitOutput(repoDir, "status", "--porcelain")
		if len(status) > 0 {
			continue // Don't pull if dirty
		}

		// Fast-forward pull the target branch
		if m.gitOK(repoDir, "pull", "--ff-only", "origin", targetBranch) {
			pulled = append(pulled, repo.Repo)
		}
	}

	// Only advance timestamp if we handled all reported repos.
	// If any were skipped (dirty, wrong branch, missing), keep the
	// old timestamp so the server reports them again next cycle.
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

// lastSyncTimestamp is stored on the subsystem — add it via the check cycle.
// Initialised to "now" on first run so we don't pull everything on startup.
func (m *Subsystem) initSyncTimestamp() {
	m.mu.Lock()
	if m.lastSyncTimestamp == 0 {
		m.lastSyncTimestamp = time.Now().Unix()
	}
	m.mu.Unlock()
}
