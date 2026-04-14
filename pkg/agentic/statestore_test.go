// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
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
