// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- planTaskValue ---

func TestPlanValue_PlanTaskValue_Good_TypedPassthrough(t *testing.T) {
	in := PlanTask{ID: "t1", Title: "build"}
	got, ok := planTaskValue(in)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "t1", got.ID)
	core.AssertEqual(t, "build", got.Title)
}

func TestPlanValue_PlanTaskValue_Good_MapAllFields(t *testing.T) {
	got, ok := planTaskValue(map[string]any{
		"id":          "t9",
		"title":       "ship it",
		"description": "do the thing",
		"priority":    "high",
		"category":    "build",
		"status":      "pending",
		"notes":       "careful",
		"file":        "plan.go",
		"line":        42,
	})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "t9", got.ID)
	core.AssertEqual(t, "ship it", got.Title)
	core.AssertEqual(t, "do the thing", got.Description)
	core.AssertEqual(t, "high", got.Priority)
	core.AssertEqual(t, "build", got.Category)
	core.AssertEqual(t, "pending", got.Status)
	core.AssertEqual(t, "careful", got.Notes)
	core.AssertEqual(t, "plan.go", got.File)
	core.AssertEqual(t, 42, got.Line)
	core.AssertEqual(t, "plan.go", got.FileRef)
	core.AssertEqual(t, 42, got.LineRef)
}

func TestPlanValue_PlanTaskValue_Good_NameAndRefAliases(t *testing.T) {
	got, ok := planTaskValue(map[string]any{
		"name":     "via name",
		"file_ref": "ref.go",
		"line_ref": 7,
	})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "via name", got.Title)
	core.AssertEqual(t, "ref.go", got.File)
	core.AssertEqual(t, 7, got.Line)
}

func TestPlanValue_PlanTaskValue_Bad_MapNoTitle(t *testing.T) {
	_, ok := planTaskValue(map[string]any{"status": "pending"})
	core.AssertFalse(t, ok)
}

func TestPlanValue_PlanTaskValue_Good_MapStringString(t *testing.T) {
	got, ok := planTaskValue(map[string]string{"title": "strmap"})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "strmap", got.Title)
}

func TestPlanValue_PlanTaskValue_Good_PlainString(t *testing.T) {
	got, ok := planTaskValue("just a title")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "just a title", got.Title)
}

func TestPlanValue_PlanTaskValue_Good_JSONObjectString(t *testing.T) {
	got, ok := planTaskValue(`{"title":"from json","status":"done"}`)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "from json", got.Title)
	core.AssertEqual(t, "done", got.Status)
}

func TestPlanValue_PlanTaskValue_Bad_EmptyString(t *testing.T) {
	_, ok := planTaskValue("   ")
	core.AssertFalse(t, ok)
}

func TestPlanValue_PlanTaskValue_Bad_UnsupportedType(t *testing.T) {
	_, ok := planTaskValue(12345)
	core.AssertFalse(t, ok)
}

// --- planTaskSliceValue ---

func TestPlanValue_PlanTaskSliceValue_Good_TypedSlice(t *testing.T) {
	in := []PlanTask{{Title: "a"}, {Title: "b"}}
	got := planTaskSliceValue(in)
	core.AssertEqual(t, 2, len(got))
}

func TestPlanValue_PlanTaskSliceValue_Good_StringSlice(t *testing.T) {
	got := planTaskSliceValue([]string{"one", "", "two"})
	core.AssertEqual(t, 2, len(got))
	core.AssertEqual(t, "one", got[0].Title)
	core.AssertEqual(t, "two", got[1].Title)
}

func TestPlanValue_PlanTaskSliceValue_Good_AnySlice(t *testing.T) {
	got := planTaskSliceValue([]any{"x", map[string]any{"title": "y"}})
	core.AssertEqual(t, 2, len(got))
}

func TestPlanValue_PlanTaskSliceValue_Good_MapSlice(t *testing.T) {
	got := planTaskSliceValue([]map[string]any{{"title": "m1"}, {"status": "no-title"}})
	core.AssertEqual(t, 1, len(got))
	core.AssertEqual(t, "m1", got[0].Title)
}

func TestPlanValue_PlanTaskSliceValue_Good_JSONArrayOfObjects(t *testing.T) {
	got := planTaskSliceValue(`[{"title":"j1"},{"title":"j2"}]`)
	core.AssertEqual(t, 2, len(got))
}

func TestPlanValue_PlanTaskSliceValue_Good_JSONArrayOfStrings(t *testing.T) {
	got := planTaskSliceValue(`["s1","s2"]`)
	core.AssertEqual(t, 2, len(got))
	core.AssertEqual(t, "s1", got[0].Title)
}

func TestPlanValue_PlanTaskSliceValue_Good_SingleStringFallback(t *testing.T) {
	got := planTaskSliceValue("lonely")
	core.AssertEqual(t, 1, len(got))
	core.AssertEqual(t, "lonely", got[0].Title)
}

func TestPlanValue_PlanTaskSliceValue_Ugly_EmptyString(t *testing.T) {
	got := planTaskSliceValue("")
	core.AssertEqual(t, 0, len(got))
}

func TestPlanValue_PlanTaskSliceValue_Bad_UnsupportedReturnsNil(t *testing.T) {
	got := planTaskSliceValue(3.14)
	core.AssertEqual(t, 0, len(got))
}

// --- phaseCheckpointValue ---

func TestPlanValue_PhaseCheckpointValue_Good_TypedWithNote(t *testing.T) {
	got, ok := phaseCheckpointValue(PhaseCheckpoint{Note: "passes"})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "passes", got.Note)
}

func TestPlanValue_PhaseCheckpointValue_Bad_TypedNoNote(t *testing.T) {
	_, ok := phaseCheckpointValue(PhaseCheckpoint{})
	core.AssertFalse(t, ok)
}

func TestPlanValue_PhaseCheckpointValue_Good_Map(t *testing.T) {
	got, ok := phaseCheckpointValue(map[string]any{
		"note":       "build green",
		"created_at": "2026-03-31T00:00:00Z",
		"context":    map[string]any{"sha": "abc"},
	})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "build green", got.Note)
	core.AssertEqual(t, "2026-03-31T00:00:00Z", got.CreatedAt)
	core.AssertEqual(t, "abc", got.Context["sha"])
}

func TestPlanValue_PhaseCheckpointValue_Bad_MapNoNote(t *testing.T) {
	_, ok := phaseCheckpointValue(map[string]any{"created_at": "now"})
	core.AssertFalse(t, ok)
}

func TestPlanValue_PhaseCheckpointValue_Good_MapStringString(t *testing.T) {
	got, ok := phaseCheckpointValue(map[string]string{"note": "ok"})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "ok", got.Note)
}

func TestPlanValue_PhaseCheckpointValue_Good_PlainString(t *testing.T) {
	got, ok := phaseCheckpointValue("a note")
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "a note", got.Note)
}

func TestPlanValue_PhaseCheckpointValue_Good_JSONObjectString(t *testing.T) {
	got, ok := phaseCheckpointValue(`{"note":"jnote"}`)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, "jnote", got.Note)
}

func TestPlanValue_PhaseCheckpointValue_Bad_EmptyString(t *testing.T) {
	_, ok := phaseCheckpointValue("  ")
	core.AssertFalse(t, ok)
}

func TestPlanValue_PhaseCheckpointValue_Bad_UnsupportedType(t *testing.T) {
	_, ok := phaseCheckpointValue(99)
	core.AssertFalse(t, ok)
}

// --- phaseCheckpointSliceValue ---

func TestPlanValue_PhaseCheckpointSliceValue_Good_TypedSlice(t *testing.T) {
	in := []PhaseCheckpoint{{Note: "a"}, {Note: "b"}}
	got := phaseCheckpointSliceValue(in)
	core.AssertEqual(t, 2, len(got))
}

func TestPlanValue_PhaseCheckpointSliceValue_Good_AnySlice(t *testing.T) {
	got := phaseCheckpointSliceValue([]any{"note1", map[string]any{"note": "note2"}})
	core.AssertEqual(t, 2, len(got))
}

func TestPlanValue_PhaseCheckpointSliceValue_Good_MapSlice(t *testing.T) {
	got := phaseCheckpointSliceValue([]map[string]any{{"note": "m1"}, {"created_at": "no-note"}})
	core.AssertEqual(t, 1, len(got))
}

func TestPlanValue_PhaseCheckpointSliceValue_Good_JSONArrayOfObjects(t *testing.T) {
	got := phaseCheckpointSliceValue(`[{"note":"j1"},{"note":"j2"}]`)
	core.AssertEqual(t, 2, len(got))
}

func TestPlanValue_PhaseCheckpointSliceValue_Good_SingleFallback(t *testing.T) {
	got := phaseCheckpointSliceValue("only")
	core.AssertEqual(t, 1, len(got))
	core.AssertEqual(t, "only", got[0].Note)
}

func TestPlanValue_PhaseCheckpointSliceValue_Ugly_EmptyString(t *testing.T) {
	got := phaseCheckpointSliceValue("")
	core.AssertEqual(t, 0, len(got))
}

func TestPlanValue_PhaseCheckpointSliceValue_Bad_UnsupportedReturnsNil(t *testing.T) {
	got := phaseCheckpointSliceValue(1.5)
	core.AssertEqual(t, 0, len(got))
}

// --- phaseValue: Tasks + Checkpoints branches ---

func TestPlanValue_PhaseValue_Good_WithTasksAndCheckpoints(t *testing.T) {
	got, ok := phaseValue(map[string]any{
		"number":      2,
		"name":        "Phase Two",
		"status":      "active",
		"tasks":       []any{map[string]any{"title": "task-a"}},
		"checkpoints": []any{map[string]any{"note": "cp-a"}},
		"tests":       5,
		"notes":       "phase notes",
	})
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 2, got.Number)
	core.AssertEqual(t, "Phase Two", got.Name)
	core.AssertEqual(t, 1, len(got.Tasks))
	core.AssertEqual(t, "task-a", got.Tasks[0].Title)
	core.AssertEqual(t, 1, len(got.Checkpoints))
	core.AssertEqual(t, "cp-a", got.Checkpoints[0].Note)
	core.AssertEqual(t, 5, got.Tests)
	core.AssertEqual(t, "phase notes", got.Notes)
}

// --- phaseSliceValue: map-slice + single fallback ---

func TestPlanValue_PhaseSliceValue_Good_MapSlice(t *testing.T) {
	got := phaseSliceValue([]map[string]any{
		{"number": 1, "name": "P1"},
		{"number": 2, "name": "P2"},
	})
	core.AssertEqual(t, 2, len(got))
	core.AssertEqual(t, "P1", got[0].Name)
}

func TestPlanValue_PhaseSliceValue_Good_SingleMapFallback(t *testing.T) {
	got := phaseSliceValue(map[string]any{"number": 9, "name": "solo"})
	core.AssertEqual(t, 1, len(got))
	core.AssertEqual(t, "solo", got[0].Name)
}

func TestPlanValue_PhaseSliceValue_Ugly_EmptyString(t *testing.T) {
	got := phaseSliceValue("")
	core.AssertEqual(t, 0, len(got))
}

func TestPlanValue_PhaseSliceValue_Good_JSONArrayString(t *testing.T) {
	got := phaseSliceValue(`[{"number":1,"name":"jp1"}]`)
	core.AssertEqual(t, 1, len(got))
	core.AssertEqual(t, "jp1", got[0].Name)
}
