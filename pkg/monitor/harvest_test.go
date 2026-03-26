// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"testing"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Create workspace dir with src/ clone
	wsDir = core.JoinPath(t.TempDir(), "workspace")
	srcDir := core.JoinPath(wsDir, "src")
	fs.EnsureDir(wsDir)
	run(t, wsDir, "git", "clone", sourceDir, "src")

	// Create agent branch with a commit
	run(t, srcDir, "git", "checkout", "-b", "agent/test-task")
	fs.Write(core.JoinPath(srcDir, "new.go"), "package main\n")
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "agent work")

	return sourceDir, wsDir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	gitEnv := []string{"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test"}
	r := testMon.Core().Process().RunWithEnv(context.Background(), dir, gitEnv, name, args...)
	require.True(t, r.OK, "command %s %v failed: %s", name, args, r.Value)
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

func TestHarvest_DetectBranch_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := core.JoinPath(wsDir, "src")

	branch := testMon.detectBranch(srcDir)
	assert.Equal(t, "agent/test-task", branch)
}

func TestHarvest_DetectBranch_Bad_NoRepo(t *testing.T) {
	branch := testMon.detectBranch(t.TempDir())
	assert.Equal(t, "", branch)
}

func TestHarvest_CountUnpushed_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := core.JoinPath(wsDir, "src")

	count := testMon.countUnpushed(srcDir, "agent/test-task")
	assert.Equal(t, 1, count)
}

func TestHarvest_CountChangedFiles_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := core.JoinPath(wsDir, "src")

	count := testMon.countChangedFiles(srcDir)
	assert.Equal(t, 1, count)
}

func TestHarvest_CheckSafety_Good_CleanWorkspace(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := core.JoinPath(wsDir, "src")

	reason := testMon.checkSafety(srcDir)
	assert.Equal(t, "", reason)
}

func TestHarvest_CheckSafety_Bad_BinaryFile(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := core.JoinPath(wsDir, "src")

	// Add a binary file
	fs.Write(core.JoinPath(srcDir, "app.exe"), "binary")
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "add binary")

	reason := testMon.checkSafety(srcDir)
	assert.Contains(t, reason, "binary file added")
	assert.Contains(t, reason, "app.exe")
}

func TestHarvest_CheckSafety_Bad_LargeFile(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := core.JoinPath(wsDir, "src")

	// Add a file > 1MB
	bigData := make([]byte, 1024*1024+1)
	fs.Write(core.JoinPath(srcDir, "huge.txt"), string(bigData))
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "add large file")

	reason := testMon.checkSafety(srcDir)
	assert.Contains(t, reason, "large file")
	assert.Contains(t, reason, "huge.txt")
}

func TestHarvest_HarvestWorkspace_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime

	result := mon.harvestWorkspace(wsDir)
	require.NotNil(t, result)
	assert.Equal(t, "test-repo", result.repo)
	assert.Equal(t, "agent/test-task", result.branch)
	assert.Equal(t, 1, result.files)
	assert.Equal(t, "", result.rejected)

	// Verify status updated
	var st map[string]any
	require.True(t, core.JSONUnmarshalString(fs.Read(core.JoinPath(wsDir, "status.json")).Value.(string), &st).OK)
	assert.Equal(t, "ready-for-review", st["status"])
}

func TestHarvest_HarvestWorkspace_Bad_NotCompleted(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "running", "test-repo", "agent/test-task")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	result := mon.harvestWorkspace(wsDir)
	assert.Nil(t, result)
}

func TestHarvest_HarvestWorkspace_Bad_MainBranch(t *testing.T) {
	_, wsDir := initTestRepo(t)

	// Switch back to main
	srcDir := core.JoinPath(wsDir, "src")
	run(t, srcDir, "git", "checkout", "main")

	writeStatus(t, wsDir, "completed", "test-repo", "main")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	result := mon.harvestWorkspace(wsDir)
	assert.Nil(t, result)
}

func TestHarvest_HarvestWorkspace_Bad_BinaryRejected(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := core.JoinPath(wsDir, "src")

	// Add binary
	fs.Write(core.JoinPath(srcDir, "build.so"), "elf")
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "add binary")

	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime

	result := mon.harvestWorkspace(wsDir)
	require.NotNil(t, result)
	assert.Contains(t, result.rejected, "binary file added")

	// Verify status set to rejected
	var st map[string]any
	core.JSONUnmarshalString(fs.Read(core.JoinPath(wsDir, "status.json")).Value.(string), &st)
	assert.Equal(t, "rejected", st["status"])
}

func TestHarvest_HarvestCompleted_Good_ChannelEvents(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	// Create a Core with process + IPC handler to capture HarvestComplete messages
	var captured []messages.HarvestComplete
	c := core.New(core.WithService(process.Register))
	c.ServiceStartup(context.Background(), nil)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.HarvestComplete); ok {
			captured = append(captured, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})

	// Call harvestWorkspace directly since harvestCompleted uses agentic.WorkspaceRoot()
	result := mon.harvestWorkspace(wsDir)
	require.NotNil(t, result)
	assert.Equal(t, "", result.rejected)

	// Simulate what harvestCompleted does with the result — emit IPC
	mon.Core().ACTION(messages.HarvestComplete{Repo: result.repo, Branch: result.branch, Files: result.files})

	require.Len(t, captured, 1)
	assert.Equal(t, "test-repo", captured[0].Repo)
	assert.Equal(t, "agent/test-task", captured[0].Branch)
	assert.Equal(t, 1, captured[0].Files)
}

func TestHarvest_UpdateStatus_Good(t *testing.T) {
	dir := t.TempDir()
	initial := map[string]any{"status": "completed", "repo": "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(initial))

	updateStatus(dir, "ready-for-review", "")

	var st map[string]any
	core.JSONUnmarshalString(fs.Read(core.JoinPath(dir, "status.json")).Value.(string), &st)
	assert.Equal(t, "ready-for-review", st["status"])
}

func TestHarvest_UpdateStatus_Good_WithQuestion(t *testing.T) {
	dir := t.TempDir()
	initial := map[string]any{"status": "completed", "repo": "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(initial))

	updateStatus(dir, "rejected", "binary file: app.exe")

	var st map[string]any
	core.JSONUnmarshalString(fs.Read(core.JoinPath(dir, "status.json")).Value.(string), &st)
	assert.Equal(t, "rejected", st["status"])
	assert.Equal(t, "binary file: app.exe", st["question"])
}

