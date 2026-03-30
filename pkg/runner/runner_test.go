// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"context"
	"testing"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- New ---

func TestRunner_New_Good(t *testing.T) {
	svc := New()
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.workspaces)
	assert.NotNil(t, svc.backoff)
	assert.NotNil(t, svc.failCount)
	assert.False(t, svc.frozen, "New() doesn't freeze — startRunner does based on env")
}

func TestRunner_New_Bad_NoServiceRuntime(t *testing.T) {
	svc := New()
	assert.Nil(t, svc.ServiceRuntime)
	assert.False(t, svc.IsFrozen(), "New() doesn't set frozen — startRunner does")
}

func TestRunner_New_Ugly_MultipleInstances(t *testing.T) {
	a := New()
	b := New()
	assert.NotSame(t, a, b, "each call returns a fresh instance")
	assert.NotSame(t, a.workspaces, b.workspaces)
}

// --- Register ---

func TestRunner_Register_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	r := Register(c)
	assert.True(t, r.OK)
	assert.NotNil(t, r.Value)
}

func TestRunner_Register_Bad_NilCore(t *testing.T) {
	assert.Panics(t, func() {
		Register(nil)
	})
}

func TestRunner_Register_Ugly_ConfigLoaded(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	Register(c)
	// Config should have agents.concurrency set (even if defaults)
	r := c.Config().Get("agents.dispatch")
	assert.NotNil(t, r)
}

// --- IsFrozen ---

func TestRunner_IsFrozen_Good(t *testing.T) {
	svc := New()
	svc.frozen = true
	assert.True(t, svc.IsFrozen())
}

func TestRunner_IsFrozen_Bad_AfterUnfreeze(t *testing.T) {
	svc := New()
	svc.frozen = false
	assert.False(t, svc.IsFrozen())
}

func TestRunner_IsFrozen_Ugly_ToggleRapidly(t *testing.T) {
	svc := New()
	for i := 0; i < 100; i++ {
		svc.frozen = i%2 == 0
	}
	// i=99 → 99%2==1 → false. Last write wins.
	assert.False(t, svc.IsFrozen(), "last toggle wins")
}

// --- TrackWorkspace ---

func TestRunner_TrackWorkspace_Good(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("core/go-io/dev", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 12345,
	})
	r := svc.workspaces.Get("core/go-io/dev")
	assert.True(t, r.OK)
}

func TestRunner_TrackWorkspace_Good_AgenticStatus(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("core/go-io/dev", &agentic.WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 12345,
	})
	r := svc.workspaces.Get("core/go-io/dev")
	assert.True(t, r.OK)
}

func TestRunner_TrackWorkspace_Bad_NilWorkspaces(t *testing.T) {
	svc := &Service{}
	assert.NotPanics(t, func() {
		svc.TrackWorkspace("test", &WorkspaceStatus{Status: "running"})
	})
}

func TestRunner_TrackWorkspace_Ugly_AnyType(t *testing.T) {
	svc := New()
	// TrackWorkspace accepts any — JSON round-trip converts
	svc.TrackWorkspace("test", map[string]any{
		"status": "running", "agent": "codex", "repo": "go-io",
	})
	r := svc.workspaces.Get("test")
	assert.True(t, r.OK)
	ws := r.Value.(*WorkspaceStatus)
	assert.Equal(t, "running", ws.Status)
	assert.Equal(t, "codex", ws.Agent)
}

// --- Workspaces ---

func TestRunner_Workspaces_Good(t *testing.T) {
	svc := New()
	assert.NotNil(t, svc.Workspaces())
	assert.Equal(t, 0, svc.Workspaces().Len())
}

func TestRunner_Workspaces_Bad_AfterTrack(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	assert.Equal(t, 1, svc.Workspaces().Len())
}

func TestRunner_Workspaces_Ugly_OverwriteSameName(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "completed"})
	assert.Equal(t, 1, svc.Workspaces().Len())
	r := svc.workspaces.Get("ws-1")
	ws := r.Value.(*WorkspaceStatus)
	assert.Equal(t, "completed", ws.Status)
}

// --- Poke ---

func TestRunner_Poke_Good(t *testing.T) {
	svc := New()
	svc.pokeCh = make(chan struct{}, 1)
	svc.Poke()
	assert.Len(t, svc.pokeCh, 1)
}

func TestRunner_Poke_Bad_NilChannel(t *testing.T) {
	svc := New()
	assert.NotPanics(t, func() {
		svc.Poke()
	})
}

func TestRunner_Poke_Ugly_DoublePoke(t *testing.T) {
	svc := New()
	svc.pokeCh = make(chan struct{}, 1)
	svc.Poke()
	svc.Poke() // second poke is a no-op (channel full)
	assert.Len(t, svc.pokeCh, 1)
}

// --- Actions ---

func TestRunner_ActionStatus_Good(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	svc.TrackWorkspace("ws-2", &WorkspaceStatus{Status: "completed"})
	svc.TrackWorkspace("ws-3", &WorkspaceStatus{Status: "queued"})

	r := svc.actionStatus(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
	m := r.Value.(map[string]int)
	assert.Equal(t, 1, m["running"])
	assert.Equal(t, 1, m["completed"])
	assert.Equal(t, 1, m["queued"])
	assert.Equal(t, 3, m["total"])
}

func TestRunner_ActionStatus_Bad_Empty(t *testing.T) {
	svc := New()
	r := svc.actionStatus(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
	m := r.Value.(map[string]int)
	assert.Equal(t, 0, m["total"])
}

func TestRunner_ActionStatus_Ugly_AllStatuses(t *testing.T) {
	svc := New()
	for _, s := range []string{"running", "queued", "completed", "merged", "ready-for-review", "failed", "blocked"} {
		svc.TrackWorkspace("ws-"+s, &WorkspaceStatus{Status: s})
	}
	r := svc.actionStatus(context.Background(), core.NewOptions())
	m := r.Value.(map[string]int)
	assert.Equal(t, 7, m["total"])
	assert.Equal(t, 3, m["completed"]) // completed + merged + ready-for-review
	assert.Equal(t, 2, m["failed"])    // failed + blocked
}

func TestRunner_ActionStart_Good(t *testing.T) {
	svc := New()
	svc.frozen = true
	svc.pokeCh = make(chan struct{}, 1)
	assert.True(t, svc.IsFrozen())
	r := svc.actionStart(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
	assert.False(t, svc.IsFrozen())
}

func TestRunner_ActionStop_Good(t *testing.T) {
	svc := New()
	svc.frozen = false
	r := svc.actionStop(context.Background(), core.NewOptions())
	assert.True(t, r.OK)
	assert.True(t, svc.IsFrozen())
}

func TestRunner_ActionDispatch_Bad_Frozen(t *testing.T) {
	svc := New()
	svc.frozen = true
	r := svc.actionDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "agent", Value: "codex"},
	))
	assert.False(t, r.OK, "should deny when frozen")
}

func TestRunner_ActionDispatch_Good_Unfrozen(t *testing.T) {
	svc := New()
	svc.frozen = false
	r := svc.actionDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "agent", Value: "codex"},
	))
	assert.True(t, r.OK, "should allow when unfrozen and no concurrency limit hit")
}

// --- OnStartup ---

func TestRunner_OnStartup_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	r := svc.OnStartup(context.Background())
	assert.True(t, r.OK)

	// Actions should be registered
	assert.NotNil(t, c.Action("runner.dispatch"))
	assert.NotNil(t, c.Action("runner.status"))
	assert.NotNil(t, c.Action("runner.start"))
	assert.NotNil(t, c.Action("runner.stop"))
	assert.NotNil(t, c.Action("runner.kill"))
	assert.NotNil(t, c.Action("runner.poke"))
}

func TestRunner_OnStartup_Bad_NilCore(t *testing.T) {
	svc := New()
	// No ServiceRuntime — OnStartup should panic on s.Core()
	assert.Panics(t, func() {
		svc.OnStartup(context.Background())
	})
}

func TestRunner_OnStartup_Ugly_StartsRunnerLoop(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.OnStartup(context.Background())
	assert.NotNil(t, svc.pokeCh, "runner loop should be started")
	assert.True(t, svc.IsFrozen(), "should be frozen without CORE_AGENT_DISPATCH=1")
}

// --- OnShutdown ---

func TestRunner_OnShutdown_Good(t *testing.T) {
	svc := New()
	svc.frozen = false
	r := svc.OnShutdown(context.Background())
	assert.True(t, r.OK)
	assert.True(t, svc.IsFrozen())
}

func TestRunner_OnShutdown_Bad_AlreadyFrozen(t *testing.T) {
	svc := New()
	r := svc.OnShutdown(context.Background())
	assert.True(t, r.OK)
	assert.True(t, svc.IsFrozen())
}

func TestRunner_OnShutdown_Ugly_DoesNotPanic(t *testing.T) {
	svc := New()
	assert.NotPanics(t, func() {
		svc.OnShutdown(context.Background())
	})
}

// --- HandleIPCEvents ---

func TestRunner_HandleIPCEvents_Good_UnknownMessage(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	// Unknown message type — should not panic
	r := svc.HandleIPCEvents(c, "unknown")
	assert.True(t, r.OK)
}

func TestRunner_HandleIPCEvents_Good_UpdatesMatchingWorkspaceOnly(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.TrackWorkspace("core/go-io/task-1", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 111,
	})
	svc.TrackWorkspace("core/go-io/task-2", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 222,
	})

	r := svc.HandleIPCEvents(c, messages.AgentCompleted{
		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-1", Status: "completed",
	})
	assert.True(t, r.OK)

	first := svc.workspaces.Get("core/go-io/task-1").Value.(*WorkspaceStatus)
	second := svc.workspaces.Get("core/go-io/task-2").Value.(*WorkspaceStatus)
	assert.Equal(t, "completed", first.Status)
	assert.Equal(t, 0, first.PID)
	assert.Equal(t, "running", second.Status)
	assert.Equal(t, 222, second.PID)
}

func TestRunner_HydrateWorkspaces_Good_DeepWorkspaceName(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(wsDir)
	require.True(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		PID:    99999999,
	}).OK)

	svc := New()
	svc.hydrateWorkspaces()

	r := svc.workspaces.Get("core/go-io/task-5")
	assert.True(t, r.OK)
	st := r.Value.(*WorkspaceStatus)
	assert.Equal(t, "queued", st.Status)
	assert.Equal(t, "go-io", st.Repo)
}

// --- WriteStatus / ReadStatusResult ---

func TestRunner_WriteReadStatus_Good(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", PID: 999}
	require.True(t, WriteStatus(dir, st).OK)

	got := mustReadStatus(t, dir)
	assert.Equal(t, "running", got.Status)
	assert.Equal(t, "codex", got.Agent)
	assert.Equal(t, 999, got.PID)
}

func TestRunner_ReadStatus_Bad_NoFile(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	assert.False(t, result.OK)
	_, ok := result.Value.(error)
	assert.True(t, ok)
}

func TestRunner_WriteReadStatus_Ugly_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	require.True(t, WriteStatus(dir, &WorkspaceStatus{Status: "running"}).OK)
	require.True(t, WriteStatus(dir, &WorkspaceStatus{Status: "completed"}).OK)

	got := mustReadStatus(t, dir)
	assert.Equal(t, "completed", got.Status)
}
