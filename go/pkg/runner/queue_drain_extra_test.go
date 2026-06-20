// SPDX-License-Identifier: EUPL-1.2

// drainOne deep-path coverage. The existing drainQueue/drainOne tests only
// reach the "no workspace root → finds nothing → false" leg (4% covered).
// The dispatch body — a seeded *queued* workspace discovered on disk, the
// concurrency-limit gate, and the spawn through the agentic service — was
// untested.
//
// Seam: CORE_WORKSPACE roots agentic.WorkspaceStatusPaths(), so a temp dir
// with a depth-1 workspace + a "queued" status.json is discovered by
// drainOne. A fake "agentic" service satisfying the SpawnFromQueue spawner
// interface (the same seam drainOne resolves via s.Core().Service("agentic"))
// stands in for the real container spawn — no real process. This is the
// fake-service discipline the task mandates, applied to the dispatch surface.

package runner

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
)

// fakeSpawner is a stand-in agentic service exposing only the SpawnFromQueue
// method drainOne's anonymous spawner interface requires.
type fakeSpawner struct {
	pid      int
	ok       bool
	badPID   bool // return a non-int Ok value (malformed envelope)
	calls    int
	lastDir  string
	lastAgnt string
}

func (f *fakeSpawner) SpawnFromQueue(agent, _ /*prompt*/, workspaceDir string) core.Result {
	f.calls++
	f.lastDir = workspaceDir
	f.lastAgnt = agent
	if !f.ok {
		return core.Fail(core.E("fakeSpawner", "spawn declined", nil))
	}
	if f.badPID {
		// Mux returns a malformed envelope: Ok but the value isn't the int
		// pid drainOne's `pid, ok := spawnResult.Value.(int)` expects.
		return core.Ok("not-an-int")
	}
	return core.Ok(f.pid)
}

// wrongTypeAgentic is an "agentic" service instance that does NOT implement
// the SpawnFromQueue spawner interface, so drainOne's
// `agenticService, ok := agenticResult.Value.(spawner)` assertion fails.
type wrongTypeAgentic struct{}

// seedQueuedWorkspace creates a depth-1 workspace dir under CORE_WORKSPACE
// with a "queued" status.json (so WorkspaceStatusPaths discovers it) and
// returns its absolute dir.
func seedQueuedWorkspace(t *testing.T, agent, task string) string {
	t.Helper()
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	// WorkspaceRoot() appends "/workspace" to CORE_WORKSPACE, so seed the
	// depth-1 workspace UNDER the resolved root (not the raw env value).
	wsDir := core.PathJoin(agentic.WorkspaceRoot(), "go-io-task-1")
	core.AssertTrue(t, core.MkdirAll(wsDir, 0o755).OK)
	core.AssertTrue(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status: "queued", Agent: agent, Task: task, Repo: "go-io",
	}).OK)
	// Sanity: discovery actually finds it (depth-1 status.json).
	found := agentic.WorkspaceStatusPaths()
	core.AssertEqual(t, 1, len(found))
	return wsDir
}

// coreRunner builds a Core-backed runner Service + registers a fake agentic
// spawner, returning both. ServiceRuntime is non-nil so drainOne reaches the
// spawn path (it bails at "if s.ServiceRuntime == nil { continue }" otherwise).
func coreRunner(t *testing.T, spawn *fakeSpawner) *Service {
	t.Helper()
	c := core.New(core.WithOption("name", "runner-test"))
	core.AssertTrue(t, c.RegisterService("agentic", spawn).OK)
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	return svc
}

// TestQueue_drainOne_Good_SpawnsQueued — a queued workspace with no
// concurrency limit set is dispatched: canDispatchAgent passes, the fake
// agentic SpawnFromQueue returns a pid, and the status.json is rewritten to
// "running" with that pid + Runs incremented. drainOne returns true.
func TestQueue_drainOne_Good_SpawnsQueued(t *testing.T) {
	wsDir := seedQueuedWorkspace(t, "codex", "fix the thing")
	spawn := &fakeSpawner{pid: 4242, ok: true}
	svc := coreRunner(t, spawn)

	core.AssertTrue(t, svc.drainOne())
	core.AssertEqual(t, 1, spawn.calls)
	core.AssertEqual(t, wsDir, spawn.lastDir)
	core.AssertEqual(t, "codex", spawn.lastAgnt)

	// Status flipped to running with the spawned pid.
	r := ReadStatusResult(wsDir)
	core.AssertTrue(t, r.OK)
	st, _ := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "running", st.Status)
	core.AssertEqual(t, 4242, st.PID)
	core.AssertEqual(t, 1, st.Runs)
}

// TestQueue_drainOne_ConcurrencyBlocked_SkipsSpawn — with a concurrency
// limit of total:1 for codex AND one codex workspace already tracked as
// running, canDispatchAgent returns false, so drainOne skips the queued row
// without spawning and returns false (nothing dispatched). Exercises the
// concurrency-limit gate branch.
func TestQueue_drainOne_ConcurrencyBlocked_SkipsSpawn(t *testing.T) {
	_ = seedQueuedWorkspace(t, "codex", "blocked task")
	spawn := &fakeSpawner{pid: 1, ok: true}
	svc := coreRunner(t, spawn)

	// Concurrency: codex total 1.
	svc.Core().Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"codex": {Total: 1},
	})
	// One codex workspace already running (PID<0 counts as running without
	// a live-process probe).
	svc.TrackWorkspace("core/go-io/already", &WorkspaceStatus{
		Status: "running", Agent: "codex", PID: -1,
	})

	core.AssertFalse(t, svc.drainOne())
	core.AssertEqual(t, 0, spawn.calls)
}

// TestQueue_drainOne_SpawnFails_NoStatusFlip — the queued row is dispatched
// but the fake agentic SpawnFromQueue declines; drainOne logs + continues
// (no status flip, no true return). Covers the spawn-failure branch.
func TestQueue_drainOne_SpawnFails_NoStatusFlip(t *testing.T) {
	wsDir := seedQueuedWorkspace(t, "codex", "spawn will fail")
	spawn := &fakeSpawner{ok: false}
	svc := coreRunner(t, spawn)

	core.AssertFalse(t, svc.drainOne())
	core.AssertEqual(t, 1, spawn.calls)

	// Status stays queued — the spawn failed before the flip.
	r := ReadStatusResult(wsDir)
	core.AssertTrue(t, r.OK)
	st, _ := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "queued", st.Status)
}

// TestQueue_drainOne_AgenticMissing_Skips — a queued row with NO agentic
// service registered hits the "agentic service not found" branch and
// continues (returns false). Exercises the service-lookup-miss leg.
func TestQueue_drainOne_AgenticMissing_Skips(t *testing.T) {
	_ = seedQueuedWorkspace(t, "codex", "no agentic")
	c := core.New(core.WithOption("name", "runner-test"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	core.AssertFalse(t, svc.drainOne())
}

// TestQueue_drainOne_AgenticWrongType_Skips — the malformed-envelope leg on
// the service side: the registered "agentic" instance resolves (OK) but does
// NOT implement SpawnFromQueue, so drainOne's
// `agenticService, ok := agenticResult.Value.(spawner)` assertion fails and
// the "unexpected type" branch continues (returns false).
func TestQueue_drainOne_AgenticWrongType_Skips(t *testing.T) {
	_ = seedQueuedWorkspace(t, "codex", "wrong-type agentic")
	c := core.New(core.WithOption("name", "runner-test"))
	core.AssertTrue(t, c.RegisterService("agentic", &wrongTypeAgentic{}).OK)
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	core.AssertFalse(t, svc.drainOne())
}

// TestQueue_drainOne_SpawnNonIntPID_Skips — the malformed-envelope leg on
// the spawn result: SpawnFromQueue returns Ok but with a non-int value, so
// drainOne's `pid, ok := spawnResult.Value.(int)` assertion fails and the
// "non-int pid" branch continues. The status stays queued (no flip).
func TestQueue_drainOne_SpawnNonIntPID_Skips(t *testing.T) {
	wsDir := seedQueuedWorkspace(t, "codex", "non-int pid")
	spawn := &fakeSpawner{ok: true, badPID: true}
	svc := coreRunner(t, spawn)

	core.AssertFalse(t, svc.drainOne())
	core.AssertEqual(t, 1, spawn.calls)

	r := ReadStatusResult(wsDir)
	core.AssertTrue(t, r.OK)
	st, _ := r.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "queued", st.Status)
}
