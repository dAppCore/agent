// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

// newTestPrep creates a PrepSubsystem for testing with testCore wired in.
func newTestPrep(t *testing.T) *PrepSubsystem {
	t.Helper()
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

// --- planCreate (MCP handler) ---

func TestPlan_PlanCreate_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Migrate Core",
		Objective: "Use v0.7.0 API everywhere",
		Repo:      "go-io",
		Phases: []Phase{
			{Name: "Update imports", Criteria: []string{"All imports changed"}},
			{Name: "Run tests"},
		},
		Notes: "Priority: high",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertNotEmpty(t, out.ID)
	assertCoreIDFormat(t, out.ID)
	core.AssertNotEmpty(t, out.Path)

	core.AssertTrue(t, fs.Exists(out.Path))
}

func TestPlan_PlanCreate_Good_UniqueIDs(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, first, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Repeated Title",
		Objective: "Repeated objective",
	})
	core.RequireNoError(t, err)
	assertCoreIDFormat(t, first.ID)

	_, second, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Repeated Title",
		Objective: "Repeated objective",
	})
	core.RequireNoError(t, err)
	assertCoreIDFormat(t, second.ID)
	core.AssertNotEqual(t, first.ID, second.ID)
}

func TestPlan_PlanCreate_Bad_MissingTitle(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Objective: "something",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "title is required")
}

func TestPlan_PlanCreate_Bad_MissingObjective(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "My Plan",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "objective is required")
}

func TestPlan_PlanCreate_Good_DefaultPhaseStatus(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Test Plan",
		Objective: "Test defaults",
		Phases:    []Phase{{Name: "Phase 1"}, {Name: "Phase 2"}},
	})
	core.RequireNoError(t, err)

	plan, readErr := readPlan(PlansRoot(), out.ID)
	core.RequireNoError(t, readErr)
	core.AssertEqual(t, "pending", plan.Phases[0].Status)
	core.AssertEqual(t, "pending", plan.Phases[1].Status)
	core.AssertEqual(t, 1, plan.Phases[0].Number)
	core.AssertEqual(t, 2, plan.Phases[1].Number)
}

// --- planRead (MCP handler) ---

func TestPlan_PlanRead_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Read Test",
		Objective: "Verify read works",
	})
	core.RequireNoError(t, err)

	_, readOut, err := s.planRead(context.Background(), nil, PlanReadInput{ID: createOut.ID})
	core.RequireNoError(t, err)
	core.AssertTrue(t, readOut.Success)
	core.AssertEqual(t, createOut.ID, readOut.Plan.ID)
	core.AssertEqual(t, "Read Test", readOut.Plan.Title)
	core.AssertEqual(t, "draft", readOut.Plan.Status)
}

func TestPlan_PlanRead_Bad_MissingID(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "id is required")
}

func TestPlan_PlanRead_Bad_NotFound(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{ID: "nonexistent"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not found")
}

// --- planUpdate (MCP handler) ---

func TestPlan_PlanUpdate_Good_Status(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Update Test",
		Objective: "Verify update",
	})

	_, updateOut, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:     createOut.ID,
		Status: "ready",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, updateOut.Success)
	core.AssertEqual(t, "ready", updateOut.Plan.Status)
}

func TestPlan_PlanUpdate_Good_PartialUpdate(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Partial Update",
		Objective: "Original objective",
		Notes:     "Original notes",
	})

	_, updateOut, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:    createOut.ID,
		Title: "New Title",
		Agent: "codex",
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "New Title", updateOut.Plan.Title)
	core.AssertEqual(t, "Original objective", updateOut.Plan.Objective)
	core.AssertEqual(t, "Original notes", updateOut.Plan.Notes)
	core.AssertEqual(t, "codex", updateOut.Plan.Agent)
}

func TestPlan_PlanUpdate_Good_AllStatusTransitions(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "Status Lifecycle", Objective: "Test transitions",
	})

	transitions := []string{"ready", "in_progress", "needs_verification", "verified", "approved"}
	for _, status := range transitions {
		_, out, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
			ID: createOut.ID, Status: status,
		})
		core.RequireNoError(t, err, "transition to %s", status)
		core.AssertEqual(t, status, out.Plan.Status)
	}
}

func TestPlan_PlanUpdate_Bad_InvalidStatus(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "Bad Status", Objective: "Test",
	})

	_, _, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID: createOut.ID, Status: "invalid_status",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid status")
}

func TestPlan_PlanUpdate_Bad_MissingID(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{Status: "ready"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "id is required")
}

func TestPlan_PlanUpdate_Good_ReplacePhases(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Phase Replace",
		Objective: "Test phase replacement",
		Phases:    []Phase{{Name: "Old Phase"}},
	})

	_, updateOut, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:     createOut.ID,
		Phases: []Phase{{Number: 1, Name: "New Phase", Status: "done"}, {Number: 2, Name: "Phase 2"}},
	})
	core.RequireNoError(t, err)
	core.AssertLen(t, updateOut.Plan.Phases, 2)
	core.AssertEqual(t, "New Phase", updateOut.Plan.Phases[0].Name)
}

// --- planDelete (MCP handler) ---

func TestPlan_PlanDelete_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "Delete Me", Objective: "Will be deleted",
	})

	planBeforeDelete, err := readPlan(PlansRoot(), createOut.ID)
	core.RequireNoError(t, err)
	core.RequireTrue(t, writePlanStates(planBeforeDelete.Slug, []WorkspaceState{{
		Key:   "pattern",
		Value: "observer",
	}}).OK)

	_, delOut, err := s.planDelete(context.Background(), nil, PlanDeleteInput{
		ID:     createOut.ID,
		Reason: "No longer needed",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, delOut.Success)
	core.AssertEqual(t, createOut.ID, delOut.Deleted)

	core.AssertFalse(t, fs.Exists(createOut.Path))
	core.AssertFalse(t, fs.Exists(statePath(planBeforeDelete.Slug)))

	_, readErr := readPlan(PlansRoot(), createOut.ID)
	core.AssertError(t, readErr)
}

func TestPlan_PlanDelete_Bad_MissingID(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "id is required")
}

func TestPlan_PlanDelete_Bad_NotFound(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{ID: "nonexistent"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not found")
}

// --- planList (MCP handler) ---

func TestPlan_PlanList_Good_Empty(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 0, out.Count)
}

func TestPlan_PlanList_Good_Multiple(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "A", Objective: "A", Repo: "go-io"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "B", Objective: "B", Repo: "go-crypt"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "C", Objective: "C", Repo: "go-io"})

	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 3, out.Count)
}

func TestPlan_PlanList_Good_FilterByRepo(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "A", Objective: "A", Repo: "go-io"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "B", Objective: "B", Repo: "go-crypt"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "C", Objective: "C", Repo: "go-io"})

	_, out, err := s.planList(context.Background(), nil, PlanListInput{Repo: "go-io"})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 2, out.Count)
}

func TestPlan_HandlePlanCreate_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	result := s.handlePlanCreate(context.Background(), core.NewOptions(
		core.Option{Key: "title", Value: "Named plan action"},
		core.Option{Key: "objective", Value: "Expose plan CRUD as named actions"},
		core.Option{Key: "repo", Value: "agent"},
		core.Option{Key: "phases", Value: []any{
			map[string]any{
				"name":     "Register actions",
				"criteria": []any{"plan.create exists", "tests cover handlers"},
				"tests":    2,
			},
		}},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(PlanCreateOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	assertCoreIDFormat(t, output.ID)

	read, err := readPlan(PlansRoot(), output.ID)
	core.RequireNoError(t, err)
	core.AssertLen(t, read.Phases, 1)
	core.AssertEqual(t, "Register actions", read.Phases[0].Name)
	core.AssertEqual(t, []string{"plan.create exists", "tests cover handlers"}, read.Phases[0].Criteria)
	core.AssertEqual(t, 2, read.Phases[0].Tests)
}

func TestPlan_HandlePlanUpdate_Good_JSONPhases(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Update via action",
		Objective: "Parse phase JSON from action options",
	})
	core.RequireNoError(t, err)

	result := s.handlePlanUpdate(context.Background(), core.NewOptions(
		core.Option{Key: "id", Value: created.ID},
		core.Option{Key: "status", Value: "ready"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "phases", Value: `[{"number":1,"name":"Review drift","status":"pending","criteria":["actions registered"]}]`},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(PlanUpdateOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "ready", output.Plan.Status)
	core.AssertEqual(t, "codex", output.Plan.Agent)
	core.AssertLen(t, output.Plan.Phases, 1)
	core.AssertEqual(t, "Review drift", output.Plan.Phases[0].Name)
	core.AssertEqual(t, []string{"actions registered"}, output.Plan.Phases[0].Criteria)
}

func TestPlan_PlanList_Good_FilterByStatus(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Draft", Objective: "D"})
	_, c2, _ := s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Ready", Objective: "R"})
	s.planUpdate(context.Background(), nil, PlanUpdateInput{ID: c2.ID, Status: "ready"})

	_, out, err := s.planList(context.Background(), nil, PlanListInput{Status: "ready"})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, out.Count)
	core.AssertEqual(t, "ready", out.Plans[0].Status)
}

func TestPlan_PlanList_Good_IgnoresNonJSON(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Real", Objective: "Real plan"})

	// Write a non-JSON file in the plans dir
	plansDir := PlansRoot()
	fs.Write(plansDir+"/notes.txt", "not a plan")

	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, out.Count, "should skip non-JSON files")
}

func TestPlan_PlanList_Good_DefaultLimit(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	for i := 0; i < 21; i++ {
		_, _, err := s.planCreate(context.Background(), nil, PlanCreateInput{
			Title:     core.Sprintf("Plan %d", i+1),
			Objective: "Test default list limit",
		})
		core.RequireNoError(t, err)
	}

	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 20, out.Count)
	core.AssertLen(t, out.Plans, 20)
}

// --- planPath edge cases ---

func TestPlan_PlanPath_Bad_PathTraversal(t *testing.T) {
	p := planPath("/tmp/plans", "../../etc/passwd")
	base := core.PathBase(p)
	core.AssertNotContains(t, p, "..")
	core.AssertEqual(t, "passwd.json", base)
}

func TestPlan_PlanPath_Bad_Dot(t *testing.T) {
	core.AssertContains(t, planPath("/tmp", "."), "invalid")
	core.AssertContains(t, planPath("/tmp", ".."), "invalid")
	core.AssertContains(t, planPath("/tmp", ""), "invalid")
}

// --- planCreate Ugly ---

func TestPlan_PlanCreate_Ugly_VeryLongTitle(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	longTitle := repeatString("Long Title With Many Words ", 20)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     longTitle,
		Objective: "Test very long title handling",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertNotEmpty(t, out.ID)
	assertCoreIDFormat(t, out.ID)
}

func TestPlan_PlanCreate_Ugly_UnicodeTitle(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "\u00e9\u00e0\u00fc\u00f1\u00f0 Plan \u2603\u2764\u270c",
		Objective: "Handle unicode gracefully",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertNotEmpty(t, out.ID)
	assertCoreIDFormat(t, out.ID)
	// Should be readable from disk
	core.AssertTrue(t, fs.Exists(out.Path))
}

// --- planRead Ugly ---

func TestPlan_PlanRead_Ugly_SpecialCharsInID(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	// Try to read with special chars — should safely not find it
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{ID: "plan-with-<script>-chars"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not found")
}

func TestPlan_PlanRead_Ugly_UnicodeID(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{ID: "\u00e9\u00e0\u00fc-plan"})
	core.AssertError(t, err)
}

// --- planUpdate Ugly ---

func TestPlan_PlanUpdate_Ugly_EmptyPhasesArray(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Phase Test",
		Objective: "Test empty phases",
		Phases:    []Phase{{Name: "Phase 1", Status: "pending"}},
	})

	// Update with empty phases array — should replace with no phases
	_, updateOut, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:     createOut.ID,
		Phases: []Phase{},
	})
	core.RequireNoError(t, err)
	// Empty slice is still non-nil, so it replaces
	core.AssertEmpty(t, updateOut.Plan.Phases)
}

func TestPlan_PlanUpdate_Ugly_UnicodeNotes(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Unicode Notes",
		Objective: "Test unicode in notes",
	})

	_, updateOut, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:    createOut.ID,
		Notes: "\u00e9\u00e0\u00fc\u00f1 notes with \u2603 snowman and \u00a3 pound sign",
	})
	core.RequireNoError(t, err)
	core.AssertContains(t, updateOut.Plan.Notes, "\u2603")
}

// --- planDelete Ugly ---

func TestPlan_PlanDelete_Ugly_PathTraversalAttempt(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	// Path traversal attempt should be sanitised and not find anything
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{ID: "../../etc/passwd"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not found")
}

func TestPlan_PlanDelete_Ugly_UnicodeID(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{ID: "\u00e9\u00e0\u00fc-to-delete"})
	core.AssertError(t, err)
}

// --- planPath Ugly ---

func TestPlan_PlanPath_Ugly_UnicodeID(t *testing.T) {
	result := planPath("/tmp/plans", "\u00e9\u00e0\u00fc-plan-\u2603")
	core.AssertNotPanics(t, func() {
		_ = planPath("/tmp", "\u00e9\u00e0\u00fc")
	})
	core.AssertContains(t, result, ".json")
}

func TestPlan_PlanPath_Ugly_VeryLongID(t *testing.T) {
	longID := repeatString("a", 500)
	result := planPath("/tmp/plans", longID)
	core.AssertContains(t, result, ".json")
	core.AssertNotEmpty(t, result)
}

// --- validPlanStatus Ugly ---

func TestPlan_ValidPlanStatus_Ugly_UnicodeStatus(t *testing.T) {
	core.AssertFalse(t, validPlanStatus("\u00e9\u00e0\u00fc"))
	core.AssertFalse(t, validPlanStatus("\u2603"))
	core.AssertFalse(t, validPlanStatus("\u0000"))
}

func TestPlan_ValidPlanStatus_Ugly_NearMissStatus(t *testing.T) {
	core.AssertFalse(t, validPlanStatus("Draft"))       // capital D
	core.AssertFalse(t, validPlanStatus("DRAFT"))       // all caps
	core.AssertFalse(t, validPlanStatus("in-progress")) // hyphen instead of underscore
	core.AssertFalse(t, validPlanStatus(" draft"))      // leading space
	core.AssertFalse(t, validPlanStatus("draft "))      // trailing space
}

// --- planList Bad/Ugly ---

func TestPlan_PlanList_Bad_Case(t *testing.T) {
	// Plans dir doesn't exist yet — should create it
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 0, out.Count)
}

func TestPlan_PlanList_Ugly_Case(t *testing.T) {
	// Plans dir has corrupt JSON files
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	// Create a real plan
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Real Plan", Objective: "Test"})

	// Write corrupt JSON file in plans dir
	plansDir := PlansRoot()
	fs.Write(plansDir+"/corrupt-plan.json", "not valid json {{{")

	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, out.Count, "corrupt JSON file should be skipped")
}

// --- writePlan Bad/Ugly ---

func TestPlan_WritePlan_Bad_Case(t *testing.T) {
	// Plan with empty ID
	dir := t.TempDir()
	plan := &Plan{
		ID:        "",
		Title:     "No ID Plan",
		Status:    "draft",
		Objective: "Test empty ID",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Should write with planPath sanitising empty ID to "invalid"
	path, err := writePlan(dir, plan)
	core.RequireNoError(t, err)
	core.AssertContains(t, path, "invalid.json")
}

func TestPlan_WritePlan_Ugly_Case(t *testing.T) {
	// Plan with moderately long ID (within filesystem limits)
	dir := t.TempDir()
	longID := repeatString("a", 100)
	plan := &Plan{
		ID:        longID,
		Title:     "Long ID Plan",
		Status:    "draft",
		Objective: "Test long ID",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	path, err := writePlan(dir, plan)
	core.RequireNoError(t, err)
	core.AssertNotEmpty(t, path)
	core.AssertContains(t, path, ".json")

	// Verify we can read it back
	readBack, err := readPlan(dir, longID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Long ID Plan", readBack.Title)
}
