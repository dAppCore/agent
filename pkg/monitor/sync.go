// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"encoding/json"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
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
	apiURL := os.Getenv("CORE_API_URL")
	if apiURL == "" {
		apiURL = "https://api.lthn.sh"
	}

	agentName := agentic.AgentName()

	checkinURL := core.Sprintf("%s/v1/agent/checkin?agent=%s&since=%d", apiURL, neturl.QueryEscape(agentName), m.lastSyncTimestamp)

	req, err := http.NewRequest("GET", checkinURL, nil)
	if err != nil {
		return ""
	}

	// Use brain key for auth
	brainKey := os.Getenv("CORE_BRAIN_KEY")
	if brainKey == "" {
		home, _ := os.UserHomeDir()
		if r := fs.Read(brainKeyPath(home)); r.OK {
			if value, ok := resultString(r); ok {
				brainKey = core.Trim(value)
			}
		}
	}
	if brainKey != "" {
		req.Header.Set("Authorization", core.Concat("Bearer ", brainKey))
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return ""
	}

	var checkin CheckinResponse
	if json.NewDecoder(resp.Body).Decode(&checkin) != nil {
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
	basePath := os.Getenv("CODE_PATH")
	if basePath == "" {
		home, _ := os.UserHomeDir()
		basePath = filepath.Join(home, "Code", "core")
	}

	var pulled []string
	for _, repo := range checkin.Changed {
		// Sanitise repo name to prevent path traversal from API response
		repoName := filepath.Base(repo.Repo)
		if repoName == "." || repoName == ".." || repoName == "" {
			continue
		}
		repoDir := core.Concat(basePath, "/", repoName)
		if _, err := os.Stat(repoDir); err != nil {
			continue
		}

		// Check if on the default branch and clean
		branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		branchCmd.Dir = repoDir
		currentBranch, err := branchCmd.Output()
		if err != nil {
			continue
		}
		current := core.Trim(string(currentBranch))

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

		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = repoDir
		status, _ := statusCmd.Output()
		if len(core.Trim(string(status))) > 0 {
			continue // Don't pull if dirty
		}

		// Fast-forward pull the target branch
		pullCmd := exec.Command("git", "pull", "--ff-only", "origin", targetBranch)
		pullCmd.Dir = repoDir
		if pullCmd.Run() == nil {
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
