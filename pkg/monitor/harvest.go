// SPDX-License-Identifier: EUPL-1.2

// result := m.harvestWorkspace("/srv/.core/workspace/core/go-io/task-5")
// if result != nil && result.rejected == "" { core.Print(nil, "%s", result.repo) }

package monitor

import (
	"context"
	"strconv"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
)

type harvestResult struct {
	repo     string
	branch   string
	files    int
	rejected string
}

// summary := m.harvestCompleted()
// if summary != "" { core.Print(nil, summary) }
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

// result := m.harvestWorkspace("/srv/.core/workspace/core/go-io/task-5")
// if result != nil && result.rejected == "" { core.Print(nil, "%s", result.repo) }
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

	if workspaceStatus.Status != "completed" {
		return nil
	}

	repoDir := agentic.WorkspaceRepoDir(workspaceDir)
	if !fs.IsDir(repoDir) {
		return nil
	}

	branch := workspaceStatus.Branch
	if branch == "" {
		branch = m.detectBranch(repoDir)
	}
	base := m.defaultBranch(repoDir)
	if branch == "" || branch == base {
		return nil
	}

	unpushed := m.countUnpushed(repoDir, branch)
	if unpushed == 0 {
		return nil
	}

	if reason := m.checkSafety(repoDir); reason != "" {
		updateStatus(workspaceDir, "rejected", reason)
		return &harvestResult{repo: workspaceStatus.Repo, branch: branch, rejected: reason}
	}

	files := m.countChangedFiles(repoDir)

	updateStatus(workspaceDir, "ready-for-review", "")

	return &harvestResult{repo: workspaceStatus.Repo, branch: branch, files: files}
}

// output := m.gitOutput("/srv/.core/workspace/core/go-io/task-5/repo", "log", "--oneline")
func (m *Subsystem) gitOutput(repoDir string, args ...string) string {
	processResult := m.Core().Process().RunIn(context.Background(), repoDir, "git", args...)
	if !processResult.OK {
		return ""
	}
	return core.Trim(processResult.Value.(string))
}

// ok := m.gitOK("/srv/.core/workspace/core/go-io/task-5/repo", "rev-parse", "--verify", "main")
func (m *Subsystem) gitOK(repoDir string, args ...string) bool {
	return m.Core().Process().RunIn(context.Background(), repoDir, "git", args...).OK
}

// branch := m.detectBranch("/srv/.core/workspace/core/go-io/task-5/repo")
func (m *Subsystem) detectBranch(repoDir string) string {
	return m.gitOutput(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
}

// base := m.defaultBranch("/srv/.core/workspace/core/go-io/task-5/repo")
func (m *Subsystem) defaultBranch(repoDir string) string {
	if ref := m.gitOutput(repoDir, "symbolic-ref", "refs/remotes/origin/HEAD", "--short"); ref != "" {
		if core.HasPrefix(ref, "origin/") {
			return core.TrimPrefix(ref, "origin/")
		}
		return ref
	}
	for _, branch := range []string{"main", "master"} {
		if m.gitOK(repoDir, "rev-parse", "--verify", branch) {
			return branch
		}
	}
	return "main"
}

// ahead := m.countUnpushed("/srv/.core/workspace/core/go-io/task-5/repo", "feature/ax-cleanup")
func (m *Subsystem) countUnpushed(repoDir, branch string) int {
	base := m.defaultBranch(repoDir)
	out := m.gitOutput(repoDir, "rev-list", "--count", core.Concat("origin/", base, "..", branch))
	if out == "" {
		out2 := m.gitOutput(repoDir, "log", "--oneline", core.Concat(base, "..", branch))
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

// reason := m.checkSafety("/srv/.core/workspace/core/go-io/task-5/repo")
func (m *Subsystem) checkSafety(repoDir string) string {
	base := m.defaultBranch(repoDir)
	out := m.gitOutput(repoDir, "diff", "--name-only", core.Concat(base, "...HEAD"))
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

		fullPath := core.JoinPath(repoDir, file)
		if stat := fs.Stat(fullPath); stat.OK {
			if info, ok := stat.Value.(interface{ Size() int64 }); ok && info.Size() > 1024*1024 {
				return core.Sprintf("large file: %s (%d bytes)", file, info.Size())
			}
		}
	}

	return ""
}

// files := m.countChangedFiles("/srv/.core/workspace/core/go-io/task-5/repo")
func (m *Subsystem) countChangedFiles(repoDir string) int {
	base := m.defaultBranch(repoDir)
	out := m.gitOutput(repoDir, "diff", "--name-only", core.Concat(base, "...HEAD"))
	if out == "" {
		return 0
	}
	lines := core.Split(out, "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// _ = m.pushBranch("/srv/.core/workspace/core/go-io/task-5/repo", "feature/ax-cleanup")
func (m *Subsystem) pushBranch(repoDir, branch string) error {
	processResult := m.Core().Process().RunIn(context.Background(), repoDir, "git", "push", "origin", branch)
	if !processResult.OK {
		if err, ok := processResult.Value.(error); ok {
			return core.E("harvest.pushBranch", "push failed", err)
		}
		return core.E("harvest.pushBranch", "push failed", nil)
	}
	return nil
}

// updateStatus(workspaceDir, "ready-for-review", "")
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
		delete(workspaceStatus, "question")
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
