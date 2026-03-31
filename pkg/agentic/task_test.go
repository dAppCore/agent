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
	})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "completed", output.Task.Status)
	assert.Equal(t, "Done", output.Task.Notes)
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
