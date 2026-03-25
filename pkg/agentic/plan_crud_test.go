// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPrep creates a PrepSubsystem for testing.
func newTestPrep(t *testing.T) *PrepSubsystem {
	t.Helper()
	return &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
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

	_, statErr := os.Stat(out.Path)
	assert.NoError(t, statErr)
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

	_, statErr := os.Stat(createOut.Path)
	assert.True(t, os.IsNotExist(statErr))
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
	os.WriteFile(plansDir+"/notes.txt", []byte("not a plan"), 0o644)

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
