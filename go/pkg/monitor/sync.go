// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"forge.lthn.ai/core/agent/agentic"
	coreio "forge.lthn.ai/core/go-io"
)

// CheckinResponse is what the API returns for an agent checkin.
type CheckinResponse struct {
	// Repos that have new commits since the agent's last checkin.
	Changed []ChangedRepo `json:"changed,omitempty"`
	// Server timestamp — use as "since" on next checkin.
	Timestamp int64 `json:"timestamp"`
}

// ChangedRepo is a repo that has new commits.
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

	url := fmt.Sprintf("%s/v1/agent/checkin?agent=%s&since=%d", apiURL, agentName, m.lastSyncTimestamp)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	// Use brain key for auth
	brainKey := os.Getenv("CORE_BRAIN_KEY")
	if brainKey == "" {
		home, _ := os.UserHomeDir()
		if data, err := coreio.Local.Read(filepath.Join(home, ".claude", "brain.key")); err == nil {
			brainKey = strings.TrimSpace(data)
		}
	}
	if brainKey != "" {
		req.Header.Set("Authorization", "Bearer "+brainKey)
	}

	resp, err := http.DefaultClient.Do(req)
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

	// Update timestamp for next checkin
	m.mu.Lock()
	m.lastSyncTimestamp = checkin.Timestamp
	m.mu.Unlock()

	if len(checkin.Changed) == 0 {
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
		repoDir := filepath.Join(basePath, repo.Repo)
		if _, err := os.Stat(repoDir); err != nil {
			continue
		}

		// Check if we're already on main and clean
		branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		branchCmd.Dir = repoDir
		branch, err := branchCmd.Output()
		if err != nil || strings.TrimSpace(string(branch)) != "main" {
			continue // Don't pull if not on main
		}

		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = repoDir
		status, _ := statusCmd.Output()
		if len(strings.TrimSpace(string(status))) > 0 {
			continue // Don't pull if dirty
		}

		// Fast-forward pull
		pullCmd := exec.Command("git", "pull", "--ff-only", "origin", "main")
		pullCmd.Dir = repoDir
		if pullCmd.Run() == nil {
			pulled = append(pulled, repo.Repo)
		}
	}

	if len(pulled) == 0 {
		return ""
	}

	return fmt.Sprintf("Synced %d repo(s): %s", len(pulled), strings.Join(pulled, ", "))
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
