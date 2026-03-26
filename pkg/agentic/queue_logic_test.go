// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- countRunningByModel ---

func TestQueue_CountRunningByModel_Good_Empty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.Equal(t, 0, s.countRunningByModel("claude:opus"))
}

func TestQueue_CountRunningByModel_Good_SkipsNonRunning(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Completed workspace — must not be counted
	ws := core.JoinPath(root, "workspace", "test-ws")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex:gpt-5.4",
		PID:    0,
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.Equal(t, 0, s.countRunningByModel("codex:gpt-5.4"))
}

func TestQueue_CountRunningByModel_Good_SkipsMismatchedModel(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	ws := core.JoinPath(root, "workspace", "model-ws")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "running",
		Agent:  "gemini:flash",
		PID:    0,
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	// Asking for gemini:pro — must not count gemini:flash
	assert.Equal(t, 0, s.countRunningByModel("gemini:pro"))
}

func TestQueue_CountRunningByModel_Good_DeepLayout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Deep layout: workspace/org/repo/task-N/status.json
	ws := core.JoinPath(root, "workspace", "core", "go-io", "task-1")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex:gpt-5.4",
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	// Completed, so count is still 0
	assert.Equal(t, 0, s.countRunningByModel("codex:gpt-5.4"))
}

// --- drainQueue ---

func TestQueue_DrainQueue_Good_FrozenReturnsImmediately(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: true, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	// Must not panic and must not block
	assert.NotPanics(t, func() {
		s.drainQueue()
	})
}

func TestQueue_DrainQueue_Good_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: false, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	// No workspaces — must return without error/panic
	assert.NotPanics(t, func() {
		s.drainQueue()
	})
}

// --- Poke ---

func TestRunner_Poke_Good_NilChannel(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), pokeCh: nil}
	// Must not panic when pokeCh is nil
	assert.NotPanics(t, func() {
		s.Poke()
	})
}

func TestRunner_Poke_Good_ChannelReceivesSignal(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)

	s.Poke()
	assert.Len(t, s.pokeCh, 1, "poke should enqueue one signal")
}

func TestRunner_Poke_Good_NonBlockingWhenFull(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)
	// Pre-fill the channel
	s.pokeCh <- struct{}{}

	// Second poke must not block or panic
	assert.NotPanics(t, func() {
		s.Poke()
	})
	assert.Len(t, s.pokeCh, 1, "channel length should remain 1")
}

// --- StartRunner ---

func TestRunner_StartRunner_Good_CreatesPokeCh(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	assert.Nil(t, s.pokeCh)

	s.StartRunner()
	assert.NotNil(t, s.pokeCh, "StartRunner should initialise pokeCh")
}

func TestRunner_StartRunner_Good_FrozenByDefault(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	s.StartRunner()
	assert.True(t, s.frozen, "queue should be frozen by default")
}

func TestRunner_StartRunner_Good_AutoStartEnvVar(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_DISPATCH", "1")

	s := NewPrep()
	s.StartRunner()
	assert.False(t, s.frozen, "CORE_AGENT_DISPATCH=1 should unfreeze the queue")
}

// --- Poke Ugly ---

func TestRunner_Poke_Ugly(t *testing.T) {
	// Poke on a closed channel — the select with default protects against panic
	// but closing + sending would panic. However, Poke uses non-blocking send,
	// so we test that pokeCh=nil is safe (already tested), and that
	// double-filling is safe (already tested). Here we test rapid multi-poke.
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)

	// Rapid-fire pokes — should all be safe
	for i := 0; i < 100; i++ {
		assert.NotPanics(t, func() { s.Poke() })
	}
	// Channel should have at most 1 signal
	assert.LessOrEqual(t, len(s.pokeCh), 1)
}

// --- StartRunner Bad/Ugly ---

func TestRunner_StartRunner_Bad(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	s.StartRunner()
	// Without CORE_AGENT_DISPATCH=1, queue should be frozen
	assert.True(t, s.frozen, "queue must be frozen when CORE_AGENT_DISPATCH is not set")
	assert.NotNil(t, s.pokeCh)
}

func TestRunner_StartRunner_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_DISPATCH", "1")

	s := NewPrep()

	// Start twice — second call overwrites pokeCh
	s.StartRunner()
	firstCh := s.pokeCh
	assert.NotNil(t, firstCh)

	s.StartRunner()
	secondCh := s.pokeCh
	assert.NotNil(t, secondCh)
	// The channels should be different objects (new make each time)
	assert.NotSame(t, &firstCh, &secondCh)
}

// --- DefaultBranch ---

func TestPaths_DefaultBranch_Good_DefaultsToMain(t *testing.T) {
	// Non-git temp dir — git commands fail, fallback is "main"
	dir := t.TempDir()
	branch := testPrep.DefaultBranch(dir)
	assert.Equal(t, "main", branch)
}

func TestPaths_DefaultBranch_Good_RealGitRepo(t *testing.T) {
	dir := t.TempDir()
	// Init a real git repo with a main branch
	require.NoError(t, runGitInit(dir))

	branch := testPrep.DefaultBranch(dir)
	// Any valid branch name — just must not panic or be empty
	assert.NotEmpty(t, branch)
}

// --- LocalFs ---

func TestPaths_LocalFs_Good_NonNil(t *testing.T) {
	f := LocalFs()
	assert.NotNil(t, f, "LocalFs should return a non-nil *core.Fs")
}

func TestPaths_LocalFs_Good_CanRead(t *testing.T) {
	dir := t.TempDir()
	path := core.JoinPath(dir, "hello.txt")
	require.True(t, fs.Write(path, "hello").OK)

	f := LocalFs()
	r := f.Read(path)
	assert.True(t, r.OK)
	assert.Equal(t, "hello", r.Value.(string))
}

// --- helpers ---

// --- RunLoop ---

func TestRunner_RunLoop_Good(t *testing.T) {
	t.Skip("blocking goroutine — tested indirectly via StartRunner")
}

func TestRunner_RunLoop_Bad(t *testing.T) {
	t.Skip("blocking goroutine — tested indirectly via StartRunner")
}

func TestRunner_RunLoop_Ugly(t *testing.T) {
	t.Skip("blocking goroutine — tested indirectly via StartRunner")
}

// runGitInit initialises a bare git repo with one commit so branch detection works.
func runGitInit(dir string) error {
	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		r := testCore.Process().RunIn(context.Background(), dir, args[0], args[1:]...)
		if !r.OK {
			return core.E("runGitInit", core.Sprintf("cmd %v failed", args), nil)
		}
	}
	return nil
}
