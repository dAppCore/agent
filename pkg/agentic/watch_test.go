// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// --- resolveWorkspaceDir ---

func TestWatch_ResolveWorkspaceDir_Good_RelativeName(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	dir := s.resolveWorkspaceDir("go-io-abc123")
	assert.Contains(t, dir, "go-io-abc123")
	assert.True(t, core.PathIsAbs(dir))
}

func TestWatch_ResolveWorkspaceDir_Good_AbsolutePath(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
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
	os.MkdirAll(ws1, 0o755)
	st1, _ := json.Marshal(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"})
	os.WriteFile(core.JoinPath(ws1, "status.json"), st1, 0o644)

	// Create completed workspace (should not be in active list)
	ws2 := core.JoinPath(wsRoot, "ws-done")
	os.MkdirAll(ws2, 0o755)
	st2, _ := json.Marshal(WorkspaceStatus{Status: "completed", Repo: "go-crypt", Agent: "codex"})
	os.WriteFile(core.JoinPath(ws2, "status.json"), st2, 0o644)

	// Create queued workspace
	ws3 := core.JoinPath(wsRoot, "ws-queued")
	os.MkdirAll(ws3, 0o755)
	st3, _ := json.Marshal(WorkspaceStatus{Status: "queued", Repo: "go-log", Agent: "gemini"})
	os.WriteFile(core.JoinPath(ws3, "status.json"), st3, 0o644)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	active := s.findActiveWorkspaces()
	assert.Contains(t, active, "ws-running")
	assert.Contains(t, active, "ws-queued")
	assert.NotContains(t, active, "ws-done")
}

func TestWatch_FindActiveWorkspaces_Good_Empty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Ensure workspace dir exists but is empty
	os.MkdirAll(core.JoinPath(root, "workspace"), 0o755)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
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
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
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
	os.MkdirAll(ws1, 0o755)
	os.WriteFile(core.JoinPath(ws1, "status.json"), []byte("not-valid-json{{{"), 0o644)

	// Create valid running workspace
	ws2 := core.JoinPath(wsRoot, "ws-valid")
	os.MkdirAll(ws2, 0o755)
	st, _ := json.Marshal(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"})
	os.WriteFile(core.JoinPath(ws2, "status.json"), st, 0o644)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
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
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	dir := s.resolveWorkspaceDir("")
	assert.NotEmpty(t, dir, "empty name should still resolve to workspace root")
	assert.True(t, core.PathIsAbs(dir))
}

func TestWatch_ResolveWorkspaceDir_Ugly(t *testing.T) {
	// Name with path traversal "../.."
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() {
		dir := s.resolveWorkspaceDir("../..")
		// JoinPath handles traversal; result should be absolute
		assert.True(t, core.PathIsAbs(dir))
	})
}
