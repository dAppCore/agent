// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sync"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// newCoreForHandlerTests creates a Core with PrepSubsystem registered via
// RegisterService — HandleIPCEvents is auto-discovered.
func newCoreForHandlerTests(t *testing.T) (*core.Core, *PrepSubsystem) {
	t.Helper()
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		codePath:   t.TempDir(),
		pokeCh:     make(chan struct{}, 1),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}

	c := core.New()
	c.Config().Enable("auto-ingest")
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})
	// RegisterService auto-discovers HandleIPCEvents on PrepSubsystem
	c.RegisterService("agentic", s)
	RegisterHandlers(c, s)

	return c, s
}

// --- HandleIPCEvents ---

func TestHandlers_PrepSubsystem_HandleIPCEvents_Good(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)
	// HandleIPCEvents was auto-registered — Core should not panic on ACTION
	core.AssertNotPanics(t, func() {
		c.ACTION(messages.AgentCompleted{Workspace: "nonexistent", Repo: "test", Status: "completed"})
	})
}

func TestHandlers_PokeOnCompletion_Good_Case(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	poked := make(chan struct{}, 1)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if _, ok := msg.(messages.PokeQueue); ok {
			select {
			case poked <- struct{}{}:
			default:
			}
		}
		return core.Result{OK: true}
	})
	c.Action("runner.poke", func(_ context.Context, _ core.Options) core.Result { return core.Result{OK: true} })

	c.ACTION(messages.AgentCompleted{
		Workspace: "ws-test", Repo: "go-io", Status: "completed",
	})

	requireEventually(t, func() bool { return len(poked) == 1 }, time.Second, 10*time.Millisecond)
}

func TestHandlers_IngestOnCompletion_Good_Case(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	root := WorkspaceRoot()
	workspaceName := "core/test/task-2"
	workspaceDir := core.JoinPath(root, workspaceName)
	repoDir := core.JoinPath(workspaceDir, "repo")
	fs.EnsureDir(repoDir)

	st := &WorkspaceStatus{
		Status: "completed",
		Repo:   "test",
		Agent:  "codex",
		Task:   "Review code",
	}
	writeStatus(workspaceDir, st)

	// Should not panic — ingest handler runs but no findings file
	c.ACTION(messages.AgentCompleted{
		Workspace: workspaceName,
		Repo:      "test",
		Status:    "completed",
	})
}

func TestHandlers_IgnoresNonCompleted_Good_Case(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	// Non-completed status — ingest still runs (it handles all completions)
	core.AssertNotPanics(t, func() {
		c.ACTION(messages.AgentCompleted{
			Workspace: "nonexistent",
			Repo:      "test",
			Status:    "failed",
		})
	})
}

func TestHandlers_PokeQueue_Good(t *testing.T) {
	c, s := newCoreForHandlerTests(t)
	s.frozen = true // frozen so drainQueue is a no-op

	// PokeQueue message → drainQueue called
	c.ACTION(messages.PokeQueue{})
	// Should call drainQueue without panic
}

func TestCompletionPipeline_RegisterHandlers_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-5"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo")).OK)
	core.RequireNoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-tests",
		Agent:  "codex",
	}))

	var mu sync.Mutex
	called := make(map[string]bool)
	mark := func(name string) {
		mu.Lock()
		called[name] = true
		mu.Unlock()
	}
	seen := func(name string) bool {
		mu.Lock()
		defer mu.Unlock()
		return called[name]
	}

	c := core.New()
	c.Config().Enable("auto-ingest")
	RegisterHandlers(c, &PrepSubsystem{})

	c.Action("agentic.qa", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("qa")
		}
		c.ACTION(messages.QAResult{Workspace: workspaceName, Repo: "go-io", Passed: true})
		return core.Result{OK: true}
	})
	c.Action("agentic.auto-pr", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("auto-pr")
		}
		c.ACTION(messages.PRCreated{
			Repo:   "go-io",
			Branch: "agent/fix-tests",
			PRURL:  "https://forge.lthn.ai/core/go-io/pulls/12",
			PRNum:  12,
		})
		return core.Result{OK: true}
	})
	c.Action("agentic.verify", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("verify")
		}
		return core.Result{OK: true}
	})
	c.Action("agentic.ingest", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("ingest")
		}
		return core.Result{OK: true}
	})
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if _, ok := msg.(messages.PokeQueue); ok {
			mark("poke")
		}
		return core.Result{OK: true}
	})
	c.Action("runner.poke", func(_ context.Context, _ core.Options) core.Result { return core.Result{OK: true} })

	c.ACTION(messages.AgentCompleted{
		Workspace: workspaceName,
		Repo:      "go-io",
		Status:    "completed",
	})

	requireEventually(t, func() bool {
		return seen("qa") && seen("auto-pr") && seen("verify") && seen("ingest") && seen("poke")
	}, time.Second, 10*time.Millisecond)
}

func TestHandlers_FindWorkspaceByPR_Good_MatchesPRNumber(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	firstWorkspace := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-1")
	secondWorkspace := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-2")
	core.RequireTrue(t, fs.EnsureDir(firstWorkspace).OK)
	core.RequireTrue(t, fs.EnsureDir(secondWorkspace).OK)

	core.RequireNoError(t, writeStatus(firstWorkspace, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/first",
		PRURL:  "https://forge.lthn.ai/core/go-io/pulls/12",
	}))
	core.RequireNoError(t, writeStatus(secondWorkspace, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/second",
		PRURL:  "https://forge.lthn.ai/core/go-io/pulls/13",
	}))

	result := findWorkspaceByPRWithInfo("go-io", "", 13, "https://forge.lthn.ai/core/go-io/pulls/13")
	core.AssertEqual(t, secondWorkspace, result)
}

func TestHandlers_IngestDisabled_Bad_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		pokeCh:     make(chan struct{}, 1),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}

	c := core.New()
	c.Config().Disable("auto-ingest") // disabled
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})
	c.RegisterService("agentic", s)

	wsDir := core.JoinPath(WorkspaceRoot(), "ws-test")
	fs.EnsureDir(core.JoinPath(wsDir, "repo"))
	writeStatus(wsDir, &WorkspaceStatus{Status: "completed", Repo: "test", Agent: "codex"})

	// With auto-ingest disabled, should still not panic
	c.ACTION(messages.AgentCompleted{Workspace: "ws-test", Repo: "test", Status: "completed"})
}

func TestHandlers_ResolveWorkspace_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "core", "go-io", "task-15")
	fs.EnsureDir(ws)

	result := resolveWorkspace("core/go-io/task-15")
	core.AssertEqual(t, ws, result)
}

func TestHandlers_ResolveWorkspace_Bad_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	result := resolveWorkspace("nonexistent")
	core.AssertEmpty(t, result)
}

func TestHandlers_FindWorkspaceByPR_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "ws-test")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Repo: "go-io", Branch: "agent/fix", Status: "completed"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	result := findWorkspaceByPR("go-io", "agent/fix")
	core.AssertEqual(t, ws, result)
}

func TestHandlers_FindWorkspaceByPR_Ugly_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Deep layout: org/repo/task
	ws := core.JoinPath(wsRoot, "core", "agent", "task-5")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Repo: "agent", Branch: "agent/tests", Status: "completed"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	result := findWorkspaceByPR("agent", "agent/tests")
	core.AssertEqual(t, ws, result)
}

// --- command registration ---

func TestHandlers_Commandsforge_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	core.AssertNotPanics(t, func() { s.registerForgeCommands() })
}

func TestHandlers_Commandsworkspace_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	core.AssertNotPanics(t, func() { s.registerWorkspaceCommands() })
}

func TestHandlers_RegisterHandlers_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-5"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo")).OK)
	core.RequireNoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-tests",
		Agent:  "codex",
	}))

	var mu sync.Mutex
	called := make(map[string]bool)
	mark := func(name string) {
		mu.Lock()
		called[name] = true
		mu.Unlock()
	}
	seen := func(name string) bool {
		mu.Lock()
		defer mu.Unlock()
		return called[name]
	}

	c := core.New()
	c.Config().Enable("auto-ingest")
	RegisterHandlers(c, &PrepSubsystem{})

	c.Action("agentic.qa", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("qa")
		}
		c.ACTION(messages.QAResult{Workspace: workspaceName, Repo: "go-io", Passed: true})
		return core.Result{OK: true}
	})
	c.Action("agentic.auto-pr", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("auto-pr")
		}
		c.ACTION(messages.PRCreated{
			Repo:   "go-io",
			Branch: "agent/fix-tests",
			PRURL:  "https://forge.lthn.ai/core/go-io/pulls/12",
			PRNum:  12,
		})
		return core.Result{OK: true}
	})
	c.Action("agentic.verify", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("verify")
		}
		return core.Result{OK: true}
	})
	c.Action("agentic.ingest", func(_ context.Context, options core.Options) core.Result {
		if options.String("workspace") == workspaceDir {
			mark("ingest")
		}
		return core.Result{OK: true}
	})
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if _, ok := msg.(messages.PokeQueue); ok {
			mark("poke")
		}
		return core.Result{OK: true}
	})
	c.Action("runner.poke", func(_ context.Context, _ core.Options) core.Result { return core.Result{OK: true} })

	c.ACTION(messages.AgentCompleted{
		Workspace: workspaceName,
		Repo:      "go-io",
		Status:    "completed",
	})

	requireEventually(t, func() bool {
		return seen("qa") && seen("auto-pr") && seen("verify") && seen("ingest") && seen("poke")
	}, time.Second, 10*time.Millisecond)
}

func TestHandlers_RegisterHandlers_Bad(t *testing.T) {
	core.AssertNotPanics(t, func() {
		RegisterHandlers(nil, nil)
	})
}

func TestHandlers_RegisterHandlers_Ugly(t *testing.T) {
	c := core.New()
	c.Action("runner.poke", func(context.Context, core.Options) core.Result { return core.Result{OK: true} })

	RegisterHandlers(c, &PrepSubsystem{})
	RegisterHandlers(c, &PrepSubsystem{})
	core.AssertNotPanics(t, func() {
		c.ACTION(messages.AgentCompleted{})
	})
}

func TestHandlers_PrepSubsystem_HandleIPCEvents_Bad(t *testing.T) {
	c, subsystem := newCoreForHandlerTests(t)
	result := subsystem.HandleIPCEvents(c, messages.PokeQueue{})

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestHandlers_PrepSubsystem_HandleIPCEvents_Ugly(t *testing.T) {
	c, subsystem := newCoreForHandlerTests(t)
	result := subsystem.HandleIPCEvents(c, messages.SpawnQueued{Workspace: "missing", Agent: "claude", Task: "Write docs"})

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestHandlers_PrepSubsystem_SpawnFromQueue_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	writeFakeAgentBinary(t, "claude")

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-3")
	core.RequireTrue(t, fs.EnsureDir(WorkspaceRepoDir(workspaceDir)).OK)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceMetaDir(workspaceDir)).OK)

	subsystem := newPrepWithProcess()
	result := subsystem.SpawnFromQueue("claude", "Write docs", workspaceDir)
	core.RequireTrue(t, result.OK)

	pid, ok := result.Value.(int)
	core.RequireTrue(t, ok)
	if pid <= 0 {
		t.Fatalf("expected positive pid, got %d", pid)
	}

	outputPath := agentOutputFile(workspaceDir, "claude")
	requireEventually(t, func() bool { return fs.IsFile(outputPath) }, 5*time.Second, 10*time.Millisecond)
	outputResult := fs.Read(outputPath)
	core.RequireTrue(t, outputResult.OK)
	output := core.Trim(outputResult.Value.(string))
	core.AssertEqual(t, "done", output)
}

func TestHandlers_PrepSubsystem_SpawnFromQueue_Bad(t *testing.T) {
	subsystem := newPrepWithProcess()
	result := subsystem.SpawnFromQueue("robot-from-the-future", "Write docs", t.TempDir())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "unknown agent")
}

func TestHandlers_PrepSubsystem_SpawnFromQueue_Ugly(t *testing.T) {
	writeFakeAgentBinary(t, "claude")

	c := core.New()
	subsystem := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	result := subsystem.SpawnFromQueue("claude", "Write docs", t.TempDir())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "process service not registered")
}
