// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- runCmd ---

func TestProc_RunCmd_Good(t *testing.T) {
	dir := t.TempDir()
	out, err := runCmd(context.Background(), dir, "echo", "hello")
	assert.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(out), "hello")
}

func TestProc_RunCmd_Bad(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(context.Background(), dir, "nonexistent-command-xyz")
	assert.Error(t, err)
}

func TestProc_RunCmd_Ugly(t *testing.T) {
	dir := t.TempDir()
	// Empty command string — should error
	_, err := runCmd(context.Background(), dir, "")
	assert.Error(t, err)
}

// --- runCmdEnv ---

func TestProc_RunCmdEnv_Good(t *testing.T) {
	dir := t.TempDir()
	out, err := runCmdEnv(context.Background(), dir, []string{"MY_CUSTOM_VAR=hello_test"}, "env")
	assert.NoError(t, err)
	assert.Contains(t, out, "MY_CUSTOM_VAR=hello_test")
}

func TestProc_RunCmdEnv_Bad(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmdEnv(context.Background(), dir, []string{"FOO=bar"}, "nonexistent-command-xyz")
	assert.Error(t, err)
}

func TestProc_RunCmdEnv_Ugly(t *testing.T) {
	dir := t.TempDir()
	// Empty env slice — should work fine, just no extra vars
	out, err := runCmdEnv(context.Background(), dir, []string{}, "echo", "works")
	assert.NoError(t, err)
	assert.Contains(t, strings.TrimSpace(out), "works")
}

// --- runCmdOK ---

func TestProc_RunCmdOK_Good(t *testing.T) {
	dir := t.TempDir()
	assert.True(t, runCmdOK(context.Background(), dir, "echo", "ok"))
}

func TestProc_RunCmdOK_Bad(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, runCmdOK(context.Background(), dir, "nonexistent-command-xyz"))
}

func TestProc_RunCmdOK_Ugly(t *testing.T) {
	dir := t.TempDir()
	// "false" command returns exit 1
	assert.False(t, runCmdOK(context.Background(), dir, "false"))
}

// --- gitCmd ---

func TestProc_GitCmd_Good(t *testing.T) {
	dir := t.TempDir()
	_, err := gitCmd(context.Background(), dir, "--version")
	assert.NoError(t, err)
}

func TestProc_GitCmd_Bad(t *testing.T) {
	// git log in a non-git dir should fail
	dir := t.TempDir()
	_, err := gitCmd(context.Background(), dir, "log")
	assert.Error(t, err)
}

func TestProc_GitCmd_Ugly(t *testing.T) {
	dir := t.TempDir()
	// Empty args — git with no arguments exits 1
	_, err := gitCmd(context.Background(), dir)
	assert.Error(t, err)
}

// --- gitCmdOK ---

func TestProc_GitCmdOK_Good(t *testing.T) {
	dir := t.TempDir()
	assert.True(t, gitCmdOK(context.Background(), dir, "--version"))
}

func TestProc_GitCmdOK_Bad(t *testing.T) {
	// git log in non-git dir returns false
	dir := t.TempDir()
	assert.False(t, gitCmdOK(context.Background(), dir, "log"))
}

func TestProc_GitCmdOK_Ugly(t *testing.T) {
	// Empty dir string — git may use cwd, which may or may not be a repo
	// Just ensure no panic
	assert.NotPanics(t, func() {
		gitCmdOK(context.Background(), "", "--version")
	})
}

// --- gitOutput ---

func TestProc_GitOutput_Good(t *testing.T) {
	dir := t.TempDir()
	// Init a git repo with a commit so we can read the branch
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Test")
	run("git", "config", "user.email", "test@test.com")
	run("git", "commit", "--allow-empty", "-m", "init")

	branch := gitOutput(context.Background(), dir, "rev-parse", "--abbrev-ref", "HEAD")
	assert.Equal(t, "main", branch)
}

func TestProc_GitOutput_Bad(t *testing.T) {
	// Non-git dir returns empty string
	dir := t.TempDir()
	out := gitOutput(context.Background(), dir, "rev-parse", "--abbrev-ref", "HEAD")
	assert.Equal(t, "", out)
}

func TestProc_GitOutput_Ugly(t *testing.T) {
	// Failed command returns empty string
	dir := t.TempDir()
	out := gitOutput(context.Background(), dir, "log", "--oneline", "-5")
	assert.Equal(t, "", out)
}

// --- processIsRunning ---

func TestProc_ProcessIsRunning_Good(t *testing.T) {
	// Own PID should be running
	pid := os.Getpid()
	assert.True(t, processIsRunning("", pid))
}

func TestProc_ProcessIsRunning_Bad(t *testing.T) {
	// PID 999999 should not be running (extremely unlikely to exist)
	assert.False(t, processIsRunning("", 999999))
}

func TestProc_ProcessIsRunning_Ugly(t *testing.T) {
	// PID 0 — should return false (invalid PID guard: pid > 0 is false for 0)
	assert.False(t, processIsRunning("", 0))

	// Empty processID with PID 0 — both paths fail
	assert.False(t, processIsRunning("", 0))
}

// --- processKill ---

func TestProc_ProcessKill_Good(t *testing.T) {
	t.Skip("would need real process to kill")
}

func TestProc_ProcessKill_Bad(t *testing.T) {
	// PID 999999 should fail to kill
	assert.False(t, processKill("", 999999))
}

func TestProc_ProcessKill_Ugly(t *testing.T) {
	// PID 0 — pid > 0 guard returns false
	assert.False(t, processKill("", 0))

	// Empty processID with PID 0 — both paths fail
	assert.False(t, processKill("", 0))
}

// --- ensureProcess ---

func TestProc_EnsureProcess_Good(t *testing.T) {
	// Call twice — verify no panic (idempotent via sync.Once)
	assert.NotPanics(t, func() {
		ensureProcess()
		ensureProcess()
	})
}

func TestProc_EnsureProcess_Bad(t *testing.T) {
	t.Skip("no bad path without mocking")
}

func TestProc_EnsureProcess_Ugly(t *testing.T) {
	// Call from multiple goroutines concurrently — sync.Once should handle this
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NotPanics(t, func() {
				ensureProcess()
			})
		}()
	}
	wg.Wait()
}

// --- initTestRepo is a helper to create a git repo with commits for proc tests ---

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, string(out))
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Test")
	run("git", "config", "user.email", "test@test.com")
	require.True(t, fs.Write(filepath.Join(dir, "README.md"), "# Test").OK)
	run("git", "add", "README.md")
	run("git", "commit", "-m", "initial commit")
	return dir
}
