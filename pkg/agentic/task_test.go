// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_TaskUpdate_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Update",
		Description: "Update task by identifier",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.taskUpdate(context.Background(), nil, TaskUpdateInput{
		PlanSlug:       plan.Slug,
		PhaseOrder:     1,
		TaskIdentifier: "1",
		Status:         "completed",
		Notes:          "Done",
		File:           "pkg/agentic/task.go",
		Line:           128,
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "completed", output.Task.Status)
	assert.Equal(t, "Done", output.Task.Notes)
	assert.Equal(t, "pkg/agentic/task.go", output.Task.File)
	assert.Equal(t, 128, output.Task.Line)
}

func TestTask_TaskCreate_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Create",
		Description: "Create task by phase",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:    plan.Slug,
		PhaseOrder:  1,
		Title:       "Patch code",
		Description: "Update the implementation",
		Status:      "pending",
		Notes:       "Do this first",
		File:        "pkg/agentic/task.go",
		Line:        153,
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "Patch code", output.Task.Title)
	assert.Equal(t, "pending", output.Task.Status)
	assert.Equal(t, "Do this first", output.Task.Notes)
	assert.Equal(t, "pkg/agentic/task.go", output.Task.File)
	assert.Equal(t, 153, output.Task.Line)
}

func TestTask_TaskCreate_Bad_MissingTitle(t *testing.T) {
	s := newTestPrep(t)

	_, _, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:   "my-plan",
		PhaseOrder: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "title is required")
}

func TestTask_TaskToggle_Bad_MissingIdentifier(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.taskToggle(context.Background(), nil, TaskToggleInput{
		PlanSlug:   "my-plan",
		PhaseOrder: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task_identifier is required")
}

func TestTask_TaskToggle_Ugly_CriteriaFallback(t *testing.T) {
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

	_, output, err := s.taskToggle(context.Background(), nil, TaskToggleInput{
		PlanSlug:       plan.Slug,
		PhaseOrder:     1,
		TaskIdentifier: 1,
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "completed", output.Task.Status)
	assert.Equal(t, "Review RFC", output.Task.Title)
}

func TestTask_TaskCreate_Ugly_CriteriaFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Create Criteria",
		Description: "Create task from criteria fallback",
		Phases: []Phase{
			{Name: "Setup", Criteria: []string{"Review RFC"}},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, output, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
		Title:      "Patch code",
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "Patch code", output.Task.Title)

	updated, err := readPlan(PlansRoot(), plan.ID)
	require.NoError(t, err)
	require.Len(t, updated.Phases[0].Tasks, 2)
	assert.Equal(t, "Review RFC", updated.Phases[0].Tasks[0].Title)
	assert.Equal(t, "Patch code", updated.Phases[0].Tasks[1].Title)
	assert.Empty(t, updated.Phases[0].Tasks[1].File)
	assert.Zero(t, updated.Phases[0].Tasks[1].Line)
}

func TestTask_TaskFileRefAliases_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task File Ref Aliases",
		Description: "Accept RFC task file reference names",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	require.NoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)

	_, createdOutput, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
		Title:      "Patch code",
		FileRef:    "pkg/agentic/task.go",
		LineRef:    153,
	})
	require.NoError(t, err)
	assert.Equal(t, "pkg/agentic/task.go", createdOutput.Task.FileRef)
	assert.Equal(t, 153, createdOutput.Task.LineRef)
	assert.Equal(t, "pkg/agentic/task.go", createdOutput.Task.File)
	assert.Equal(t, 153, createdOutput.Task.Line)

	_, updatedOutput, err := s.taskUpdate(context.Background(), nil, TaskUpdateInput{
		PlanSlug:       plan.Slug,
		PhaseOrder:     1,
		TaskIdentifier: createdOutput.Task.ID,
		FileRef:        "pkg/agentic/task.go",
		LineRef:        171,
	})
	require.NoError(t, err)
	assert.Equal(t, "pkg/agentic/task.go", updatedOutput.Task.FileRef)
	assert.Equal(t, 171, updatedOutput.Task.LineRef)
	assert.Equal(t, "pkg/agentic/task.go", updatedOutput.Task.File)
	assert.Equal(t, 171, updatedOutput.Task.Line)
}
