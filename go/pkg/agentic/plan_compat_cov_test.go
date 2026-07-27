// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestPlanCompatCov_HandlePlanCheck_Bad_UnknownSlug — the plan.check action
// wrapper surfaces the read error when the slug resolves to no plan (the error
// arm; the success arm is covered elsewhere).
func TestPlanCompatCov_HandlePlanCheck_Bad_UnknownSlug(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	result := s.handlePlanCheck(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "does-not-exist"},
	))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "plan not found")
}

// TestPlanCompatCov_HandlePlanUpdateStatus_Good_ActivatesPlan — the
// plan.update_status action wrapper maps the public "active" status to the
// internal status and returns the updated compatibility view (the success arm).
func TestPlanCompatCov_HandlePlanUpdateStatus_Good_ActivatesPlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Status Action",
		Objective: "Drive the named status action",
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	result := s.handlePlanUpdateStatus(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: plan.Slug},
		core.Option{Key: "status", Value: "active"},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(PlanCompatibilityGetOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "active", output.Plan.Status)

	// The internal status persisted is in_progress (active maps to in_progress).
	reread, err := readPlan(PlansRoot(), plan.ID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "in_progress", reread.Status)
}

// TestPlanCompatCov_InputStatus_Good_MapsPublicToInternal — the public status
// vocabulary maps onto the internal lifecycle; unknown values pass through.
func TestPlanCompatCov_InputStatus_Good_MapsPublicToInternal(t *testing.T) {
	core.AssertEqual(t, "in_progress", planCompatibilityInputStatus("active"))
	core.AssertEqual(t, "approved", planCompatibilityInputStatus("completed"))
	core.AssertEqual(t, "draft", planCompatibilityInputStatus("draft"))
}

// TestPlanCompatCov_OutputStatus_Good_MapsInternalToPublic — the internal
// lifecycle collapses onto the public vocabulary; unknown values pass through.
func TestPlanCompatCov_OutputStatus_Good_MapsInternalToPublic(t *testing.T) {
	core.AssertEqual(t, "active", planCompatibilityOutputStatus("in_progress"))
	core.AssertEqual(t, "active", planCompatibilityOutputStatus("needs_verification"))
	core.AssertEqual(t, "active", planCompatibilityOutputStatus("verified"))
	core.AssertEqual(t, "completed", planCompatibilityOutputStatus("approved"))
	core.AssertEqual(t, "draft", planCompatibilityOutputStatus("draft"))
}

// TestPlanCompatCov_PlanProgress_Good_PhaseStatusWithoutTasks — phases that
// carry no tasks/criteria each count as one unit, and a "done"/"approved"
// phase status counts as completed (the phase-status fallback branch).
func TestPlanCompatCov_PlanProgress_Good_PhaseStatusWithoutTasks(t *testing.T) {
	plan := Plan{
		Phases: []Phase{
			{Name: "Design", Status: "completed"},
			{Name: "Build", Status: "done"},
			{Name: "Review", Status: "approved"},
			{Name: "Ship", Status: "pending"},
		},
	}

	progress := planProgress(plan)

	core.AssertEqual(t, 4, progress.Total)
	core.AssertEqual(t, 3, progress.Completed)
	core.AssertEqual(t, 75, progress.Percentage)
}

// TestPlanCompatCov_PlanProgress_Ugly_NoPhasesIsZero — a plan with no phases
// reports zero total and zero percentage (the total==0 guard).
func TestPlanCompatCov_PlanProgress_Ugly_NoPhasesIsZero(t *testing.T) {
	progress := planProgress(Plan{})

	core.AssertEqual(t, 0, progress.Total)
	core.AssertEqual(t, 0, progress.Percentage)
}

// TestPlanCompatCov_PlanProgress_Good_TasksTakePrecedence — a phase with tasks
// is scored by its task completion, not its phase status (the task branch).
func TestPlanCompatCov_PlanProgress_Good_TasksTakePrecedence(t *testing.T) {
	plan := Plan{
		Phases: []Phase{
			{
				Name:   "Implement",
				Status: "pending",
				Tasks: []PlanTask{
					{Title: "one", Status: "completed"},
					{Title: "two", Status: "pending"},
				},
			},
		},
	}

	progress := planProgress(plan)

	core.AssertEqual(t, 2, progress.Total)
	core.AssertEqual(t, 1, progress.Completed)
	core.AssertEqual(t, 50, progress.Percentage)
}
