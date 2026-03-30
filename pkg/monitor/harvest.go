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
	"strconv"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
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
	var harvested []harvestResult

	for _, entry := range agentic.WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(entry)
		result := m.harvestWorkspace(workspaceDir)
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
			parts = append(parts, core.Sprintf("%s: REJECTED (%s)", h.repo, h.rejected))
			if m.ServiceRuntime != nil {
				m.Core().ACTION(messages.HarvestRejected{Repo: h.repo, Branch: h.branch, Reason: h.rejected})
			}
		} else {
			parts = append(parts, core.Sprintf("%s: ready-for-review %s (%d files)", h.repo, h.branch, h.files))
			if m.ServiceRuntime != nil {
				m.Core().ACTION(messages.HarvestComplete{Repo: h.repo, Branch: h.branch, Files: h.files})
			}
		}
	}
	return core.Concat("Harvested: ", core.Join(", ", parts...))
}

// harvestWorkspace checks a single workspace and pushes if ready.
func (m *Subsystem) harvestWorkspace(workspaceDir string) *harvestResult {
	statusResult := fs.Read(agentic.WorkspaceStatusPath(workspaceDir))
	if !statusResult.OK {
		return nil
	}
	statusData, ok := resultString(statusResult)
	if !ok {
		return nil
	}

	var workspaceStatus struct {
		Status string `json:"status"`
		Repo   string `json:"repo"`
		Branch string `json:"branch"`
	}
	if parseResult := core.JSONUnmarshalString(statusData, &workspaceStatus); !parseResult.OK {
		return nil
	}

	// Only harvest completed workspaces (not merged, running, etc.)
	if workspaceStatus.Status != "completed" {
		return nil
	}

	repoDir := agentic.WorkspaceRepoDir(workspaceDir)
	if !fs.IsDir(repoDir) {
		return nil
	}

	// Check if there are commits to push
	branch := workspaceStatus.Branch
	if branch == "" {
		branch = m.detectBranch(repoDir)
	}
	base := m.defaultBranch(repoDir)
	if branch == "" || branch == base {
		return nil
	}

	// Check for unpushed commits
	unpushed := m.countUnpushed(repoDir, branch)
	if unpushed == 0 {
		return nil // already pushed or no commits
	}

	// Safety checks before pushing
	if reason := m.checkSafety(repoDir); reason != "" {
		updateStatus(workspaceDir, "rejected", reason)
		return &harvestResult{repo: workspaceStatus.Repo, branch: branch, rejected: reason}
	}

	// Count changed files
	files := m.countChangedFiles(repoDir)

	// Mark ready for review — do NOT auto-push.
	// Pushing is a high-impact mutation that should happen during
	// explicit review (/review command), not silently in the background.
	updateStatus(workspaceDir, "ready-for-review", "")

	return &harvestResult{repo: workspaceStatus.Repo, branch: branch, files: files}
}

// gitOutput runs a git command and returns trimmed stdout via Core Process.
func (m *Subsystem) gitOutput(dir string, args ...string) string {
	processResult := m.Core().Process().RunIn(context.Background(), dir, "git", args...)
	if !processResult.OK {
		return ""
	}
	return core.Trim(processResult.Value.(string))
}

// gitOK runs a git command and returns true if it exits 0.
func (m *Subsystem) gitOK(dir string, args ...string) bool {
	return m.Core().Process().RunIn(context.Background(), dir, "git", args...).OK
}

// detectBranch returns the current branch name.
func (m *Subsystem) detectBranch(srcDir string) string {
	return m.gitOutput(srcDir, "rev-parse", "--abbrev-ref", "HEAD")
}

// defaultBranch detects the default branch of the repo (main, master, etc.).
func (m *Subsystem) defaultBranch(srcDir string) string {
	if ref := m.gitOutput(srcDir, "symbolic-ref", "refs/remotes/origin/HEAD", "--short"); ref != "" {
		if core.HasPrefix(ref, "origin/") {
			return core.TrimPrefix(ref, "origin/")
		}
		return ref
	}
	for _, branch := range []string{"main", "master"} {
		if m.gitOK(srcDir, "rev-parse", "--verify", branch) {
			return branch
		}
	}
	return "main"
}

// countUnpushed returns the number of commits ahead of origin's default branch.
func (m *Subsystem) countUnpushed(srcDir, branch string) int {
	base := m.defaultBranch(srcDir)
	out := m.gitOutput(srcDir, "rev-list", "--count", core.Concat("origin/", base, "..", branch))
	if out == "" {
		// Fallback
		out2 := m.gitOutput(srcDir, "log", "--oneline", core.Concat(base, "..", branch))
		if out2 == "" {
			return 0
		}
		lines := core.Split(out2, "\n")
		if len(lines) == 1 && lines[0] == "" {
			return 0
		}
		return len(lines)
	}
	count, err := strconv.Atoi(out)
	if err != nil {
		return 0
	}
	return count
}

// checkSafety rejects workspaces with binaries or oversized files.
func (m *Subsystem) checkSafety(srcDir string) string {
	base := m.defaultBranch(srcDir)
	out := m.gitOutput(srcDir, "diff", "--name-only", core.Concat(base, "...HEAD"))
	if out == "" {
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

	for _, file := range core.Split(out, "\n") {
		if file == "" {
			continue
		}
		ext := core.Lower(core.PathExt(file))
		if binaryExts[ext] {
			return core.Sprintf("binary file added: %s", file)
		}

		fullPath := core.JoinPath(srcDir, file)
		if stat := fs.Stat(fullPath); stat.OK {
			if info, ok := stat.Value.(interface{ Size() int64 }); ok && info.Size() > 1024*1024 {
				return core.Sprintf("large file: %s (%d bytes)", file, info.Size())
			}
		}
	}

	return ""
}

// countChangedFiles returns the number of files changed vs the default branch.
func (m *Subsystem) countChangedFiles(srcDir string) int {
	base := m.defaultBranch(srcDir)
	out := m.gitOutput(srcDir, "diff", "--name-only", core.Concat(base, "...HEAD"))
	if out == "" {
		return 0
	}
	lines := core.Split(out, "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// pushBranch pushes the agent's branch to origin.
func (m *Subsystem) pushBranch(srcDir, branch string) error {
	processResult := m.Core().Process().RunIn(context.Background(), srcDir, "git", "push", "origin", branch)
	if !processResult.OK {
		if err, ok := processResult.Value.(error); ok {
			return core.E("harvest.pushBranch", "push failed", err)
		}
		return core.E("harvest.pushBranch", "push failed", nil)
	}
	return nil
}

// updateStatus rewrites status.json after a harvest decision.
//
//	updateStatus(workspaceDir, "ready-for-review", "")
func updateStatus(workspaceDir, status, question string) {
	statusResult := fs.Read(agentic.WorkspaceStatusPath(workspaceDir))
	if !statusResult.OK {
		return
	}
	statusData, ok := resultString(statusResult)
	if !ok {
		return
	}
	var workspaceStatus map[string]any
	if parseResult := core.JSONUnmarshalString(statusData, &workspaceStatus); !parseResult.OK {
		return
	}
	workspaceStatus["status"] = status
	if question != "" {
		workspaceStatus["question"] = question
	} else {
		delete(workspaceStatus, "question") // clear stale question from previous state
	}
	statusPath := agentic.WorkspaceStatusPath(workspaceDir)
	if writeResult := fs.WriteAtomic(statusPath, core.JSONMarshalString(workspaceStatus)); !writeResult.OK {
		if err, ok := writeResult.Value.(error); ok {
			core.Warn("monitor.updateStatus: failed to write status", "path", statusPath, "reason", err)
			return
		}
		core.Warn("monitor.updateStatus: failed to write status", "path", statusPath)
	}
}
