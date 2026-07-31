// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
)

// --- New ---

func TestRunner_New_Good(t *testing.T) {
	svc := New()
	core.AssertNotNil(t, svc)
	core.AssertNotNil(t, svc.workspaces)
	core.AssertNotNil(t, svc.backoff)
	core.AssertNotNil(t, svc.failCount)
	core.AssertFalse(t, svc.frozen, "New() doesn't freeze — startRunner does based on env")
}

func TestNoServiceRuntime_New_Bad(t *testing.T) {
	svc := New()
	core.AssertNil(t, svc.ServiceRuntime)
	core.AssertFalse(t, svc.IsFrozen(), "New() doesn't set frozen — startRunner does")
}

func TestMultipleInstances_New_Ugly(t *testing.T) {
	a := New()
	b := New()
	assertNotSame(t, a, b, "each call returns a fresh instance")
	assertNotSame(t, a.workspaces, b.workspaces)
}

// --- Register ---

func TestRunner_Register_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	r := Register(c)
	core.AssertTrue(t, r.OK)
	core.AssertNotNil(t, r.Value)
}

func TestNilCore_Register_Bad(t *testing.T) {
	core.AssertPanics(t, func() {
		Register(nil)
	})
}

func TestConfigLoaded_Register_Ugly(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	Register(c)
	// Config should have agents.concurrency set (even if defaults)
	r := c.Config().Get("agents.dispatch")
	core.AssertNotNil(t, r)
}

// --- IsFrozen ---

func TestFrozen_Service_IsFrozen_Good(t *testing.T) {
	svc := New()
	svc.frozen = true
	core.AssertTrue(t, svc.IsFrozen())
}

func TestUnfrozen_Service_IsFrozen_Bad(t *testing.T) {
	svc := New()
	svc.frozen = false
	core.AssertFalse(t, svc.IsFrozen())
}

func TestRapidToggle_Service_IsFrozen_Ugly(t *testing.T) {
	svc := New()
	for i := range 100 {
		svc.frozen = i%2 == 0
	}
	// i=99 → 99%2==1 → false. Last write wins.
	core.AssertFalse(t, svc.IsFrozen(), "last toggle wins")
}

// --- TrackWorkspace ---

func TestRunnerStatus_Service_TrackWorkspace_Good(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("core/go-io/dev", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 12345,
	})
	r := svc.workspaces.Get("core/go-io/dev")
	core.AssertTrue(t, r.OK)
}

func TestAgenticStatus_Service_TrackWorkspace_Good(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("core/go-io/dev", &agentic.WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 12345,
	})
	r := svc.workspaces.Get("core/go-io/dev")
	core.AssertTrue(t, r.OK)
}

func TestNilRegistry_Service_TrackWorkspace_Bad(t *testing.T) {
	svc := &Service{}
	core.AssertNotPanics(t, func() {
		svc.TrackWorkspace("test", &WorkspaceStatus{Status: "running"})
	})
}

func TestAnyType_Service_TrackWorkspace_Ugly(t *testing.T) {
	svc := New()
	// TrackWorkspace accepts any — JSON round-trip converts
	svc.TrackWorkspace("test", map[string]any{
		"status": "running", "agent": "codex", "repo": "go-io",
	})
	r := svc.workspaces.Get("test")
	core.AssertTrue(t, r.OK)
	ws := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "running", ws.Status)
	core.AssertEqual(t, "codex", ws.Agent)
}

// --- Workspaces ---

func TestEmptyRegistry_Service_Workspaces_Good(t *testing.T) {
	svc := New()
	core.AssertNotNil(t, svc.Workspaces())
	core.AssertEqual(t, 0, svc.Workspaces().Len())
}

func TestTrackedWorkspace_Service_Workspaces_Bad(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	core.AssertEqual(t, 1, svc.Workspaces().Len())
}

func TestOverwriteSameName_Service_Workspaces_Ugly(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "completed"})
	core.AssertEqual(t, 1, svc.Workspaces().Len())
	r := svc.workspaces.Get("ws-1")
	ws := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "completed", ws.Status)
}

// --- Workspace Query ---

func TestRunner_HandleWorkspaceQuery_Good_Name(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{Status: "running", Agent: "codex"})

	result := svc.handleWorkspaceQuery(nil, WorkspaceQuery{Name: "core/go-io/task-5"})
	core.RequireTrue(t, result.OK)
	status, ok := result.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "running", status.Status)
}

func TestRunner_HandleWorkspaceQuery_Bad_UnknownQuery(t *testing.T) {
	svc := New()

	result := svc.handleWorkspaceQuery(nil, "not a workspace query")
	core.AssertFalse(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestRunner_HandleWorkspaceQuery_Ugly_StatusFilter(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-running", &WorkspaceStatus{Status: "running"})
	svc.TrackWorkspace("ws-completed", &WorkspaceStatus{Status: "completed"})

	result := svc.handleWorkspaceQuery(nil, WorkspaceQuery{Status: "completed"})
	core.RequireTrue(t, result.OK)
	names, ok := result.Value.([]string)
	core.RequireTrue(t, ok)
	core.AssertContains(t, names, "ws-completed")
	core.AssertNotContains(t, names, "ws-running")
}

// --- Poke ---

func TestBufferedChannel_Service_Poke_Good(t *testing.T) {
	svc := New()
	svc.pokeCh = make(chan struct{}, 1)
	svc.Poke()
	core.AssertLen(t, svc.pokeCh, 1)
}

func TestNilChannel_Service_Poke_Bad(t *testing.T) {
	svc := New()
	core.AssertNotPanics(t, func() {
		svc.Poke()
	})
}

func TestDoublePoke_Service_Poke_Ugly(t *testing.T) {
	svc := New()
	svc.pokeCh = make(chan struct{}, 1)
	svc.Poke()
	svc.Poke() // second poke is a no-op (channel full)
	core.AssertLen(t, svc.pokeCh, 1)
}

func TestRunner_ActionPoke_Bad_NoRuntimeDoesNotPanic(t *testing.T) {
	svc := New()
	core.AssertNotPanics(t, func() {
		result := svc.actionPoke(context.Background(), core.NewOptions())
		core.AssertTrue(t, result.OK)
	})
}

// --- Actions ---

func TestRunner_ActionStatus_Good_Case(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	svc.TrackWorkspace("ws-2", &WorkspaceStatus{Status: "completed"})
	svc.TrackWorkspace("ws-3", &WorkspaceStatus{Status: "queued"})

	r := svc.actionStatus(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)
	m := r.Value.(map[string]int)
	core.AssertEqual(t, 1, m["running"])
	core.AssertEqual(t, 1, m["completed"])
	core.AssertEqual(t, 1, m["queued"])
	core.AssertEqual(t, 3, m["total"])
}

func TestRunner_ActionStatus_Bad_Empty(t *testing.T) {
	svc := New()
	r := svc.actionStatus(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)
	m := r.Value.(map[string]int)
	core.AssertEqual(t, 0, m["total"])
}

func TestRunner_ActionStatus_Ugly_AllStatuses(t *testing.T) {
	svc := New()
	for _, s := range []string{"running", "queued", "completed", "merged", "ready-for-review", "failed", "blocked"} {
		svc.TrackWorkspace("ws-"+s, &WorkspaceStatus{Status: s})
	}
	r := svc.actionStatus(context.Background(), core.NewOptions())
	m := r.Value.(map[string]int)
	core.AssertEqual(t, 7, m["total"])
	core.AssertEqual(t, 3, m["completed"]) // completed + merged + ready-for-review
	core.AssertEqual(t, 2, m["failed"])    // failed + blocked
}

func TestRunner_ActionStart_Good_Case(t *testing.T) {
	svc := New()
	svc.frozen = true
	svc.pokeCh = make(chan struct{}, 1)
	core.AssertTrue(t, svc.IsFrozen())
	r := svc.actionStart(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)
	core.AssertFalse(t, svc.IsFrozen())
}

func TestRunner_ActionStop_Good_Case(t *testing.T) {
	svc := New()
	svc.frozen = false
	r := svc.actionStop(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, svc.IsFrozen())
}

func TestRunner_ActionKill_Good_ClearsQueuedWorkspaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	queuedDir := core.JoinPath(wsRoot, "task-queued")
	runningDir := core.JoinPath(wsRoot, "task-running")
	core.RequireTrue(t, fs.EnsureDir(queuedDir).OK)
	core.RequireTrue(t, fs.EnsureDir(runningDir).OK)

	core.RequireTrue(t, WriteStatus(queuedDir, &WorkspaceStatus{
		Status: "queued",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "Queued work",
	}).OK)
	core.RequireTrue(t, WriteStatus(runningDir, &WorkspaceStatus{
		Status: "running",
		Agent:  "claude",
		Repo:   "go-log",
		Task:   "Running work",
	}).OK)

	svc := New()
	result := svc.actionKill(context.Background(), core.NewOptions())
	core.RequireTrue(t, result.OK)
	core.AssertContains(t, result.Value.(string), "cleared 1 queued")

	core.AssertFalse(t, fs.Read(core.JoinPath(queuedDir, "status.json")).OK, "queued workspace should be deleted")
	running := mustReadStatus(t, runningDir)
	core.AssertEqual(t, "failed", running.Status)
	core.AssertEqual(t, 0, running.PID)
}

func TestRunner_ActionKill_Bad_EmptyWorkspaceRoot(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	svc := New()
	result := svc.actionKill(context.Background(), core.NewOptions())
	core.RequireTrue(t, result.OK)
	core.AssertContains(t, result.Value.(string), "cleared 0 queued")
}

func TestRunner_ActionKill_Ugly_InvalidStatusFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")
	wsDir := core.JoinPath(wsRoot, "task-bad")
	core.RequireTrue(t, fs.EnsureDir(wsDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(wsDir, "status.json"), "{not-json").OK)

	svc := New()
	result := svc.actionKill(context.Background(), core.NewOptions())
	core.RequireTrue(t, result.OK)
	core.AssertContains(t, result.Value.(string), "cleared 0 queued")
}

func TestRunner_ActionDispatch_Bad_Frozen(t *testing.T) {
	svc := New()
	svc.frozen = true
	r := svc.actionDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "agent", Value: "codex"},
	))
	core.AssertFalse(t, r.OK, "should deny when frozen")
}

func TestRunner_ActionDispatch_Good_Unfrozen(t *testing.T) {
	svc := New()
	svc.frozen = false
	r := svc.actionDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "agent", Value: "codex"},
	))
	core.AssertTrue(t, r.OK, "should allow when unfrozen and no concurrency limit hit")
}

// --- OnStartup ---

func TestRunner_Service_OnStartup_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	r := svc.OnStartup(context.Background())
	core.AssertTrue(t, r.OK)

	// Actions should be registered
	core.AssertNotNil(t, c.Action("runner.dispatch"))
	core.AssertNotNil(t, c.Action("runner.status"))
	core.AssertNotNil(t, c.Action("runner.start"))
	core.AssertNotNil(t, c.Action("runner.stop"))
	core.AssertNotNil(t, c.Action("runner.kill"))
	core.AssertNotNil(t, c.Action("runner.poke"))
}

func TestRunner_Service_OnStartup_Bad(t *testing.T) {
	svc := New()
	// No ServiceRuntime — OnStartup should panic on s.Core()
	core.AssertPanics(t, func() {
		svc.OnStartup(context.Background())
	})
}

func TestRunner_Service_OnStartup_Ugly(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.OnStartup(context.Background())
	core.AssertNotNil(t, svc.pokeCh, "runner loop should be started")
	core.AssertTrue(t, svc.IsFrozen(), "should be frozen without CORE_AGENT_DISPATCH=1")
}

// --- OnShutdown ---

func TestRunner_Service_OnShutdown_Good(t *testing.T) {
	svc := New()
	svc.frozen = false
	r := svc.OnShutdown(context.Background())
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, svc.IsFrozen())
}

func TestRunner_Service_OnShutdown_Bad(t *testing.T) {
	svc := New()
	r := svc.OnShutdown(context.Background())
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, svc.IsFrozen())
}

func TestRunner_Service_OnShutdown_Ugly(t *testing.T) {
	svc := New()
	core.AssertNotPanics(t, func() {
		svc.OnShutdown(context.Background())
	})
}

// --- HandleIPCEvents ---

func TestRunner_Service_HandleIPCEvents_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	// Unknown message type — should not panic
	r := svc.HandleIPCEvents(c, "unknown")
	core.AssertTrue(t, r.OK)
}

func TestRunner_Service_HandleIPCEvents_Bad(t *testing.T) {
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
	core.AssertTrue(t, r.OK)

	first := svc.workspaces.Get("core/go-io/task-1").Value.(*WorkspaceStatus)
	second := svc.workspaces.Get("core/go-io/task-2").Value.(*WorkspaceStatus)
	core.AssertEqual(t, "completed", first.Status)
	core.AssertEqual(t, 0, first.PID)
	core.AssertEqual(t, "running", second.Status)
	core.AssertEqual(t, 222, second.PID)
}

func TestRunner_Service_HandleIPCEvents_Ugly(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	var captured []messages.QueueDrained
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.QueueDrained); ok {
			captured = append(captured, ev)
		}
		return core.Result{OK: true}
	})

	r := svc.HandleIPCEvents(c, messages.PokeQueue{})
	core.AssertTrue(t, r.OK)
	core.AssertLen(t, captured, 1)
	core.AssertEqual(t, 0, captured[0].Completed)
}

func TestRunner_HydrateWorkspaces_Good_DeepWorkspaceName(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(wsDir)
	core.RequireTrue(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		PID:    99999999,
	}).OK)

	svc := New()
	svc.hydrateWorkspaces()

	r := svc.workspaces.Get("core/go-io/task-5")
	core.AssertTrue(t, r.OK)
	st := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "queued", st.Status)
	core.AssertEqual(t, "go-io", st.Repo)
}

// --- WriteStatus / ReadStatusResult ---

func TestRunner_WriteReadStatus_Good_Case(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", PID: 999}
	core.RequireTrue(t, WriteStatus(dir, st).OK)

	got := mustReadStatus(t, dir)
	core.AssertEqual(t, "running", got.Status)
	core.AssertEqual(t, "codex", got.Agent)
	core.AssertEqual(t, 999, got.PID)
}

func TestRunner_ReadStatus_Bad_NoFile(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.AssertTrue(t, ok)
}

func TestRunner_WriteReadStatus_Ugly_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, WriteStatus(dir, &WorkspaceStatus{Status: "running"}).OK)
	core.RequireTrue(t, WriteStatus(dir, &WorkspaceStatus{Status: "completed"}).OK)

	got := mustReadStatus(t, dir)
	core.AssertEqual(t, "completed", got.Status)
}

func TestRunner_New_Bad(t *testing.T) {
	svc := New()
	core.AssertNil(t, svc.ServiceRuntime)
	core.AssertFalse(t, svc.IsFrozen(), "New() doesn't set frozen — startRunner does")
}

func TestRunner_New_Ugly(t *testing.T) {
	a := New()
	b := New()
	assertNotSame(t, a, b, "each call returns a fresh instance")
	assertNotSame(t, a.workspaces, b.workspaces)
}

func TestRunner_Register_Bad(t *testing.T) {
	core.AssertPanics(t, func() {
		Register(nil)
	})
}

func TestRunner_Register_Ugly(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	Register(c)
	// Config should have agents.concurrency set (even if defaults)
	r := c.Config().Get("agents.dispatch")
	core.AssertNotNil(t, r)
}

func TestRunner_Service_IsFrozen_Good(t *testing.T) {
	svc := New()
	svc.frozen = true
	core.AssertTrue(t, svc.IsFrozen())
}

func TestRunner_Service_IsFrozen_Bad(t *testing.T) {
	svc := New()
	svc.frozen = false
	core.AssertFalse(t, svc.IsFrozen())
}

func TestRunner_Service_IsFrozen_Ugly(t *testing.T) {
	svc := New()
	for i := range 100 {
		svc.frozen = i%2 == 0
	}
	// i=99 → 99%2==1 → false. Last write wins.
	core.AssertFalse(t, svc.IsFrozen(), "last toggle wins")
}

func TestRunner_Service_Poke_Good(t *testing.T) {
	svc := New()
	svc.pokeCh = make(chan struct{}, 1)
	svc.Poke()
	core.AssertLen(t, svc.pokeCh, 1)
}

func TestRunner_Service_Poke_Bad(t *testing.T) {
	svc := New()
	core.AssertNotPanics(t, func() {
		svc.Poke()
	})
}

func TestRunner_Service_Poke_Ugly(t *testing.T) {
	svc := New()
	svc.pokeCh = make(chan struct{}, 1)
	svc.Poke()
	svc.Poke() // second poke is a no-op (channel full)
	core.AssertLen(t, svc.pokeCh, 1)
}

func TestRunner_Service_TrackWorkspace_Good(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("core/go-io/dev", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 12345,
	})
	r := svc.workspaces.Get("core/go-io/dev")
	core.AssertTrue(t, r.OK)
}

func TestRunner_Service_TrackWorkspace_Bad(t *testing.T) {
	svc := &Service{}
	core.AssertNotPanics(t, func() {
		svc.TrackWorkspace("test", &WorkspaceStatus{Status: "running"})
	})
}

func TestRunner_Service_TrackWorkspace_Ugly(t *testing.T) {
	svc := New()
	// TrackWorkspace accepts any — JSON round-trip converts
	svc.TrackWorkspace("test", map[string]any{
		"status": "running", "agent": "codex", "repo": "go-io",
	})
	r := svc.workspaces.Get("test")
	core.AssertTrue(t, r.OK)
	ws := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "running", ws.Status)
	core.AssertEqual(t, "codex", ws.Agent)
}

func TestRunner_Service_Workspaces_Good(t *testing.T) {
	svc := New()
	core.AssertNotNil(t, svc.Workspaces())
	core.AssertEqual(t, 0, svc.Workspaces().Len())
}

func TestRunner_Service_Workspaces_Bad(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	core.AssertEqual(t, 1, svc.Workspaces().Len())
}

func TestRunner_Service_Workspaces_Ugly(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "completed"})
	core.AssertEqual(t, 1, svc.Workspaces().Len())
	r := svc.workspaces.Get("ws-1")
	ws := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "completed", ws.Status)
}
