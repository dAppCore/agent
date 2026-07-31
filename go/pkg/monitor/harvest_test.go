// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
)

// initTestRepo creates a bare git repo and a workspace clone with a branch.
func initTestRepo(t *testing.T) (sourceDir, wsDir string) {
	t.Helper()

	// Create bare "source" repo
	sourceDir = core.JoinPath(t.TempDir(), "source")
	fs.EnsureDir(sourceDir)
	run(t, sourceDir, "git", "init")
	run(t, sourceDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(sourceDir, "README.md"), "# test")
	run(t, sourceDir, "git", "add", ".")
	run(t, sourceDir, "git", "commit", "-m", "init")

	// Create workspace dir with repo/ clone
	wsDir = core.JoinPath(t.TempDir(), "workspace")
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(wsDir)
	run(t, wsDir, "git", "clone", sourceDir, "repo")

	// Create agent branch with a commit
	run(t, repoDir, "git", "checkout", "-b", "agent/test-task")
	fs.Write(core.JoinPath(repoDir, "new.go"), "package main\n")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "agent work")

	return sourceDir, wsDir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	gitEnv := []string{"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test"}
	r := testMon.Core().Process().RunWithEnv(context.Background(), dir, gitEnv, name, args...)
	if !r.OK {
		t.Fatalf("command %s %v failed: %v", name, args, r.Value)
	}
}

func writeStatus(t *testing.T, wsDir, status, repo, branch string) {
	t.Helper()
	st := map[string]any{
		"status": status,
		"repo":   repo,
		"branch": branch,
	}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))
}

// --- Tests ---

func TestHarvest_DetectBranch_Good_Case(t *testing.T) {
	_, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	branch := testMon.detectBranch(repoDir)
	core.AssertEqual(t, "agent/test-task", branch)
}

func TestHarvest_DetectBranch_Bad_NoRepo(t *testing.T) {
	branch := testMon.detectBranch(t.TempDir())
	core.AssertEqual(t, "", branch)
	core.AssertEmpty(t, branch)
}

func TestHarvest_CountUnpushed_Good_Case(t *testing.T) {
	_, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	count := testMon.countUnpushed(repoDir, "agent/test-task")
	core.AssertEqual(t, 1, count)
}

func TestHarvest_CountChangedFiles_Good_Case(t *testing.T) {
	_, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	count := testMon.countChangedFiles(repoDir)
	core.AssertEqual(t, 1, count)
}

func TestHarvest_CheckSafety_Good_CleanWorkspace(t *testing.T) {
	_, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	reason := testMon.checkSafety(repoDir)
	core.AssertEqual(t, "", reason)
}

func TestHarvest_CheckSafety_Bad_BinaryFile(t *testing.T) {
	_, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	// Add a binary file
	fs.Write(core.JoinPath(repoDir, "app.exe"), "binary")
	run(t, repoDir, "git", "add", "-f", "app.exe")
	run(t, repoDir, "git", "commit", "-m", "add binary")

	reason := testMon.checkSafety(repoDir)
	core.AssertContains(t, reason, "binary file added")
	core.AssertContains(t, reason, "app.exe")
}

func TestHarvest_CheckSafety_Bad_LargeFile(t *testing.T) {
	_, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	// Add a file > 1MB
	bigData := make([]byte, 1024*1024+1)
	fs.Write(core.JoinPath(repoDir, "huge.txt"), string(bigData))
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "add large file")

	reason := testMon.checkSafety(repoDir)
	core.AssertContains(t, reason, "large file")
	core.AssertContains(t, reason, "huge.txt")
}

func TestHarvest_UpdateStatus_Bad_WriteFailure(t *testing.T) {
	core.AssertNotPanics(t, func() {
		updateStatus("/dev/null/impossible", "ready-for-review", "")
	})
}

func TestHarvest_HarvestWorkspace_Good_Case(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime

	result := mon.harvestWorkspace(wsDir)
	core.AssertNotNil(t, result)
	core.AssertEqual(t, "test-repo", result.repo)
	core.AssertEqual(t, "agent/test-task", result.branch)
	core.AssertEqual(t, 1, result.files)
	core.AssertEqual(t, "", result.rejected)

	// Verify status updated
	var st map[string]any
	core.RequireTrue(t, core.JSONUnmarshalString(fs.Read(core.JoinPath(wsDir, "status.json")).Value.(string), &st).OK)
	core.AssertEqual(t, "ready-for-review", st["status"])
}

func TestHarvest_HarvestWorkspace_Bad_NotCompleted(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "running", "test-repo", "agent/test-task")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	result := mon.harvestWorkspace(wsDir)
	core.AssertNil(t, result)
}

func TestHarvest_HarvestWorkspace_Bad_MainBranch(t *testing.T) {
	_, wsDir := initTestRepo(t)

	// Switch back to main
	repoDir := core.JoinPath(wsDir, "repo")
	run(t, repoDir, "git", "checkout", "main")

	writeStatus(t, wsDir, "completed", "test-repo", "main")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	result := mon.harvestWorkspace(wsDir)
	core.AssertNil(t, result)
}

func TestHarvest_HarvestWorkspace_Bad_BinaryRejected(t *testing.T) {
	_, wsDir := initTestRepo(t)
	repoDir := core.JoinPath(wsDir, "repo")

	// Add binary
	fs.Write(core.JoinPath(repoDir, "build.so"), "elf")
	run(t, repoDir, "git", "add", "-f", "build.so")
	run(t, repoDir, "git", "commit", "-m", "add binary")

	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime

	result := mon.harvestWorkspace(wsDir)
	core.AssertNotNil(t, result)
	core.AssertContains(t, result.rejected, "binary file added")

	// Verify status set to rejected
	var st map[string]any
	core.JSONUnmarshalString(fs.Read(core.JoinPath(wsDir, "status.json")).Value.(string), &st)
	core.AssertEqual(t, "rejected", st["status"])
}

func TestHarvest_HarvestCompleted_Good_ChannelEvents(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	// Create a Core with process + IPC handler to capture HarvestComplete messages
	var captured []messages.HarvestComplete
	c := core.New(core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.HarvestComplete); ok {
			captured = append(captured, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	// Call harvestWorkspace directly since harvestCompleted uses agentic.WorkspaceRoot()
	result := mon.harvestWorkspace(wsDir)
	core.AssertNotNil(t, result)
	core.AssertEqual(t, "", result.rejected)

	// Simulate what harvestCompleted does with the result — emit IPC
	mon.Core().ACTION(messages.HarvestComplete{Repo: result.repo, Branch: result.branch, Files: result.files})

	core.AssertLen(t, captured, 1)
	core.AssertEqual(t, "test-repo", captured[0].Repo)
	core.AssertEqual(t, "agent/test-task", captured[0].Branch)
	core.AssertEqual(t, 1, captured[0].Files)
}

func TestHarvest_HarvestCompleted_Good_MultipleWorkspaces(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	for i := range 2 {
		name := core.Sprintf("ws-%d", i)
		wsDir := core.JoinPath(wsRoot, "workspace", name)

		sourceDir := core.JoinPath(wsRoot, core.Sprintf("source-%d", i))
		fs.EnsureDir(sourceDir)
		run(t, sourceDir, "git", "init")
		run(t, sourceDir, "git", "checkout", "-b", "main")
		fs.Write(core.JoinPath(sourceDir, "README.md"), "# test")
		run(t, sourceDir, "git", "add", ".")
		run(t, sourceDir, "git", "commit", "-m", "init")

		fs.EnsureDir(wsDir)
		run(t, wsDir, "git", "clone", sourceDir, "repo")
		repoDir := core.JoinPath(wsDir, "repo")
		run(t, repoDir, "git", "checkout", "-b", "agent/test-task")
		fs.Write(core.JoinPath(repoDir, "new.go"), "package main\n")
		run(t, repoDir, "git", "add", ".")
		run(t, repoDir, "git", "commit", "-m", "agent work")

		writeStatus(t, wsDir, "completed", core.Sprintf("repo-%d", i), "agent/test-task")
	}

	var harvests []messages.HarvestComplete
	c := core.New(core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.HarvestComplete); ok {
			harvests = append(harvests, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	msg := mon.harvestCompleted()
	core.AssertContains(t, msg, "Harvested:")
	core.AssertContains(t, msg, "repo-0")
	core.AssertContains(t, msg, "repo-1")

	core.AssertGreaterOrEqual(t, len(harvests), 2)
}

func TestHarvest_HarvestCompleted_Good_Empty(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	msg := mon.harvestCompleted()
	core.AssertEqual(t, "", msg)
}

func TestHarvest_HarvestCompleted_Good_RejectedWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	sourceDir := core.JoinPath(wsRoot, "source-rej")
	fs.EnsureDir(sourceDir)
	run(t, sourceDir, "git", "init")
	run(t, sourceDir, "git", "checkout", "-b", "main")
	fs.Write(core.JoinPath(sourceDir, "README.md"), "# test")
	run(t, sourceDir, "git", "add", ".")
	run(t, sourceDir, "git", "commit", "-m", "init")

	wsDir := core.JoinPath(wsRoot, "workspace", "ws-rej")
	fs.EnsureDir(wsDir)
	run(t, wsDir, "git", "clone", sourceDir, "repo")
	repoDir := core.JoinPath(wsDir, "repo")
	run(t, repoDir, "git", "checkout", "-b", "agent/test-task")
	fs.Write(core.JoinPath(repoDir, "new.go"), "package main\n")
	run(t, repoDir, "git", "add", ".")
	run(t, repoDir, "git", "commit", "-m", "agent work")

	fs.Write(core.JoinPath(repoDir, "app.exe"), "binary")
	run(t, repoDir, "git", "add", "-f", "app.exe")
	run(t, repoDir, "git", "commit", "-m", "add binary")

	writeStatus(t, wsDir, "completed", "rej-repo", "agent/test-task")

	var rejections []messages.HarvestRejected
	c := core.New(core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.HarvestRejected); ok {
			rejections = append(rejections, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	msg := mon.harvestCompleted()
	core.AssertContains(t, msg, "REJECTED")

	core.AssertLen(t, rejections, 1)
	core.AssertContains(t, rejections[0].Reason, "binary file added")
}

func TestHarvest_UpdateStatus_Good_Case(t *testing.T) {
	dir := t.TempDir()
	initial := map[string]any{"status": "completed", "repo": "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(initial))

	updateStatus(dir, "ready-for-review", "")

	var st map[string]any
	core.JSONUnmarshalString(fs.Read(core.JoinPath(dir, "status.json")).Value.(string), &st)
	core.AssertEqual(t, "ready-for-review", st["status"])
}

func TestHarvest_UpdateStatus_Good_WithQuestion(t *testing.T) {
	dir := t.TempDir()
	initial := map[string]any{"status": "completed", "repo": "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(initial))

	updateStatus(dir, "rejected", "binary file: app.exe")

	var st map[string]any
	core.JSONUnmarshalString(fs.Read(core.JoinPath(dir, "status.json")).Value.(string), &st)
	core.AssertEqual(t, "rejected", st["status"])
	core.AssertEqual(t, "binary file: app.exe", st["question"])
}
