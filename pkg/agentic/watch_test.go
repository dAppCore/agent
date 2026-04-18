// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func writeWatchStatus(root, name string, status WorkspaceStatus) string {
	wsDir := core.JoinPath(root, "workspace", name)
	fs.EnsureDir(wsDir)
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(status))
	return wsDir
}

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
	setTestWorkspace(t, root)

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
	setTestWorkspace(t, root)

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
	setTestWorkspace(t, root)

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
	setTestWorkspace(t, root)
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

// --- watch Good/Bad/Ugly ---

func TestWatch_Watch_Good_AutoDiscoversAndCompletes(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	writeWatchStatus(root, "core/go-io/task-42", WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeWatchStatus(root, "core/go-io/task-42", WorkspaceStatus{
			Status: "ready-for-review",
			Repo:   "go-io",
			Agent:  "codex",
			PRURL:  "https://forge.lthn.ai/core/go-io/pulls/42",
		})
	}()

	s := newPrepWithProcess()
	_, out, err := s.watch(context.Background(), nil, WatchInput{
		PollInterval: 1,
		Timeout:      2,
	})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Len(t, out.Completed, 1)
	assert.Empty(t, out.Failed)
	assert.Equal(t, "core/go-io/task-42", out.Completed[0].Workspace)
	assert.Equal(t, "ready-for-review", out.Completed[0].Status)
	assert.Equal(t, "https://forge.lthn.ai/core/go-io/pulls/42", out.Completed[0].PRURL)
}

func TestWatch_Watch_Good_ExpandsParentWorkspacePrefix(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	writeWatchStatus(root, "core/go-io/task-41", WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
	})
	writeWatchStatus(root, "core/go-io/task-42", WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		writeWatchStatus(root, "core/go-io/task-41", WorkspaceStatus{
			Status: "completed",
			Repo:   "go-io",
			Agent:  "codex",
		})
		time.Sleep(50 * time.Millisecond)
		writeWatchStatus(root, "core/go-io/task-42", WorkspaceStatus{
			Status: "completed",
			Repo:   "go-io",
			Agent:  "codex",
		})
	}()

	s := newPrepWithProcess()
	_, out, err := s.watch(context.Background(), nil, WatchInput{
		Workspaces:   []string{"core/go-io"},
		PollInterval: 1,
		Timeout:      2,
	})
	assert.NoError(t, err)
	assert.True(t, out.Success)
	assert.Empty(t, out.Failed)
	assert.Len(t, out.Completed, 2)
	assert.ElementsMatch(t, []string{"core/go-io/task-41", "core/go-io/task-42"}, []string{
		out.Completed[0].Workspace,
		out.Completed[1].Workspace,
	})
}

func TestWatch_Watch_Bad_CancelledContext(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	writeWatchStatus(root, "ws-running", WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s := newPrepWithProcess()
	_, out, err := s.watch(ctx, nil, WatchInput{
		Workspaces:   []string{"ws-running"},
		PollInterval: 1,
		Timeout:      2,
	})
	assert.Error(t, err)
	assert.False(t, out.Success)
	assert.Empty(t, out.Completed)
	assert.Empty(t, out.Failed)
}

func TestWatch_Watch_Ugly_TimeoutMarksRemainingFailed(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	writeWatchStatus(root, "ws-stuck", WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
	})

	s := newPrepWithProcess()
	_, out, err := s.watch(context.Background(), nil, WatchInput{
		Workspaces:   []string{"ws-stuck"},
		PollInterval: 1,
		Timeout:      1,
	})
	assert.NoError(t, err)
	assert.False(t, out.Success)
	assert.Empty(t, out.Completed)
	assert.Len(t, out.Failed, 1)
	assert.Equal(t, "ws-stuck", out.Failed[0].Workspace)
	assert.Equal(t, "timeout", out.Failed[0].Status)
}
