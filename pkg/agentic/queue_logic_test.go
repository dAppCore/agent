// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- countRunningByModel ---

func TestQueue_CountRunningByModel_Good_Empty(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	core.AssertEqual(t, 0, s.countRunningByModel("claude:opus"))
}

func TestQueue_CountRunningByModel_Good_SkipsNonRunning(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Completed workspace — must not be counted
	ws := core.JoinPath(root, "workspace", "test-ws")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex:gpt-5.4",
		PID:    0,
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	core.AssertEqual(t, 0, s.countRunningByModel("codex:gpt-5.4"))
}

func TestQueue_CountRunningByModel_Good_SkipsMismatchedModel(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	ws := core.JoinPath(root, "workspace", "model-ws")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "running",
		Agent:  "gemini:flash",
		PID:    0,
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	// Asking for gemini:pro — must not count gemini:flash
	core.AssertEqual(t, 0, s.countRunningByModel("gemini:pro"))
}

func TestQueue_CountRunningByModel_Good_DeepLayout(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Deep layout: workspace/org/repo/task-N/status.json
	ws := core.JoinPath(root, "workspace", "core", "go-io", "task-1")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex:gpt-5.4",
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	// Completed, so count is still 0
	core.AssertEqual(t, 0, s.countRunningByModel("codex:gpt-5.4"))
}

// --- drainQueue ---

func TestQueue_DrainQueue_Good_FrozenReturnsImmediately(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: true, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	// Must not panic and must not block
	core.AssertNotPanics(t, func() {
		s.drainQueue()
	})
}

func TestQueue_DrainQueue_Good_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: false, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	// No workspaces — must return without error/panic
	core.AssertNotPanics(t, func() {
		s.drainQueue()
	})
}

// --- Poke ---

func TestPokeNilChannel_PrepSubsystem_Poke_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), pokeCh: nil}
	// Must not panic when pokeCh is nil
	core.AssertNotPanics(t, func() {
		s.Poke()
	})
}

func TestPokeChannel_PrepSubsystem_Poke_Good(t *testing.T) {
	// Poke is now a no-op — queue poke is owned by pkg/runner.Service.
	// Verify it does not send to the channel and does not panic.
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)

	core.AssertNotPanics(t, func() { s.Poke() })
	core.AssertLen(t, s.pokeCh, 0, "no-op poke should not enqueue a signal")
}

func TestRunner_Poke_Good_NonBlockingWhenFull(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)
	// Pre-fill the channel
	s.pokeCh <- struct{}{}

	// Second poke must not block or panic
	core.AssertNotPanics(t, func() {
		s.Poke()
	})
	core.AssertLen(t, s.pokeCh, 1, "channel length should remain 1")
}

// --- StartRunner ---

func TestStartRunner_PrepSubsystem_StartRunner_Good(t *testing.T) {
	// StartRunner is now a no-op — queue drain is owned by pkg/runner.Service.
	// Verify it does not panic and does not set pokeCh.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	core.AssertNil(t, s.pokeCh)

	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s.pokeCh, "no-op StartRunner should not initialise pokeCh")
}

func TestRunner_StartRunner_Good_FrozenByDefault(t *testing.T) {
	// StartRunner is now a no-op — frozen state is owned by pkg/runner.Service.
	// Verify it does not panic; frozen state is not managed here.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	core.AssertNotPanics(t, func() { s.StartRunner() })
}

func TestRunner_StartRunner_Good_AutoStartEnvVar(t *testing.T) {
	// StartRunner is now a no-op — env var handling is in pkg/runner.Service.
	// Verify the no-op does not panic.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "1")

	s := NewPrep()
	core.AssertNotPanics(t, func() { s.StartRunner() })
}

// --- Poke Ugly ---

func TestPokeRapidFire_PrepSubsystem_Poke_Ugly(t *testing.T) {
	// Poke on a closed channel — the select with default protects against panic
	// but closing + sending would panic. However, Poke uses non-blocking send,
	// so we test that pokeCh=nil is safe (already tested), and that
	// double-filling is safe (already tested). Here we test rapid multi-poke.
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)

	// Rapid-fire pokes — should all be safe
	for i := 0; i < 100; i++ {
		core.AssertNotPanics(t, func() { s.Poke() })
	}
	// Channel should have at most 1 signal
	core.AssertLessOrEqual(t, len(s.pokeCh), 1)
}

// --- StartRunner Bad/Ugly ---

func TestStartRunner_PrepSubsystem_StartRunner_Bad(t *testing.T) {
	// StartRunner is now a no-op — frozen state and pokeCh are owned by pkg/runner.Service.
	// Verify the no-op does not panic and does not modify state.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s.pokeCh, "no-op StartRunner should not create pokeCh")
}

func TestStartRunner_PrepSubsystem_StartRunner_Ugly(t *testing.T) {
	// StartRunner is now a no-op — calling it multiple times must not panic.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "1")

	s := NewPrep()

	// Call twice — both are no-ops, must not panic
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s.pokeCh, "no-op StartRunner should not create pokeCh")
}

// --- DefaultBranch ---

func TestDefaultBranchFallback_PrepSubsystem_DefaultBranch_Bad(t *testing.T) {
	// Non-git temp dir — git commands fail, fallback is "main"
	dir := t.TempDir()
	branch := testPrep.DefaultBranch(dir)
	core.AssertEqual(t, "main", branch)
}

func TestDefaultBranchRepo_PrepSubsystem_DefaultBranch_Good(t *testing.T) {
	dir := t.TempDir()
	// Init a real git repo with a main branch
	core.RequireNoError(t, runGitInit(dir))

	branch := testPrep.DefaultBranch(dir)
	// Any valid branch name — just must not panic or be empty
	core.AssertNotEmpty(t, branch)
}

// --- LocalFs ---

func TestPaths_LocalFs_Good_NonNil(t *testing.T) {
	f := LocalFs()
	missing := core.JoinPath(t.TempDir(), "missing.txt")
	core.AssertNotNil(t, f, "LocalFs should return a non-nil *core.Fs")
	core.AssertFalse(t, f.Read(missing).OK)
}

func TestCanRead_Agentic_LocalFs_Good(t *testing.T) {
	dir := t.TempDir()
	path := core.JoinPath(dir, "hello.txt")
	core.RequireTrue(t, fs.Write(path, "hello").OK)

	f := LocalFs()
	r := f.Read(path)
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "hello", r.Value.(string))
}

// --- helpers ---

// --- RunLoop ---

func TestRunner_RunLoop_Good_Case(t *testing.T) {
	coreApp := core.New(core.WithOption("name", "test"))
	called := false
	coreApp.Action("runner.start", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(coreApp, AgentOptions{})}
	s.StartRunner()
	core.AssertTrue(t, called)
}

func TestRunner_RunLoop_Bad_Case(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{})}
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s.pokeCh)
}

func TestRunner_RunLoop_Ugly_Case(t *testing.T) {
	var s *PrepSubsystem
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s)
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
