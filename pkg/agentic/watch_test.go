// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- resolveWorkspaceDir ---

func TestWatch_ResolveWorkspaceDir_Good_RelativeName(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	dir := s.resolveWorkspaceDir("go-io-abc123")
	assert.Contains(t, dir, "go-io-abc123")
	assert.True(t, filepath.IsAbs(dir))
}

func TestWatch_ResolveWorkspaceDir_Good_AbsolutePath(t *testing.T) {
	s := &PrepSubsystem{
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

	wsRoot := filepath.Join(root, "workspace")

	// Create running workspace
	ws1 := filepath.Join(wsRoot, "ws-running")
	os.MkdirAll(ws1, 0o755)
	st1, _ := json.Marshal(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"})
	os.WriteFile(filepath.Join(ws1, "status.json"), st1, 0o644)

	// Create completed workspace (should not be in active list)
	ws2 := filepath.Join(wsRoot, "ws-done")
	os.MkdirAll(ws2, 0o755)
	st2, _ := json.Marshal(WorkspaceStatus{Status: "completed", Repo: "go-crypt", Agent: "codex"})
	os.WriteFile(filepath.Join(ws2, "status.json"), st2, 0o644)

	// Create queued workspace
	ws3 := filepath.Join(wsRoot, "ws-queued")
	os.MkdirAll(ws3, 0o755)
	st3, _ := json.Marshal(WorkspaceStatus{Status: "queued", Repo: "go-log", Agent: "gemini"})
	os.WriteFile(filepath.Join(ws3, "status.json"), st3, 0o644)

	s := &PrepSubsystem{
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
	os.MkdirAll(filepath.Join(root, "workspace"), 0o755)

	s := &PrepSubsystem{
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
	t.Setenv("CORE_WORKSPACE", filepath.Join(root, "nonexistent"))

	s := &PrepSubsystem{
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
	wsRoot := filepath.Join(root, "workspace")

	// Create workspace with corrupt status.json
	ws1 := filepath.Join(wsRoot, "ws-corrupt")
	os.MkdirAll(ws1, 0o755)
	os.WriteFile(filepath.Join(ws1, "status.json"), []byte("not-valid-json{{{"), 0o644)

	// Create valid running workspace
	ws2 := filepath.Join(wsRoot, "ws-valid")
	os.MkdirAll(ws2, 0o755)
	st, _ := json.Marshal(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"})
	os.WriteFile(filepath.Join(ws2, "status.json"), st, 0o644)

	s := &PrepSubsystem{
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
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	dir := s.resolveWorkspaceDir("")
	assert.NotEmpty(t, dir, "empty name should still resolve to workspace root")
	assert.True(t, filepath.IsAbs(dir))
}

func TestWatch_ResolveWorkspaceDir_Ugly(t *testing.T) {
	// Name with path traversal "../.."
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() {
		dir := s.resolveWorkspaceDir("../..")
		// JoinPath handles traversal; result should be absolute
		assert.True(t, filepath.IsAbs(dir))
	})
}
