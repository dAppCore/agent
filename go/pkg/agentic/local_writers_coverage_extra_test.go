// SPDX-License-Identifier: EUPL-1.2

// Extra coverage for the local-file write helpers and the cmdState
// dispatcher. The write helpers are void/Result-returning persisters whose
// success arm (EnsureDir + WriteAtomic) was unexercised; each test drives
// the success path under a temp workspace and asserts the observable effect
// via the matching read helper. Failure arms need fs fault injection that
// does not exist here, so they are intentionally not covered.

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- cmdState dispatcher ---------------------------------------------

// TestCommandsState_CmdState_Usage_NoAction — no action prints usage and OK.
func TestCommandsState_CmdState_Usage_NoAction(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	s := newTestPrep(t)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdState(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent state")
}

// TestCommandsState_CmdState_Unknown_Action — an unrecognised action prints
// usage and returns the unknown-command error.
func TestCommandsState_CmdState_Unknown_Action(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	s := newTestPrep(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdState(core.NewOptions(core.Option{Key: "action", Value: "frobnicate"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent state")
	core.AssertContains(t, r.Value.(error).Error(), "unknown state command: frobnicate")
}

// TestCommandsState_CmdState_RoutesByAction — the dispatcher routes set / get
// / list / delete to their sub-handlers (local plan-state files).
func TestCommandsState_CmdState_RoutesByAction(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	s := newTestPrep(t)

	captureStdout(t, func() {
		set := s.cmdState(core.NewOptions(
			core.Option{Key: "action", Value: "set"},
			core.Option{Key: "_arg", Value: "plan-a"},
			core.Option{Key: "key", Value: "pattern"},
			core.Option{Key: "value", Value: "observer"},
		))
		core.AssertTrue(t, set.OK)

		get := s.cmdState(core.NewOptions(
			core.Option{Key: "action", Value: "get"},
			core.Option{Key: "_arg", Value: "plan-a"},
			core.Option{Key: "key", Value: "pattern"},
		))
		core.AssertTrue(t, get.OK)

		list := s.cmdState(core.NewOptions(
			core.Option{Key: "action", Value: "list"},
			core.Option{Key: "_arg", Value: "plan-a"},
		))
		core.AssertTrue(t, list.OK)

		del := s.cmdState(core.NewOptions(
			core.Option{Key: "action", Value: "delete"},
			core.Option{Key: "_arg", Value: "plan-a"},
			core.Option{Key: "key", Value: "pattern"},
		))
		core.AssertTrue(t, del.OK)
	})
}

// --- writePlanStates -------------------------------------------------

func TestLocalWriters_writePlanStates_Good_RoundTrip(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	states := []WorkspaceState{{Key: "pattern", Value: "observer", Type: "general"}}

	r := writePlanStates("plan-a", states)
	core.RequireTrue(t, r.OK)

	read := readPlanStates("plan-a")
	core.RequireTrue(t, read.OK)
	got, ok := read.Value.([]WorkspaceState)
	core.RequireTrue(t, ok)
	core.AssertLen(t, got, 1)
	core.AssertEqual(t, "pattern", got[0].Key)
}

// --- writePlanResult -------------------------------------------------

func TestLocalWriters_writePlanResult_Bad_NilPlan(t *testing.T) {
	r := writePlanResult(t.TempDir(), nil)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "plan is required")
}

func TestLocalWriters_writePlanResult_Good_WritesFile(t *testing.T) {
	dir := t.TempDir()
	plan := &Plan{ID: "core-plan-1", Title: "P", Status: "draft", Objective: "o"}

	r := writePlanResult(dir, plan)
	core.RequireTrue(t, r.OK)
	path, ok := r.Value.(string)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, fs.Exists(path))
}

// --- writePromptSnapshot ---------------------------------------------

func TestLocalWriters_writePromptSnapshot_EmptyInput_NoOp(t *testing.T) {
	// Empty workspace dir or blank prompt is a no-op success.
	core.AssertTrue(t, writePromptSnapshot("", "prompt").OK)
	core.AssertTrue(t, writePromptSnapshot(t.TempDir(), "   ").OK)
}

func TestLocalWriters_writePromptSnapshot_Good_WritesAndSkipsExisting(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	ws := core.JoinPath(t.TempDir(), "workspace", "core", "go-io", "feature", "x")

	// First write creates the snapshot.
	r1 := writePromptSnapshot(ws, "the prompt body")
	core.RequireTrue(t, r1.OK)

	// Second write with the same content hits the "already exists" skip arm.
	r2 := writePromptSnapshot(ws, "the prompt body")
	core.RequireTrue(t, r2.OK)
}

// --- writeSync* (void persisters) ------------------------------------

func TestLocalWriters_writeSyncContext_Good_RoundTrip(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	writeSyncContext([]map[string]any{{"id": "m1", "text": "hello"}})

	got := readSyncContext()
	core.AssertLen(t, got, 1)
	core.AssertEqual(t, "m1", got[0]["id"])
}

func TestLocalWriters_writeSyncRecords_Good_RoundTrip(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	writeSyncRecords([]SyncRecord{{Direction: "push", ItemsCount: 3, SyncedAt: "2026-01-01"}})

	got := readSyncRecords()
	core.AssertLen(t, got, 1)
	core.AssertEqual(t, "push", got[0].Direction)
}

func TestLocalWriters_writeSyncStatusState_Good_RoundTrip(t *testing.T) {
	setTestWorkspace(t, t.TempDir())
	now := core.Now()
	writeSyncStatusState(syncStatusState{LastPushAt: now})

	got := readSyncStatusState()
	core.AssertFalse(t, got.LastPushAt.IsZero())
}

func TestLocalWriters_writeSyncLedger_Good_WriteThenDelete(t *testing.T) {
	setTestWorkspace(t, t.TempDir())

	// Non-empty ledger writes the file.
	writeSyncLedger(map[string]string{"ws-1": "2026-01-01#2"})
	core.AssertEqual(t, "2026-01-01#2", readSyncLedger()["ws-1"])

	// Empty ledger deletes it (the delete arm).
	writeSyncLedger(map[string]string{})
	core.AssertLen(t, readSyncLedger(), 0)
}
