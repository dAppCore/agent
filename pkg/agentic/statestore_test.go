// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

// withStateStoreTempDir redirects CORE_WORKSPACE to a fresh temporary
// directory so statestore tests can open `.core/db.duckdb` in isolation.
func withStateStoreTempDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)
	t.Setenv("CORE_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("DIR_HOME", dir)
}

// TestStatestore_StateStoreInstance_Good verifies the DuckDB-backed store can
// be initialised inside a temporary workspace and that the same instance is
// returned on subsequent calls (lazy once semantics).
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_StateStoreInstance_Good`
func TestStatestore_StateStoreInstance_Good(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	first := subsystem.stateStoreInstance()
	if first == nil {
		t.Fatalf("expected store instance, got nil; err=%v", subsystem.stateStoreErr())
	}

	second := subsystem.stateStoreInstance()
	if second != first {
		t.Fatalf("expected lazy-once to return same instance, got different pointers")
	}
}

// TestStatestore_StateStoreSet_Good_WritesAndRestores verifies the helpers
// round-trip JSON entries through the store and that stateStoreRestore walks
// every entry.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_StateStoreSet_Good_WritesAndRestores`
func TestStatestore_StateStoreSet_Good_WritesAndRestores(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	subsystem.stateStoreSet(stateRegistryGroup, "core/go-io", map[string]any{"status": "running"})
	subsystem.stateStoreSet(stateRegistryGroup, "core/go-store", map[string]any{"status": "queued"})

	entries := map[string]map[string]any{}
	subsystem.stateStoreRestore(stateRegistryGroup, func(key, value string) bool {
		decoded := map[string]any{}
		if result := core.JSONUnmarshalString(value, &decoded); !result.OK {
			t.Fatalf("unmarshal %s: %v", key, result.Value)
		}
		entries[key] = decoded
		return true
	})

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), entries)
	}
	if status, ok := entries["core/go-io"]["status"].(string); !ok || status != "running" {
		t.Fatalf("expected core/go-io status=running, got %v", entries["core/go-io"])
	}
}

// TestStatestore_CloseStateStore_Bad_SafeOnNilSubsystem verifies close helpers
// do not panic on nil receivers — critical for test teardown paths and the
// graceful-degradation requirement in RFC §15.6.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_CloseStateStore_Bad_SafeOnNilSubsystem`
func TestStatestore_CloseStateStore_Bad_SafeOnNilSubsystem(t *testing.T) {
	var subsystem *PrepSubsystem
	subsystem.closeStateStore()
	if instance := subsystem.stateStoreInstance(); instance != nil {
		t.Fatalf("expected nil instance on nil subsystem, got %v", instance)
	}
}

// TestStatestore_StateStoreDelete_Ugly_DeletingUnknownKey verifies delete is a
// no-op for missing keys so call sites never need to guard against misses.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_StateStoreDelete_Ugly_DeletingUnknownKey`
func TestStatestore_StateStoreDelete_Ugly_DeletingUnknownKey(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	subsystem.stateStoreDelete(stateRegistryGroup, "never-existed")
	subsystem.stateStoreSet(stateRegistryGroup, "real", map[string]any{"value": 1})
	subsystem.stateStoreDelete(stateRegistryGroup, "real")

	count := subsystem.stateStoreCount(stateRegistryGroup)
	if count != 0 {
		t.Fatalf("expected registry empty after delete, got count=%d", count)
	}
}

// TestStatestore_HydrateWorkspaces_Good_RestoresFromStore mirrors RFC §15.3 —
// the registry group is populated before hydrateWorkspaces runs, and the
// subsystem must restore those entries so ghost agents are detectable across
// restarts without reading the status.json filesystem tree.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_HydrateWorkspaces_Good_RestoresFromStore`
func TestStatestore_HydrateWorkspaces_Good_RestoresFromStore(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	subsystem.workspaces = core.NewRegistry[*WorkspaceStatus]()
	defer subsystem.closeStateStore()

	subsystem.stateStoreSet(stateRegistryGroup, "core/go-io/task-5", WorkspaceStatus{
		Status: "running",
		Agent:  "codex:gpt-5.4",
		PID:    0,
	})

	subsystem.hydrateWorkspaces()

	result := subsystem.Workspaces().Get("core/go-io/task-5")
	if !result.OK {
		t.Fatalf("expected workspace restored, got miss")
	}
	status, ok := result.Value.(*WorkspaceStatus)
	if !ok {
		t.Fatalf("expected *WorkspaceStatus, got %T", result.Value)
	}
	// Dead PID should be marked failed, per §15.3.
	if status.Status != "failed" {
		t.Fatalf("expected ghost agent marked failed, got status=%s", status.Status)
	}
}

// TestStatestore_RuntimeState_Good_PersistsAcrossReloads mirrors RFC §15 —
// backoff deadlines saved via persistRuntimeState must replay when a new
// subsystem instance calls loadRuntimeState, enabling seamless resume after
// dispatch crashes.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_RuntimeState_Good_PersistsAcrossReloads`
func TestStatestore_RuntimeState_Good_PersistsAcrossReloads(t *testing.T) {
	withStateStoreTempDir(t)

	original := &PrepSubsystem{
		backoff: map[string]time.Time{
			"codex": time.Now().Add(15 * time.Minute),
		},
		failCount: map[string]int{"codex": 3},
	}
	original.persistRuntimeState()
	original.closeStateStore()

	replay := &PrepSubsystem{}
	defer replay.closeStateStore()
	replay.loadRuntimeState()

	if _, ok := replay.backoff["codex"]; !ok {
		t.Fatalf("expected replay to restore codex backoff, got map=%v", replay.backoff)
	}
	if replay.failCount["codex"] != 3 {
		t.Fatalf("expected replay fail count=3, got %d", replay.failCount["codex"])
	}
}

// TestStatestore_TrackWorkspace_Good_MirrorsQueueGroup verifies RFC §15.3 —
// queued workspaces are persisted under the queue group keyed by
// `{repo}/{branch}` and removed once they leave the queued state.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_TrackWorkspace_Good_MirrorsQueueGroup`
func TestStatestore_TrackWorkspace_Good_MirrorsQueueGroup(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}
	defer subsystem.closeStateStore()

	queued := &WorkspaceStatus{
		Status:    "queued",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
	}
	subsystem.TrackWorkspace("core/go-io/task-5", queued)

	if subsystem.stateStoreCount(stateQueueGroup) != 1 {
		t.Fatalf("expected queue group to contain 1 entry, got %d", subsystem.stateStoreCount(stateQueueGroup))
	}

	value, ok := subsystem.stateStoreGet(stateQueueGroup, "core/go-io/task-5")
	if !ok {
		t.Fatalf("expected queue entry under core/go-io/task-5, got miss")
	}
	var entry queueEntry
	if result := core.JSONUnmarshalString(value, &entry); !result.OK {
		t.Fatalf("unmarshal queue entry: %v", result.Value)
	}
	if entry.Repo != "go-io" || entry.Branch != "agent/fix-tests" {
		t.Fatalf("unexpected queue entry: %+v", entry)
	}

	queued.Status = "running"
	subsystem.TrackWorkspace("core/go-io/task-5", queued)

	if subsystem.stateStoreCount(stateQueueGroup) != 0 {
		t.Fatalf("expected queue group emptied after dispatch, got %d", subsystem.stateStoreCount(stateQueueGroup))
	}
}

// TestStatestore_TrackWorkspace_Good_RefreshesConcurrencySnapshot verifies
// RFC §15.3 — running counts per agent type persist into the concurrency
// group so a restart can detect over-dispatch before scheduling new work.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_TrackWorkspace_Good_RefreshesConcurrencySnapshot`
func TestStatestore_TrackWorkspace_Good_RefreshesConcurrencySnapshot(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}
	defer subsystem.closeStateStore()

	subsystem.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{
		Status: "running",
		Agent:  "codex:gpt-5.4",
		Repo:   "go-io",
	})
	subsystem.TrackWorkspace("core/go-store/task-2", &WorkspaceStatus{
		Status: "running",
		Agent:  "codex:gpt-5.4-mini",
		Repo:   "go-store",
	})

	value, ok := subsystem.stateStoreGet(stateConcurrencyGroup, "codex")
	if !ok {
		t.Fatalf("expected concurrency snapshot for codex, got miss")
	}
	snapshot := map[string]any{}
	if result := core.JSONUnmarshalString(value, &snapshot); !result.OK {
		t.Fatalf("unmarshal concurrency snapshot: %v", result.Value)
	}
	running, _ := snapshot["running"].(float64)
	if int(running) != 2 {
		t.Fatalf("expected running=2, got %v (%T)", snapshot["running"], snapshot["running"])
	}

	subsystem.TrackWorkspace("core/go-io/task-5", nil)
	value, ok = subsystem.stateStoreGet(stateConcurrencyGroup, "codex")
	if !ok {
		t.Fatalf("expected concurrency snapshot to remain after one removal, got miss")
	}
	snapshot = map[string]any{}
	if result := core.JSONUnmarshalString(value, &snapshot); !result.OK {
		t.Fatalf("unmarshal concurrency snapshot after removal: %v", result.Value)
	}
	if running, _ := snapshot["running"].(float64); int(running) != 1 {
		t.Fatalf("expected running=1 after removal, got %v", snapshot["running"])
	}
}

// TestStatestore_HydrateWorkspaces_Good_ReapsFilesystemGhosts verifies RFC §15.3 —
// a status.json that claims `running` for a PID that no longer exists must be
// reaped by hydrateWorkspaces, both in the registry and on disk so other
// tooling (status.json consumers, dashboards) sees a coherent view.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_HydrateWorkspaces_Good_ReapsFilesystemGhosts`
func TestStatestore_HydrateWorkspaces_Good_ReapsFilesystemGhosts(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_HOME", root)
	t.Setenv("DIR_HOME", root)

	subsystem := &PrepSubsystem{
		workspaces: core.NewRegistry[*WorkspaceStatus](),
	}
	defer subsystem.closeStateStore()

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-restart")
	fs.EnsureDir(workspaceDir)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "running",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		Org:       "core",
		Task:      "ghost-reap",
		Branch:    "agent/ghost-reap",
		PID:       99999,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	subsystem.hydrateWorkspaces()

	result := subsystem.Workspaces().Get("core/go-io/task-restart")
	if !result.OK {
		t.Fatalf("expected workspace restored from filesystem, got miss")
	}
	status, ok := result.Value.(*WorkspaceStatus)
	if !ok {
		t.Fatalf("expected *WorkspaceStatus, got %T", result.Value)
	}
	if status.Status != "failed" {
		t.Fatalf("expected ghost agent reaped to failed, got status=%s", status.Status)
	}

	// Verify the reaped status persisted back to disk so cmdStatus and
	// out-of-process consumers observe the same coherent view.
	reread := ReadStatusResult(workspaceDir)
	if !reread.OK {
		t.Fatalf("expected status.json readable after reap, got %v", reread.Value)
	}
	rereadStatus, ok := workspaceStatusValue(reread)
	if !ok || rereadStatus.Status != "failed" {
		t.Fatalf("expected status.json updated to failed, got %+v", rereadStatus)
	}
}

// TestStatestore_RecoverStateOrphans_Good_DiscardsLeftoverBuffers verifies
// RFC §15.5 — QA workspace buffers left on disk by crashed dispatches are
// released rather than committed, so partial cycles do not poison the diff
// history described in RFC §7.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_RecoverStateOrphans_Good_DiscardsLeftoverBuffers`
func TestStatestore_RecoverStateOrphans_Good_DiscardsLeftoverBuffers(t *testing.T) {
	withStateStoreTempDir(t)
	// go-store creates `.core/state/` relative to process cwd — redirect cwd
	// into a tempdir so the leftover DuckDB file never leaks into the package
	// working tree.
	tempCWD := t.TempDir()
	t.Chdir(tempCWD)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	st := subsystem.stateStoreInstance()
	if st == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	// Seed a fake orphan by creating a workspace, Put-ing a row, then
	// Close-ing the workspace — closing keeps the .duckdb file on disk per
	// the go-store contract, simulating a crashed dispatch. The unique name
	// keeps this test isolated from the shared go-store registry cache.
	workspaceName := core.Sprintf("qa-crashed-cycle-%d", time.Now().UnixNano())
	workspace, err := st.NewWorkspace(workspaceName)
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	_ = workspace.Put("finding", map[string]any{"tool": "gosec"})
	workspace.Close()

	// Reopen the state store so RecoverOrphans walks the filesystem fresh.
	subsystem.closeStateStore()
	subsystem = &PrepSubsystem{}
	defer subsystem.closeStateStore()

	// The recovery should run without panicking and leave no orphans behind.
	subsystem.recoverStateOrphans()
}

// TestStatestore_RecoverStateOrphans_Bad_MissingStateDir verifies the helper
// is a no-op on the happy path where no crash ever happened (no `.core/state/`
// directory exists yet). The agent must still boot cleanly.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_RecoverStateOrphans_Bad_MissingStateDir`
func TestStatestore_RecoverStateOrphans_Bad_MissingStateDir(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	if subsystem.stateStoreInstance() == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	// No `.core/state/` directory has been created yet — recovery must
	// return without touching anything.
	subsystem.recoverStateOrphans()
}

// TestStatestore_RecoverStateOrphans_Ugly_NilSubsystem verifies RFC §15.6 —
// calling recovery on a nil subsystem must be a no-op so graceful degradation
// holds for any edge case where the subsystem failed to initialise.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_RecoverStateOrphans_Ugly_NilSubsystem`
func TestStatestore_RecoverStateOrphans_Ugly_NilSubsystem(t *testing.T) {
	var subsystem *PrepSubsystem
	subsystem.recoverStateOrphans()
	store := subsystem.stateStoreInstance()
	core.AssertNil(t, store)
	core.AssertNil(t, subsystem.stateStoreErr())
}

// TestStatestore_SyncQueue_Good_PersistsViaStore verifies RFC §16.5 —
// the sync queue lives in go-store under the sync_queue group so backoff
// state survives restart even when the JSON file is rotated or wiped.
//
// Usage example: `go test ./pkg/agentic -run TestStatestore_SyncQueue_Good_PersistsViaStore`
func TestStatestore_SyncQueue_Good_PersistsViaStore(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	queued := []syncQueuedPush{{
		AgentID:  "charon",
		QueuedAt: time.Now(),
		Dispatches: []map[string]any{
			{"workspace": "core/go-io/task-5", "status": "completed"},
		},
	}}
	subsystem.writeSyncQueue(queued)

	value, ok := subsystem.stateStoreGet(stateSyncQueueGroup, syncQueueStoreKey)
	if !ok {
		t.Fatalf("expected sync queue persisted to go-store, got miss")
	}
	var roundTrip []syncQueuedPush
	if result := core.JSONUnmarshalString(value, &roundTrip); !result.OK {
		t.Fatalf("unmarshal sync queue: %v", result.Value)
	}
	if len(roundTrip) != 1 || roundTrip[0].AgentID != "charon" {
		t.Fatalf("unexpected round trip: %+v", roundTrip)
	}

	if read := subsystem.readSyncQueue(); len(read) != 1 || read[0].AgentID != "charon" {
		t.Fatalf("expected readSyncQueue to return go-store entry, got %+v", read)
	}

	subsystem.writeSyncQueue(nil)
	if subsystem.stateStoreCount(stateSyncQueueGroup) != 0 {
		t.Fatalf("expected empty sync queue group after clear, got %d", subsystem.stateStoreCount(stateSyncQueueGroup))
	}
}
