// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestTask_TaskUpdate_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Update",
		Description: "Update task by identifier",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.taskUpdate(context.Background(), nil, TaskUpdateInput{
		PlanSlug:       plan.Slug,
		PhaseOrder:     1,
		TaskIdentifier: "1",
		Status:         "completed",
		Notes:          "Done",
		Priority:       "high",
		Category:       "security",
		File:           "pkg/agentic/task.go",
		Line:           128,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "completed", output.Task.Status)
	core.AssertEqual(t, "Done", output.Task.Notes)
	core.AssertEqual(t, "high", output.Task.Priority)
	core.AssertEqual(t, "security", output.Task.Category)
	core.AssertEqual(t, "pkg/agentic/task.go", output.Task.File)
	core.AssertEqual(t, 128, output.Task.Line)
}

func TestTask_TaskCreate_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Create",
		Description: "Create task by phase",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:    plan.Slug,
		PhaseOrder:  1,
		Title:       "Patch code",
		Description: "Update the implementation",
		Status:      "pending",
		Notes:       "Do this first",
		Priority:    "high",
		Category:    "implementation",
		File:        "pkg/agentic/task.go",
		Line:        153,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "Patch code", output.Task.Title)
	core.AssertEqual(t, "pending", output.Task.Status)
	core.AssertEqual(t, "Do this first", output.Task.Notes)
	core.AssertEqual(t, "high", output.Task.Priority)
	core.AssertEqual(t, "implementation", output.Task.Category)
	core.AssertEqual(t, "pkg/agentic/task.go", output.Task.File)
	core.AssertEqual(t, 153, output.Task.Line)
}

func TestTask_TaskCreate_Bad_MissingTitle(t *testing.T) {
	s := newTestPrep(t)

	_, _, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:   "my-plan",
		PhaseOrder: 1,
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "title is required")
}

func TestTask_TaskToggle_Bad_MissingIdentifier(t *testing.T) {
	s := newTestPrep(t)
	_, _, err := s.taskToggle(context.Background(), nil, TaskToggleInput{
		PlanSlug:   "my-plan",
		PhaseOrder: 1,
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "task_identifier is required")
}

func TestTask_TaskToggle_Ugly_CriteriaFallback(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Toggle",
		Description: "Toggle criteria-derived task",
		Phases: []Phase{
			{Name: "Setup", Criteria: []string{"Review RFC"}},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.taskToggle(context.Background(), nil, TaskToggleInput{
		PlanSlug:       plan.Slug,
		PhaseOrder:     1,
		TaskIdentifier: 1,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "completed", output.Task.Status)
	core.AssertEqual(t, "Review RFC", output.Task.Title)
}

func TestTask_TaskCreate_Ugly_CriteriaFallback(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Create Criteria",
		Description: "Create task from criteria fallback",
		Phases: []Phase{
			{Name: "Setup", Criteria: []string{"Review RFC"}},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, output, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
		Title:      "Patch code",
		Priority:   "medium",
		Category:   "research",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "Patch code", output.Task.Title)
	core.AssertEqual(t, "medium", output.Task.Priority)
	core.AssertEqual(t, "research", output.Task.Category)

	updated, err := readPlan(PlansRoot(), plan.ID)
	core.RequireNoError(t, err)
	core.AssertLen(t, updated.Phases[0].Tasks, 2)
	core.AssertEqual(t, "Review RFC", updated.Phases[0].Tasks[0].Title)
	core.AssertEqual(t, "Patch code", updated.Phases[0].Tasks[1].Title)
	core.AssertEmpty(t, updated.Phases[0].Tasks[1].File)
	assertZero(t, updated.Phases[0].Tasks[1].Line)
}

func TestTask_TaskFileRefAliases_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task File Ref Aliases",
		Description: "Accept RFC task file reference names",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	_, createdOutput, err := s.taskCreate(context.Background(), nil, TaskCreateInput{
		PlanSlug:   plan.Slug,
		PhaseOrder: 1,
		Title:      "Patch code",
		FileRef:    "pkg/agentic/task.go",
		LineRef:    153,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "pkg/agentic/task.go", createdOutput.Task.FileRef)
	core.AssertEqual(t, 153, createdOutput.Task.LineRef)
	core.AssertEqual(t, "pkg/agentic/task.go", createdOutput.Task.File)
	core.AssertEqual(t, 153, createdOutput.Task.Line)

	_, updatedOutput, err := s.taskUpdate(context.Background(), nil, TaskUpdateInput{
		PlanSlug:       plan.Slug,
		PhaseOrder:     1,
		TaskIdentifier: createdOutput.Task.ID,
		FileRef:        "pkg/agentic/task.go",
		LineRef:        171,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "pkg/agentic/task.go", updatedOutput.Task.FileRef)
	core.AssertEqual(t, 171, updatedOutput.Task.LineRef)
	core.AssertEqual(t, "pkg/agentic/task.go", updatedOutput.Task.File)
	core.AssertEqual(t, 171, updatedOutput.Task.Line)
}
