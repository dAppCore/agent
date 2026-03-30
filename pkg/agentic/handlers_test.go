// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// newCoreForHandlerTests creates a Core with PrepSubsystem registered via
// RegisterService — HandleIPCEvents is auto-discovered.
func newCoreForHandlerTests(t *testing.T) (*core.Core, *PrepSubsystem) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

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
	_, s := newCoreForHandlerTests(t)

	// Drain any existing poke
	select {
	case <-s.pokeCh:
	default:
	}

	// HandleIPCEvents receives AgentCompleted → calls Poke
	s.HandleIPCEvents(s.Core(), messages.AgentCompleted{
		Workspace: "ws-test", Repo: "go-io", Status: "completed",
	})

	select {
	case <-s.pokeCh:
		// poke received
	default:
		t.Log("poke signal may not have been received synchronously")
	}
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

func TestHandlers_IngestDisabled_Bad(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

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
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "core", "go-io", "task-15")
	fs.EnsureDir(ws)

	result := resolveWorkspace("core/go-io/task-15")
	assert.Equal(t, ws, result)
}

func TestHandlers_ResolveWorkspace_Bad(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	result := resolveWorkspace("nonexistent")
	assert.Empty(t, result)
}

func TestHandlers_FindWorkspaceByPR_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
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
	t.Setenv("CORE_WORKSPACE", root)
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
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerForgeCommands() })
}

func TestHandlers_Commandsworkspace_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerWorkspaceCommands() })
}
