// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlancompat_PlanCreateCompat_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, output, err := s.planCreateCompat(context.Background(), nil, PlanCreateInput{
		Title:       "Compatibility Plan",
		Description: "Match the slug-first RFC surface",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{Title: "Review RFC"}}},
		},
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Contains(t, output.Plan.Slug, "compatibility-plan")
	assert.Equal(t, "draft", output.Plan.Status)
	assert.Equal(t, 1, output.Plan.Phases)
}

func TestPlancompat_PlanGetCompat_Good_BySlug(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Get Compat",
		Description: "Read by slug",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{Title: "Review RFC", Status: "completed"}, {Title: "Patch code"}}},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.planGetCompat(context.Background(), nil, PlanReadInput{Slug: plan.Slug})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, plan.Slug, output.Plan.Slug)
	assert.Equal(t, 2, output.Plan.Progress.Total)
	assert.Equal(t, 1, output.Plan.Progress.Completed)
	assert.Equal(t, 50, output.Plan.Progress.Percentage)
}

func TestPlancompat_PlanUpdateStatusCompat_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Status Compat",
		Description: "Update by slug",
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.planUpdateStatusCompat(context.Background(), nil, PlanStatusUpdateInput{
		Slug:   plan.Slug,
		Status: "active",
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "active", output.Plan.Status)
}

func TestPlancompat_PlanArchiveCompat_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Archive Compat",
		Description: "Archive by slug",
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.planArchiveCompat(context.Background(), nil, PlanDeleteInput{
		Slug:   plan.Slug,
		Reason: "No longer needed",
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, plan.Slug, output.Archived)

	archivedPlan, err := readPlan(PlansRoot(), plan.Slug)
	require.NoError(t, err)
	assert.Equal(t, "archived", archivedPlan.Status)
	assert.False(t, archivedPlan.ArchivedAt.IsZero())
	assert.Contains(t, archivedPlan.Notes, "No longer needed")
}
