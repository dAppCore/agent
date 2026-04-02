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

func TestCommandsPlan_CmdPlanUpdate_Good_StatusAndAgent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Update Command",
		Objective: "Verify the plan update command",
	})
	require.NoError(t, err)

	r := s.cmdPlanUpdate(core.NewOptions(
		core.Option{Key: "_arg", Value: created.ID},
		core.Option{Key: "status", Value: "ready"},
		core.Option{Key: "agent", Value: "codex"},
	))
	require.True(t, r.OK)

	output, ok := r.Value.(PlanUpdateOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, created.ID, output.Plan.ID)
	assert.Equal(t, "ready", output.Plan.Status)
	assert.Equal(t, "codex", output.Plan.Agent)
}

func TestCommandsPlan_CmdPlanUpdate_Bad_MissingFields(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdPlanUpdate(core.NewOptions(
		core.Option{Key: "_arg", Value: "plan-123"},
	))

	assert.False(t, r.OK)
	require.Error(t, r.Value.(error))
	assert.Contains(t, r.Value.(error).Error(), "at least one update field is required")
}

func TestCommandsPlan_HandlePlanCheck_Good_CompletePlan(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Action Check Plan",
		Description: "Confirm the plan check action reports completion",
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

	r := s.handlePlanCheck(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: plan.Slug},
	))
	require.True(t, r.OK)

	output, ok := r.Value.(PlanCheckOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.True(t, output.Complete)
	assert.Equal(t, plan.Slug, output.Plan.Slug)
}

func TestCommandsPlan_RegisterPlanCommands_Good_SpecAliasRegistered(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerPlanCommands()

	assert.Contains(t, c.Commands(), "agentic:plan")
	assert.Contains(t, c.Commands(), "plan")
	assert.Contains(t, c.Commands(), "agentic:plan/create")
	assert.Contains(t, c.Commands(), "agentic:plan/get")
	assert.Contains(t, c.Commands(), "plan/get")
	assert.Contains(t, c.Commands(), "agentic:plan/list")
	assert.Contains(t, c.Commands(), "agentic:plan/read")
	assert.Contains(t, c.Commands(), "plan/read")
	assert.Contains(t, c.Commands(), "agentic:plan/show")
	assert.Contains(t, c.Commands(), "plan/show")
	assert.Contains(t, c.Commands(), "agentic:plan/status")
	assert.Contains(t, c.Commands(), "plan/update")
	assert.Contains(t, c.Commands(), "agentic:plan/update")
	assert.Contains(t, c.Commands(), "plan/status")
	assert.Contains(t, c.Commands(), "agentic:plan/check")
	assert.Contains(t, c.Commands(), "plan/check")
	assert.Contains(t, c.Commands(), "agentic:plan/archive")
	assert.Contains(t, c.Commands(), "plan/archive")
	assert.Contains(t, c.Commands(), "agentic:plan/delete")
	assert.Contains(t, c.Commands(), "plan/delete")
}
