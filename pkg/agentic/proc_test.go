// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPrep is the package-level PrepSubsystem for tests that need process execution.
var testPrep *PrepSubsystem

// testCore is the package-level Core with go-process registered.
var testCore *core.Core

// TestMain sets up a PrepSubsystem with go-process registered for all tests in the package.
func TestMain(m *testing.M) {
	testCore = core.New(
		core.WithService(process.Register),
	)
	testCore.ServiceStartup(context.Background(), nil)

	testPrep = &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	os.Exit(m.Run())
}

// newPrepWithProcess creates a PrepSubsystem wired to testCore for tests that
// need process execution via s.Core().Process().
func newPrepWithProcess() *PrepSubsystem {
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

// --- runCmd ---

func TestProc_RunCmd_Good(t *testing.T) {
	dir := t.TempDir()
	r := testPrep.runCmd(context.Background(), dir, "echo", "hello")
	assert.True(t, r.OK)
	assert.Contains(t, core.Trim(r.Value.(string)), "hello")
}

func TestProc_RunCmd_Bad(t *testing.T) {
	dir := t.TempDir()
	r := testPrep.runCmd(context.Background(), dir, "nonexistent-command-xyz")
	assert.False(t, r.OK)
}

func TestProc_RunCmd_Ugly(t *testing.T) {
	dir := t.TempDir()
	// Empty command string — should error
	r := testPrep.runCmd(context.Background(), dir, "")
	assert.False(t, r.OK)
}

// --- runCmdEnv ---

func TestProc_RunCmdEnv_Good(t *testing.T) {
	dir := t.TempDir()
	r := testPrep.runCmdEnv(context.Background(), dir, []string{"MY_CUSTOM_VAR=hello_test"}, "env")
	assert.True(t, r.OK)
	assert.Contains(t, r.Value.(string), "MY_CUSTOM_VAR=hello_test")
}

func TestProc_RunCmdEnv_Bad(t *testing.T) {
	dir := t.TempDir()
	r := testPrep.runCmdEnv(context.Background(), dir, []string{"FOO=bar"}, "nonexistent-command-xyz")
	assert.False(t, r.OK)
}

func TestProc_RunCmdEnv_Ugly(t *testing.T) {
	dir := t.TempDir()
	// Empty env slice — should work fine, just no extra vars
	r := testPrep.runCmdEnv(context.Background(), dir, []string{}, "echo", "works")
	assert.True(t, r.OK)
	assert.Contains(t, core.Trim(r.Value.(string)), "works")
}

// --- runCmdOK ---

func TestProc_RunCmdOK_Good(t *testing.T) {
	dir := t.TempDir()
	assert.True(t, testPrep.runCmdOK(context.Background(), dir, "echo", "ok"))
}

func TestProc_RunCmdOK_Bad(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, testPrep.runCmdOK(context.Background(), dir, "nonexistent-command-xyz"))
}

func TestProc_RunCmdOK_Ugly(t *testing.T) {
	dir := t.TempDir()
	// "false" command returns exit 1
	assert.False(t, testPrep.runCmdOK(context.Background(), dir, "false"))
}

// --- gitCmd ---

func TestProc_GitCmd_Good(t *testing.T) {
	dir := t.TempDir()
	r := testPrep.gitCmd(context.Background(), dir, "--version")
	assert.True(t, r.OK)
}

func TestProc_GitCmd_Bad(t *testing.T) {
	// git log in a non-git dir should fail
	dir := t.TempDir()
	r := testPrep.gitCmd(context.Background(), dir, "log")
	assert.False(t, r.OK)
}

func TestProc_GitCmd_Ugly(t *testing.T) {
	dir := t.TempDir()
	// Empty args — git with no arguments exits 1
	r := testPrep.gitCmd(context.Background(), dir)
	assert.False(t, r.OK)
}

// --- gitCmdOK ---

func TestProc_GitCmdOK_Good(t *testing.T) {
	dir := t.TempDir()
	assert.True(t, testPrep.gitCmdOK(context.Background(), dir, "--version"))
}

func TestProc_GitCmdOK_Bad(t *testing.T) {
	// git log in non-git dir returns false
	dir := t.TempDir()
	assert.False(t, testPrep.gitCmdOK(context.Background(), dir, "log"))
}

func TestProc_GitCmdOK_Ugly(t *testing.T) {
	// Empty dir string — git may use cwd, which may or may not be a repo
	// Just ensure no panic
	assert.NotPanics(t, func() {
		testPrep.gitCmdOK(context.Background(), "", "--version")
	})
}

// --- gitOutput ---

func TestProc_GitOutput_Good(t *testing.T) {
	dir := initTestRepo(t)
	branch := testPrep.gitOutput(context.Background(), dir, "rev-parse", "--abbrev-ref", "HEAD")
	assert.Equal(t, "main", branch)
}

func TestProc_GitOutput_Bad(t *testing.T) {
	// Non-git dir returns empty string
	dir := t.TempDir()
	out := testPrep.gitOutput(context.Background(), dir, "rev-parse", "--abbrev-ref", "HEAD")
	assert.Equal(t, "", out)
}

func TestProc_GitOutput_Ugly(t *testing.T) {
	// Failed command returns empty string
	dir := t.TempDir()
	out := testPrep.gitOutput(context.Background(), dir, "log", "--oneline", "-5")
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
}

// --- initTestRepo creates a git repo with commits for proc tests ---

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
	require.True(t, fs.Write(core.JoinPath(dir, "README.md"), "# Test").OK)
	run("git", "add", "README.md")
	run("git", "commit", "-m", "initial commit")
	return dir
}
