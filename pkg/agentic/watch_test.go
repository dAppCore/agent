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
