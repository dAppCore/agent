// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sync"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestHandlers_HandleIPCEvents_Good(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)
	// HandleIPCEvents was auto-registered — Core should not panic on ACTION
	assert.NotPanics(t, func() {
		c.ACTION(messages.AgentCompleted{Workspace: "nonexistent", Repo: "test", Status: "completed"})
	})
}

func TestHandlers_PokeOnCompletion_Good(t *testing.T) {
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

	require.Eventually(t, func() bool { return len(poked) == 1 }, time.Second, 10*time.Millisecond)
}

func TestHandlers_IngestOnCompletion_Good(t *testing.T) {
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

func TestHandlers_IgnoresNonCompleted_Good(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	// Non-completed status — ingest still runs (it handles all completions)
	assert.NotPanics(t, func() {
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

func TestHandlers_RegisterHandlers_Good_CompletionPipeline(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-5"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	require.True(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo")).OK)
	require.NoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
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

	require.Eventually(t, func() bool {
		return seen("qa") && seen("auto-pr") && seen("verify") && seen("ingest") && seen("poke")
	}, time.Second, 10*time.Millisecond)
}

func TestHandlers_FindWorkspaceByPR_Good_MatchesPRNumber(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	firstWorkspace := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-1")
	secondWorkspace := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-2")
	require.True(t, fs.EnsureDir(firstWorkspace).OK)
	require.True(t, fs.EnsureDir(secondWorkspace).OK)

	require.NoError(t, writeStatus(firstWorkspace, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/first",
		PRURL:  "https://forge.lthn.ai/core/go-io/pulls/12",
	}))
	require.NoError(t, writeStatus(secondWorkspace, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/second",
		PRURL:  "https://forge.lthn.ai/core/go-io/pulls/13",
	}))

	result := findWorkspaceByPRWithInfo("go-io", "", 13, "https://forge.lthn.ai/core/go-io/pulls/13")
	assert.Equal(t, secondWorkspace, result)
}

func TestHandlers_IngestDisabled_Bad(t *testing.T) {
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

func TestHandlers_ResolveWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "core", "go-io", "task-15")
	fs.EnsureDir(ws)

	result := resolveWorkspace("core/go-io/task-15")
	assert.Equal(t, ws, result)
}

func TestHandlers_ResolveWorkspace_Bad(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	result := resolveWorkspace("nonexistent")
	assert.Empty(t, result)
}

func TestHandlers_FindWorkspaceByPR_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "ws-test")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Repo: "go-io", Branch: "agent/fix", Status: "completed"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	result := findWorkspaceByPR("go-io", "agent/fix")
	assert.Equal(t, ws, result)
}

func TestHandlers_FindWorkspaceByPR_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Deep layout: org/repo/task
	ws := core.JoinPath(wsRoot, "core", "agent", "task-5")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Repo: "agent", Branch: "agent/tests", Status: "completed"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	result := findWorkspaceByPR("agent", "agent/tests")
	assert.Equal(t, ws, result)
}

// --- command registration ---

func TestHandlers_Commandsforge_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerForgeCommands() })
}

func TestHandlers_Commandsworkspace_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerWorkspaceCommands() })
}
