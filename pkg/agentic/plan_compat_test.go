// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestPlancompat_PlanCreateCompat_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, output, err := s.planCreateCompat(context.Background(), nil, PlanCreateInput{
		Title:       "Compatibility Plan",
		Description: "Match the slug-first RFC surface",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{Title: "Review RFC"}}},
		},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertContains(t, output.Plan.Slug, "compatibility-plan")
	core.AssertEqual(t, "draft", output.Plan.Status)
	core.AssertEqual(t, 1, output.Plan.Phases)
}

func TestPlancompat_PlanGetCompat_Good_BySlug(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Get Compat",
		Description: "Read by slug",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{Title: "Review RFC", Status: "completed"}, {Title: "Patch code"}}},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.planGetCompat(context.Background(), nil, PlanReadInput{Slug: plan.Slug})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, plan.Slug, output.Plan.Slug)
	core.AssertEqual(t, 2, output.Plan.Progress.Total)
	core.AssertEqual(t, 1, output.Plan.Progress.Completed)
	core.AssertEqual(t, 50, output.Plan.Progress.Percentage)
}

func TestPlancompat_PlanUpdateStatusCompat_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Status Compat",
		Description: "Update by slug",
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.planUpdateStatusCompat(context.Background(), nil, PlanStatusUpdateInput{
		Slug:   plan.Slug,
		Status: "active",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "active", output.Plan.Status)
}

func TestPlancompat_PlanArchiveCompat_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Archive Compat",
		Description: "Archive by slug",
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.planArchiveCompat(context.Background(), nil, PlanDeleteInput{
		Slug:   plan.Slug,
		Reason: "No longer needed",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, plan.Slug, output.Archived)

	archivedPlan, err := readPlan(PlansRoot(), plan.Slug)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "archived", archivedPlan.Status)
	core.AssertFalse(t, archivedPlan.ArchivedAt.IsZero())
	core.AssertContains(t, archivedPlan.Notes, "No longer needed")
}
