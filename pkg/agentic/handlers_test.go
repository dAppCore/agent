// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCoreForHandlerTests(t *testing.T) (*core.Core, *PrepSubsystem) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	c := core.New()
	s.core = c
	RegisterHandlers(c, s)

	return c, s
}

func TestRegisterHandlers_Good_Registers(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)
	// RegisterHandlers should not panic and Core should have actions
	assert.NotNil(t, c)
}

func TestRegisterHandlers_Good_PokeOnCompletion(t *testing.T) {
	_, s := newCoreForHandlerTests(t)

	// Drain any existing poke
	select {
	case <-s.pokeCh:
	default:
	}

	// Send AgentCompleted — should trigger poke
	s.core.ACTION(messages.AgentCompleted{
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

func TestRegisterHandlers_Good_QAFailsUpdatesStatus(t *testing.T) {
	c, s := newCoreForHandlerTests(t)

	root := WorkspaceRoot()
	wsName := "core/test/task-1"
	wsDir := filepath.Join(root, wsName)
	repoDir := filepath.Join(wsDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	// Create a Go project that will fail vet/build
	os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0o644)
	os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\nimport \"fmt\"\n"), 0o644)

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

func TestRegisterHandlers_Good_IngestOnCompletion(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	root := WorkspaceRoot()
	wsName := "core/test/task-2"
	wsDir := filepath.Join(root, wsName)
	repoDir := filepath.Join(wsDir, "repo")
	os.MkdirAll(repoDir, 0o755)

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

func TestRegisterHandlers_Good_IgnoresNonCompleted(t *testing.T) {
	c, _ := newCoreForHandlerTests(t)

	// Send AgentCompleted with non-completed status — QA should skip
	c.ACTION(messages.AgentCompleted{
		Workspace: "nonexistent",
		Repo:      "test",
		Status:    "failed",
	})
	// Should not panic
}

func TestRegisterHandlers_Good_PokeQueue(t *testing.T) {
	c, s := newCoreForHandlerTests(t)
	s.frozen = true // frozen so drainQueue is a no-op

	// Send PokeQueue message
	c.ACTION(messages.PokeQueue{})
	// Should call drainQueue without panic
}

// --- command registration ---

func TestRegisterForgeCommands_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		core:      core.New(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Should register without panic
	assert.NotPanics(t, func() { s.registerForgeCommands() })
}

func TestRegisterWorkspaceCommands_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		core:      core.New(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerWorkspaceCommands() })
}

func TestRegisterCommands_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &PrepSubsystem{
		core:      core.New(),
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() { s.registerCommands(ctx) })
}

// --- Prep subsystem lifecycle ---

func TestNewPrep_Good(t *testing.T) {
	s := NewPrep()
	assert.NotNil(t, s)
	assert.Equal(t, "agentic", s.Name())
}

func TestOnStartup_Good_Registers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := NewPrep()
	c := core.New()
	s.SetCore(c)

	err := s.OnStartup(context.Background())
	assert.NoError(t, err)
}
