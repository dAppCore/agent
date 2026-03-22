// SPDX-License-Identifier: EUPL-1.2

// Harvest completed agent workspaces — push changes back to source repos.
//
// After an agent completes, its commits live in the workspace clone.
// This code pushes the agent's branch to the source repo so the
// changes are available for review. It checks for binaries and
// large files before pushing.

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dappco.re/go/agent/pkg/agentic"
	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
)

// harvestResult tracks what happened during harvest.
type harvestResult struct {
	repo     string
	branch   string
	files    int
	rejected string // non-empty if rejected (binary, too large, etc.)
}

// harvestCompleted scans for completed workspaces and pushes their
// branches back to the source repos. Returns a summary message.
func (m *Subsystem) harvestCompleted() string {
	wsRoot := agentic.WorkspaceRoot()
	entries, err := filepath.Glob(filepath.Join(wsRoot, "*/status.json"))
	if err != nil {
		return ""
	}

	var harvested []harvestResult

	for _, entry := range entries {
		wsDir := filepath.Dir(entry)
		result := m.harvestWorkspace(wsDir)
		if result != nil {
			harvested = append(harvested, *result)
		}
	}

	if len(harvested) == 0 {
		return ""
	}

	var parts []string
	for _, h := range harvested {
		if h.rejected != "" {
			parts = append(parts, fmt.Sprintf("%s: REJECTED (%s)", h.repo, h.rejected))
			if m.notifier != nil {
				m.notifier.ChannelSend(context.Background(), "harvest.rejected", map[string]any{
					"repo":   h.repo,
					"branch": h.branch,
					"reason": h.rejected,
				})
			}
		} else {
			parts = append(parts, fmt.Sprintf("%s: ready-for-review %s (%d files)", h.repo, h.branch, h.files))
			if m.notifier != nil {
				m.notifier.ChannelSend(context.Background(), "harvest.complete", map[string]any{
					"repo":   h.repo,
					"branch": h.branch,
					"files":  h.files,
				})
			}
		}
	}
	return "Harvested: " + strings.Join(parts, ", ")
}

// harvestWorkspace checks a single workspace and pushes if ready.
func (m *Subsystem) harvestWorkspace(wsDir string) *harvestResult {
	data, err := coreio.Local.Read(filepath.Join(wsDir, "status.json"))
	if err != nil {
		return nil
	}

	var st struct {
		Status string `json:"status"`
		Repo   string `json:"repo"`
		Branch string `json:"branch"`
	}
	if json.Unmarshal([]byte(data), &st) != nil {
		return nil
	}

	// Only harvest completed workspaces (not merged, running, etc.)
	if st.Status != "completed" {
		return nil
	}

	srcDir := filepath.Join(wsDir, "src")
	if _, err := os.Stat(srcDir); err != nil {
		return nil
	}

	// Check if there are commits to push
	branch := st.Branch
	if branch == "" {
		branch = detectBranch(srcDir)
	}
	base := defaultBranch(srcDir)
	if branch == "" || branch == base {
		return nil
	}

	// Check for unpushed commits
	unpushed := countUnpushed(srcDir, branch)
	if unpushed == 0 {
		return nil // already pushed or no commits
	}

	// Safety checks before pushing
	if reason := checkSafety(srcDir); reason != "" {
		updateStatus(wsDir, "rejected", reason)
		return &harvestResult{repo: st.Repo, branch: branch, rejected: reason}
	}

	// Count changed files
	files := countChangedFiles(srcDir)

	// Mark ready for review — do NOT auto-push.
	// Pushing is a high-impact mutation that should happen during
	// explicit review (/review command), not silently in the background.
	updateStatus(wsDir, "ready-for-review", "")

	return &harvestResult{repo: st.Repo, branch: branch, files: files}
}

// detectBranch returns the current branch name.
func detectBranch(srcDir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = srcDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// defaultBranch detects the default branch of the repo (main, master, etc.).
func defaultBranch(srcDir string) string {
	// Try origin/HEAD first
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short")
	cmd.Dir = srcDir
	if out, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(out))
		// returns "origin/main" — strip prefix
		if strings.HasPrefix(ref, "origin/") {
			return strings.TrimPrefix(ref, "origin/")
		}
		return ref
	}
	// Fallback: check if main exists, else master
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = srcDir
		if cmd.Run() == nil {
			return branch
		}
	}
	return "main"
}

// countUnpushed returns the number of commits ahead of origin's default branch.
func countUnpushed(srcDir, branch string) int {
	base := defaultBranch(srcDir)
	cmd := exec.Command("git", "rev-list", "--count", "origin/"+base+".."+branch)
	cmd.Dir = srcDir
	out, err := cmd.Output()
	if err != nil {
		cmd2 := exec.Command("git", "log", "--oneline", base+".."+branch)
		cmd2.Dir = srcDir
		out2, err2 := cmd2.Output()
		if err2 != nil {
			return 0
		}
		lines := strings.Split(strings.TrimSpace(string(out2)), "\n")
		if len(lines) == 1 && lines[0] == "" {
			return 0
		}
		return len(lines)
	}
	var count int
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &count)
	return count
}

// checkSafety rejects workspaces with binaries or oversized files.
// Checks ALL changed files (added, modified, renamed), not just new.
// Fails closed: if git diff fails, rejects the workspace.
func checkSafety(srcDir string) string {
	// Check all changed files — added, modified, renamed
	base := defaultBranch(srcDir)
	cmd := exec.Command("git", "diff", "--name-only", base+"...HEAD")
	cmd.Dir = srcDir
	out, err := cmd.Output()
	if err != nil {
		return "safety check failed: git diff error"
	}

	binaryExts := map[string]bool{
		".exe": true, ".bin": true, ".so": true, ".dylib": true,
		".dll": true, ".o": true, ".a": true, ".pyc": true,
		".class": true, ".jar": true, ".war": true,
		".zip": true, ".tar": true, ".gz": true, ".bz2": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".db": true, ".sqlite": true, ".sqlite3": true,
	}

	for _, file := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if file == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(file))
		if binaryExts[ext] {
			return fmt.Sprintf("binary file added: %s", file)
		}

		// Check file size (reject > 1MB)
		fullPath := filepath.Join(srcDir, file)
		info, err := os.Stat(fullPath)
		if err == nil && info.Size() > 1024*1024 {
			return fmt.Sprintf("large file: %s (%d bytes)", file, info.Size())
		}
	}

	return ""
}

// countChangedFiles returns the number of files changed vs the default branch.
func countChangedFiles(srcDir string) int {
	base := defaultBranch(srcDir)
	cmd := exec.Command("git", "diff", "--name-only", base+"...HEAD")
	cmd.Dir = srcDir
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// pushBranch pushes the agent's branch to origin.
func pushBranch(srcDir, branch string) error {
	cmd := exec.Command("git", "push", "origin", branch)
	cmd.Dir = srcDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return coreerr.E("harvest.pushBranch", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// updateStatus updates the workspace status.json.
func updateStatus(wsDir, status, question string) {
	data, err := coreio.Local.Read(filepath.Join(wsDir, "status.json"))
	if err != nil {
		return
	}
	var st map[string]any
	if json.Unmarshal([]byte(data), &st) != nil {
		return
	}
	st["status"] = status
	if question != "" {
		st["question"] = question
	} else {
		delete(st, "question") // clear stale question from previous state
	}
	updated, _ := json.MarshalIndent(st, "", "  ")
	coreio.Local.Write(filepath.Join(wsDir, "status.json"), string(updated))
}
