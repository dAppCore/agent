// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// --- resolveWorkspaceDir ---

func TestWatch_ResolveWorkspaceDir_Good_RelativeName(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	dir := s.resolveWorkspaceDir("go-io-abc123")
	assert.Contains(t, dir, "go-io-abc123")
	assert.True(t, core.PathIsAbs(dir))
}

func TestWatch_ResolveWorkspaceDir_Good_AbsolutePath(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	abs := "/some/absolute/path"
	assert.Equal(t, abs, s.resolveWorkspaceDir(abs))
}

// --- findActiveWorkspaces ---

func TestWatch_FindActiveWorkspaces_Good_WithActive(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsRoot := core.JoinPath(root, "workspace")

	// Create running workspace
	ws1 := core.JoinPath(wsRoot, "ws-running")
	fs.EnsureDir(ws1)
	fs.Write(core.JoinPath(ws1, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"}))

	// Create completed workspace (should not be in active list)
	ws2 := core.JoinPath(wsRoot, "ws-done")
	fs.EnsureDir(ws2)
	fs.Write(core.JoinPath(ws2, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "completed", Repo: "go-crypt", Agent: "codex"}))

	// Create queued workspace
	ws3 := core.JoinPath(wsRoot, "ws-queued")
	fs.EnsureDir(ws3)
	fs.Write(core.JoinPath(ws3, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "queued", Repo: "go-log", Agent: "gemini"}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	active := s.findActiveWorkspaces()
	assert.Contains(t, active, "ws-running")
	assert.Contains(t, active, "ws-queued")
	assert.NotContains(t, active, "ws-done")
}

func TestWatch_FindActiveWorkspaces_Good_DeepLayout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	ws := core.JoinPath(root, "workspace", "core", "go-io", "task-15")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status: "running", Repo: "go-io", Agent: "codex",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	active := s.findActiveWorkspaces()
	assert.Contains(t, active, "core/go-io/task-15")
}

func TestWatch_FindActiveWorkspaces_Good_Empty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Ensure workspace dir exists but is empty
	fs.EnsureDir(core.JoinPath(root, "workspace"))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	active := s.findActiveWorkspaces()
	assert.Empty(t, active)
}

// --- findActiveWorkspaces Bad/Ugly ---

func TestWatch_FindActiveWorkspaces_Bad(t *testing.T) {
	// Workspace dir doesn't exist
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", core.JoinPath(root, "nonexistent"))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() {
		active := s.findActiveWorkspaces()
		assert.Empty(t, active)
	})
}

func TestWatch_FindActiveWorkspaces_Ugly(t *testing.T) {
	// Workspaces with corrupt status.json
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create workspace with corrupt status.json
	ws1 := core.JoinPath(wsRoot, "ws-corrupt")
	fs.EnsureDir(ws1)
	fs.Write(core.JoinPath(ws1, "status.json"), "not-valid-json{{{")

	// Create valid running workspace
	ws2 := core.JoinPath(wsRoot, "ws-valid")
	fs.EnsureDir(ws2)
	fs.Write(core.JoinPath(ws2, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	active := s.findActiveWorkspaces()
	// Corrupt workspace should be skipped, valid one should be found
	assert.Contains(t, active, "ws-valid")
	assert.NotContains(t, active, "ws-corrupt")
}

// --- resolveWorkspaceDir Bad/Ugly ---

func TestWatch_ResolveWorkspaceDir_Bad(t *testing.T) {
	// Empty name
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	dir := s.resolveWorkspaceDir("")
	assert.NotEmpty(t, dir, "empty name should still resolve to workspace root")
	assert.True(t, core.PathIsAbs(dir))
}

func TestWatch_ResolveWorkspaceDir_Ugly(t *testing.T) {
	// Name with path traversal "../.."
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() {
		dir := s.resolveWorkspaceDir("../..")
		// JoinPath handles traversal; result should be absolute
		assert.True(t, core.PathIsAbs(dir))
	})
}
