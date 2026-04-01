// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsPlan_CmdPlanCheck_Good_CompletePlan(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Check Plan",
		Description: "Confirm the plan check command reports completion",
		Phases: []Phase{
			{
				Name: "Setup",
				Tasks: []PlanTask{
					{ID: "1", Title: "Review RFC", Status: "completed"},
				},
			},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	r := s.cmdPlanCheck(core.NewOptions(core.Option{Key: "_arg", Value: plan.Slug}))
	require.True(t, r.OK)

	output, ok := r.Value.(PlanCheckOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.True(t, output.Complete)
	assert.Empty(t, output.Pending)
	assert.Equal(t, plan.Slug, output.Plan.Slug)
}

func TestCommandsPlan_CmdPlanCheck_Bad_MissingSlug(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdPlanCheck(core.NewOptions())

	assert.False(t, r.OK)
	require.Error(t, r.Value.(error))
	assert.Contains(t, r.Value.(error).Error(), "slug is required")
}

func TestCommandsPlan_CmdPlanCheck_Ugly_IncompletePhase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Incomplete Plan",
		Description: "Leave one task pending",
		Phases: []Phase{
			{
				Number: 1,
				Name:   "Setup",
				Tasks: []PlanTask{
					{ID: "1", Title: "Review RFC", Status: "completed"},
					{ID: "2", Title: "Patch code", Status: "pending"},
				},
			},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	r := s.cmdPlanCheck(core.NewOptions(
		core.Option{Key: "slug", Value: plan.Slug},
		core.Option{Key: "phase", Value: 1},
	))

	assert.False(t, r.OK)
	output, ok := r.Value.(PlanCheckOutput)
	require.True(t, ok)
	assert.False(t, output.Complete)
	assert.Equal(t, 1, output.Phase)
	assert.Equal(t, "Setup", output.PhaseName)
	assert.Equal(t, []string{"Patch code"}, output.Pending)
}
