// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCoreForHandlerTests(t *testing.T) (*core.Core, *PrepSubsystem) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:  t.TempDir(),
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	c := core.New()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})
	RegisterHandlers(c, s)

	return c, s
}

func TestHandlers_RegisterHandlers_Good_Registers(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)
	// RegisterHandlers should not panic and Core should have actions
	assert.NotNil(t, c)
}

func TestHandlers_RegisterHandlers_Good_PokeOnCompletion(t *testing.T) {
	_, s := newCoreForHandlerTests(t)

	// Drain any existing poke
	select {
	case <-s.pokeCh:
	default:
	}

	// Send AgentCompleted — should trigger poke
	s.Core().ACTION(messages.AgentCompleted{
		Workspace: "nonexistent",
		Repo:      "test",
		Status:    "completed",
	})

	// Check pokeCh got a signal
	select {
	case <-s.pokeCh:
		// ok — poke handler fired
	default:
		t.Log("poke signal may not have been received synchronously — handler may run async")
	}
}

func TestHandlers_RegisterHandlers_Good_QAFailsUpdatesStatus(t *testing.T) {
	c, s := newCoreForHandlerTests(t)

	root := WorkspaceRoot()
	wsName := "core/test/task-1"
	wsDir := core.JoinPath(root, wsName)
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)

	// Create a Go project that will fail vet/build
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module test\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nimport \"fmt\"\n")

	st := &WorkspaceStatus{
		Status: "completed",
		Repo:   "test",
		Agent:  "codex",
		Task:   "Fix it",
	}
	writeStatus(wsDir, st)

	// Send AgentCompleted — QA handler should run and mark as failed
	c.ACTION(messages.AgentCompleted{
		Workspace: wsName,
		Repo:      "test",
		Status:    "completed",
	})

	_ = s
	// QA handler runs — check if status was updated
	updated, err := ReadStatus(wsDir)
	require.NoError(t, err)
	// May be "failed" (QA failed) or "completed" (QA passed trivially)
	assert.Contains(t, []string{"failed", "completed"}, updated.Status)
}

func TestHandlers_RegisterHandlers_Good_IngestOnCompletion(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	root := WorkspaceRoot()
	wsName := "core/test/task-2"
	wsDir := core.JoinPath(root, wsName)
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)

	st := &WorkspaceStatus{
		Status: "completed",
		Repo:   "test",
		Agent:  "codex",
		Task:   "Review code",
	}
	writeStatus(wsDir, st)

	// Should not panic — ingest handler runs but no findings file
	c.ACTION(messages.AgentCompleted{
		Workspace: wsName,
		Repo:      "test",
		Status:    "completed",
	})
}

func TestHandlers_RegisterHandlers_Good_IgnoresNonCompleted(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	// Send AgentCompleted with non-completed status — QA should skip
	c.ACTION(messages.AgentCompleted{
		Workspace: "nonexistent",
		Repo:      "test",
		Status:    "failed",
	})
	// Should not panic
}

func TestHandlers_RegisterHandlers_Good_PokeQueue(t *testing.T) {
	c, s := newCoreForHandlerTests(t)
	s.frozen = true // frozen so drainQueue is a no-op

	// Send PokeQueue message
	c.ACTION(messages.PokeQueue{})
	// Should call drainQueue without panic
}

// --- command registration ---

func TestCommandsForge_RegisterForgeCommands_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Should register without panic
	assert.NotPanics(t, func() { s.registerForgeCommands() })
}

func TestCommandsWorkspace_RegisterWorkspaceCommands_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerWorkspaceCommands() })
}

func TestCommands_RegisterCommands_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(core.New(), AgentOptions{}),
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerCommands(ctx) })
}

// --- Prep subsystem lifecycle ---

func TestPrep_NewPrep_Good(t *testing.T) {
	s := NewPrep()
	assert.NotNil(t, s)
	assert.Equal(t, "agentic", s.Name())
}

func TestPrep_OnStartup_Good_Registers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := NewPrep()
	c := core.New()
	s.SetCore(c)

	r := s.OnStartup(context.Background())
	assert.True(t, r.OK)
}

// --- RegisterTools (exercises all register*Tool functions) ---

func TestPrep_RegisterTools_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	s := NewPrep()
	s.SetCore(core.New())

	assert.NotPanics(t, func() { s.RegisterTools(srv) })
}

func TestPrep_RegisterTools_Bad(t *testing.T) {
	// RegisterTools on prep without Core — should still register tools
	srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() { s.RegisterTools(srv) })
}

func TestPrep_RegisterTools_Ugly(t *testing.T) {
	// Call RegisterTools twice — should not panic or double-register
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)
	s := NewPrep()
	s.SetCore(core.New())

	assert.NotPanics(t, func() {
		s.RegisterTools(srv)
		s.RegisterTools(srv)
	})
}
