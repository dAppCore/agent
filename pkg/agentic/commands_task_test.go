// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommands_TaskCommand_Good_Update(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Command",
		Description: "Update task through CLI command",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	r := s.cmdTaskUpdate(core.NewOptions(
		core.Option{Key: "plan_slug", Value: plan.Slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: "1"},
		core.Option{Key: "status", Value: "completed"},
		core.Option{Key: "notes", Value: "Done"},
		core.Option{Key: "file", Value: "pkg/agentic/task.go"},
		core.Option{Key: "line", Value: 128},
	))
	require.True(t, r.OK)

	output, ok := r.Value.(TaskOutput)
	require.True(t, ok)
	assert.Equal(t, "completed", output.Task.Status)
	assert.Equal(t, "Done", output.Task.Notes)
	assert.Equal(t, "pkg/agentic/task.go", output.Task.File)
	assert.Equal(t, 128, output.Task.Line)
}

func TestCommands_TaskCommand_Good_Create(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Command Create",
		Description: "Create task through CLI command",
		Phases: []Phase{
			{Name: "Setup"},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	r := s.cmdTaskCreate(core.NewOptions(
		core.Option{Key: "plan_slug", Value: plan.Slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "title", Value: "Patch code"},
		core.Option{Key: "description", Value: "Update the implementation"},
		core.Option{Key: "status", Value: "pending"},
		core.Option{Key: "notes", Value: "Do this first"},
		core.Option{Key: "file", Value: "pkg/agentic/task.go"},
		core.Option{Key: "line", Value: 153},
	))
	require.True(t, r.OK)

	output, ok := r.Value.(TaskCreateOutput)
	require.True(t, ok)
	assert.Equal(t, "Patch code", output.Task.Title)
	assert.Equal(t, "pending", output.Task.Status)
	assert.Equal(t, "Do this first", output.Task.Notes)
	assert.Equal(t, "pkg/agentic/task.go", output.Task.File)
	assert.Equal(t, 153, output.Task.Line)
}

func TestCommands_TaskCommand_Bad_MissingRequiredFields(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdTaskUpdate(core.NewOptions(
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: "1"},
	))

	assert.False(t, r.OK)
	assert.Contains(t, r.Value.(error).Error(), "required")
}

func TestCommands_TaskCommand_Ugly_ToggleCriteriaFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Toggle",
		Description: "Toggle criteria-derived task",
		Phases: []Phase{
			{Name: "Setup", Criteria: []string{"Review RFC"}},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	r := s.cmdTaskToggle(core.NewOptions(
		core.Option{Key: "plan_slug", Value: plan.Slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: 1},
	))
	require.True(t, r.OK)

	output, ok := r.Value.(TaskOutput)
	require.True(t, ok)
	assert.Equal(t, "completed", output.Task.Status)
	assert.Equal(t, "Review RFC", output.Task.Title)
}

func TestCommands_TaskCommand_Bad_CreateMissingTitle(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdTaskCreate(core.NewOptions(
		core.Option{Key: "plan_slug", Value: "my-plan"},
		core.Option{Key: "phase_order", Value: 1},
	))

	assert.False(t, r.OK)
	assert.Contains(t, r.Value.(error).Error(), "required")
}
