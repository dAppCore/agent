// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"strings"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPlan_PlanCreate_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

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
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.NotEmpty(t, out.ID)
	assert.Contains(t, out.ID, "migrate-core")
	assert.NotEmpty(t, out.Path)

	assert.True(t, fs.Exists(out.Path))
}

func TestPlan_PlanCreate_Bad_MissingTitle(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Objective: "something",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestPlan_PlanCreate_Bad_MissingObjective(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "My Plan",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "objective is required")
}

func TestPlan_PlanCreate_Good_DefaultPhaseStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Test Plan",
		Objective: "Test defaults",
		Phases:    []Phase{{Name: "Phase 1"}, {Name: "Phase 2"}},
	})
	require.NoError(t, err)

	plan, readErr := readPlan(PlansRoot(), out.ID)
	require.NoError(t, readErr)
	assert.Equal(t, "pending", plan.Phases[0].Status)
	assert.Equal(t, "pending", plan.Phases[1].Status)
	assert.Equal(t, 1, plan.Phases[0].Number)
	assert.Equal(t, 2, plan.Phases[1].Number)
}

// --- planRead (MCP handler) ---

func TestPlan_PlanRead_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, createOut, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Read Test",
		Objective: "Verify read works",
	})
	require.NoError(t, err)

	_, readOut, err := s.planRead(context.Background(), nil, PlanReadInput{ID: createOut.ID})
	require.NoError(t, err)
	assert.True(t, readOut.Success)
	assert.Equal(t, createOut.ID, readOut.Plan.ID)
	assert.Equal(t, "Read Test", readOut.Plan.Title)
	assert.Equal(t, "draft", readOut.Plan.Status)
}

func TestPlan_PlanRead_Bad_MissingID(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestPlan_PlanRead_Bad_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{ID: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- planUpdate (MCP handler) ---

func TestPlan_PlanUpdate_Good_Status(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Update Test",
		Objective: "Verify update",
	})

	_, updateOut, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:     createOut.ID,
		Status: "ready",
	})
	require.NoError(t, err)
	assert.True(t, updateOut.Success)
	assert.Equal(t, "ready", updateOut.Plan.Status)
}

func TestPlan_PlanUpdate_Good_PartialUpdate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

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
	require.NoError(t, err)
	assert.Equal(t, "New Title", updateOut.Plan.Title)
	assert.Equal(t, "Original objective", updateOut.Plan.Objective)
	assert.Equal(t, "Original notes", updateOut.Plan.Notes)
	assert.Equal(t, "codex", updateOut.Plan.Agent)
}

func TestPlan_PlanUpdate_Good_AllStatusTransitions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "Status Lifecycle", Objective: "Test transitions",
	})

	transitions := []string{"ready", "in_progress", "needs_verification", "verified", "approved"}
	for _, status := range transitions {
		_, out, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
			ID: createOut.ID, Status: status,
		})
		require.NoError(t, err, "transition to %s", status)
		assert.Equal(t, status, out.Plan.Status)
	}
}

func TestPlan_PlanUpdate_Bad_InvalidStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "Bad Status", Objective: "Test",
	})

	_, _, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID: createOut.ID, Status: "invalid_status",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status")
}

func TestPlan_PlanUpdate_Bad_MissingID(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{Status: "ready"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestPlan_PlanUpdate_Good_ReplacePhases(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

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
	require.NoError(t, err)
	assert.Len(t, updateOut.Plan.Phases, 2)
	assert.Equal(t, "New Phase", updateOut.Plan.Phases[0].Name)
}

// --- planDelete (MCP handler) ---

func TestPlan_PlanDelete_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title: "Delete Me", Objective: "Will be deleted",
	})

	_, delOut, err := s.planDelete(context.Background(), nil, PlanDeleteInput{ID: createOut.ID})
	require.NoError(t, err)
	assert.True(t, delOut.Success)
	assert.Equal(t, createOut.ID, delOut.Deleted)

	assert.False(t, fs.Exists(createOut.Path))
}

func TestPlan_PlanDelete_Bad_MissingID(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id is required")
}

func TestPlan_PlanDelete_Bad_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{ID: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- planList (MCP handler) ---

func TestPlan_PlanList_Good_Empty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 0, out.Count)
}

func TestPlan_PlanList_Good_Multiple(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "A", Objective: "A", Repo: "go-io"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "B", Objective: "B", Repo: "go-crypt"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "C", Objective: "C", Repo: "go-io"})

	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	require.NoError(t, err)
	assert.Equal(t, 3, out.Count)
}

func TestPlan_PlanList_Good_FilterByRepo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "A", Objective: "A", Repo: "go-io"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "B", Objective: "B", Repo: "go-crypt"})
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "C", Objective: "C", Repo: "go-io"})

	_, out, err := s.planList(context.Background(), nil, PlanListInput{Repo: "go-io"})
	require.NoError(t, err)
	assert.Equal(t, 2, out.Count)
}

func TestPlan_PlanList_Good_FilterByStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Draft", Objective: "D"})
	_, c2, _ := s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Ready", Objective: "R"})
	s.planUpdate(context.Background(), nil, PlanUpdateInput{ID: c2.ID, Status: "ready"})

	_, out, err := s.planList(context.Background(), nil, PlanListInput{Status: "ready"})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Count)
	assert.Equal(t, "ready", out.Plans[0].Status)
}

func TestPlan_PlanList_Good_IgnoresNonJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Real", Objective: "Real plan"})

	// Write a non-JSON file in the plans dir
	plansDir := PlansRoot()
	fs.Write(plansDir+"/notes.txt", "not a plan")

	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Count, "should skip non-JSON files")
}

// --- planPath edge cases ---

func TestPlan_PlanPath_Bad_PathTraversal(t *testing.T) {
	p := planPath("/tmp/plans", "../../etc/passwd")
	assert.NotContains(t, p, "..")
}

func TestPlan_PlanPath_Bad_Dot(t *testing.T) {
	assert.Contains(t, planPath("/tmp", "."), "invalid")
	assert.Contains(t, planPath("/tmp", ".."), "invalid")
	assert.Contains(t, planPath("/tmp", ""), "invalid")
}

// --- planCreate Ugly ---

func TestPlan_PlanCreate_Ugly_VeryLongTitle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	longTitle := strings.Repeat("Long Title With Many Words ", 20)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     longTitle,
		Objective: "Test very long title handling",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.NotEmpty(t, out.ID)
	// The slug portion should be truncated
	assert.LessOrEqual(t, len(out.ID), 50, "ID should be reasonably short")
}

func TestPlan_PlanCreate_Ugly_UnicodeTitle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "\u00e9\u00e0\u00fc\u00f1\u00f0 Plan \u2603\u2764\u270c",
		Objective: "Handle unicode gracefully",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.NotEmpty(t, out.ID)
	// Should be readable from disk
	assert.True(t, fs.Exists(out.Path))
}

// --- planRead Ugly ---

func TestPlan_PlanRead_Ugly_SpecialCharsInID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	// Try to read with special chars — should safely not find it
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{ID: "plan-with-<script>-chars"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPlan_PlanRead_Ugly_UnicodeID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, _, err := s.planRead(context.Background(), nil, PlanReadInput{ID: "\u00e9\u00e0\u00fc-plan"})
	assert.Error(t, err, "unicode ID should not find a file")
}

// --- planUpdate Ugly ---

func TestPlan_PlanUpdate_Ugly_EmptyPhasesArray(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

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
	require.NoError(t, err)
	// Empty slice is still non-nil, so it replaces
	assert.Empty(t, updateOut.Plan.Phases)
}

func TestPlan_PlanUpdate_Ugly_UnicodeNotes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, createOut, _ := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Unicode Notes",
		Objective: "Test unicode in notes",
	})

	_, updateOut, err := s.planUpdate(context.Background(), nil, PlanUpdateInput{
		ID:    createOut.ID,
		Notes: "\u00e9\u00e0\u00fc\u00f1 notes with \u2603 snowman and \u00a3 pound sign",
	})
	require.NoError(t, err)
	assert.Contains(t, updateOut.Plan.Notes, "\u2603")
}

// --- planDelete Ugly ---

func TestPlan_PlanDelete_Ugly_PathTraversalAttempt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	// Path traversal attempt should be sanitised and not find anything
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{ID: "../../etc/passwd"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPlan_PlanDelete_Ugly_UnicodeID(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, _, err := s.planDelete(context.Background(), nil, PlanDeleteInput{ID: "\u00e9\u00e0\u00fc-to-delete"})
	assert.Error(t, err, "unicode ID should not match a real plan")
}

// --- planPath Ugly ---

func TestPlan_PlanPath_Ugly_UnicodeID(t *testing.T) {
	result := planPath("/tmp/plans", "\u00e9\u00e0\u00fc-plan-\u2603")
	assert.NotPanics(t, func() {
		_ = planPath("/tmp", "\u00e9\u00e0\u00fc")
	})
	assert.Contains(t, result, ".json")
}

func TestPlan_PlanPath_Ugly_VeryLongID(t *testing.T) {
	longID := strings.Repeat("a", 500)
	result := planPath("/tmp/plans", longID)
	assert.Contains(t, result, ".json")
	assert.NotEmpty(t, result)
}

// --- validPlanStatus Ugly ---

func TestPlan_ValidPlanStatus_Ugly_UnicodeStatus(t *testing.T) {
	assert.False(t, validPlanStatus("\u00e9\u00e0\u00fc"))
	assert.False(t, validPlanStatus("\u2603"))
	assert.False(t, validPlanStatus("\u0000"))
}

func TestPlan_ValidPlanStatus_Ugly_NearMissStatus(t *testing.T) {
	assert.False(t, validPlanStatus("Draft"))     // capital D
	assert.False(t, validPlanStatus("DRAFT"))      // all caps
	assert.False(t, validPlanStatus("in-progress")) // hyphen instead of underscore
	assert.False(t, validPlanStatus(" draft"))      // leading space
	assert.False(t, validPlanStatus("draft "))      // trailing space
}

// --- generatePlanID Bad/Ugly ---

func TestPlan_GeneratePlanID_Bad(t *testing.T) {
	// Empty title — slug will be empty, but random suffix is still appended
	id := generatePlanID("")
	assert.NotEmpty(t, id, "should still generate an ID with random suffix")
	assert.Contains(t, id, "-", "should have random suffix separated by dash")
}

func TestPlan_GeneratePlanID_Ugly(t *testing.T) {
	// Title with only special chars — slug will be empty
	id := generatePlanID("!@#$%^&*()")
	assert.NotEmpty(t, id, "should still generate an ID with random suffix")
}

// --- planList Bad/Ugly ---

func TestPlan_PlanList_Bad(t *testing.T) {
	// Plans dir doesn't exist yet — should create it
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 0, out.Count)
}

func TestPlan_PlanList_Ugly(t *testing.T) {
	// Plans dir has corrupt JSON files
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	// Create a real plan
	s.planCreate(context.Background(), nil, PlanCreateInput{Title: "Real Plan", Objective: "Test"})

	// Write corrupt JSON file in plans dir
	plansDir := PlansRoot()
	fs.Write(plansDir+"/corrupt-plan.json", "not valid json {{{")

	_, out, err := s.planList(context.Background(), nil, PlanListInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Count, "corrupt JSON file should be skipped")
}

// --- writePlan Bad/Ugly ---

func TestPlan_WritePlan_Bad(t *testing.T) {
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
	require.NoError(t, err)
	assert.Contains(t, path, "invalid.json")
}

func TestPlan_WritePlan_Ugly(t *testing.T) {
	// Plan with moderately long ID (within filesystem limits)
	dir := t.TempDir()
	longID := strings.Repeat("a", 100)
	plan := &Plan{
		ID:        longID,
		Title:     "Long ID Plan",
		Status:    "draft",
		Objective: "Test long ID",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	path, err := writePlan(dir, plan)
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".json")

	// Verify we can read it back
	readBack, err := readPlan(dir, longID)
	require.NoError(t, err)
	assert.Equal(t, "Long ID Plan", readBack.Title)
}
