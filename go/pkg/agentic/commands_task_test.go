// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestCommands_TaskCommand_Good_Update(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Command",
		Description: "Update task through CLI command",
		Phases: []Phase{
			{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	r := s.cmdTaskUpdate(core.NewOptions(
		core.Option{Key: "plan_slug", Value: plan.Slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: "1"},
		core.Option{Key: "status", Value: "completed"},
		core.Option{Key: "notes", Value: "Done"},
		core.Option{Key: "priority", Value: "high"},
		core.Option{Key: "category", Value: "security"},
		core.Option{Key: "file", Value: "pkg/agentic/task.go"},
		core.Option{Key: "line", Value: 128},
	))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(TaskOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", output.Task.Status)
	core.AssertEqual(t, "Done", output.Task.Notes)
	core.AssertEqual(t, "high", output.Task.Priority)
	core.AssertEqual(t, "security", output.Task.Category)
	core.AssertEqual(t, "pkg/agentic/task.go", output.Task.File)
	core.AssertEqual(t, 128, output.Task.Line)
}

func TestCommands_TaskCommand_Good_SpecAliasRegistered(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerTaskCommands()

	core.AssertContains(t, c.Commands(), "agentic:task")
	core.AssertContains(t, c.Commands(), "agentic:task/create")
	core.AssertContains(t, c.Commands(), "agentic:task/update")
	core.AssertContains(t, c.Commands(), "agentic:task/toggle")
}

func TestCommands_TaskCommand_Good_Create(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Command Create",
		Description: "Create task through CLI command",
		Phases: []Phase{
			{Name: "Setup"},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	r := s.cmdTaskCreate(core.NewOptions(
		core.Option{Key: "plan_slug", Value: plan.Slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "title", Value: "Patch code"},
		core.Option{Key: "description", Value: "Update the implementation"},
		core.Option{Key: "status", Value: "pending"},
		core.Option{Key: "notes", Value: "Do this first"},
		core.Option{Key: "priority", Value: "high"},
		core.Option{Key: "category", Value: "implementation"},
		core.Option{Key: "file", Value: "pkg/agentic/task.go"},
		core.Option{Key: "line", Value: 153},
	))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(TaskCreateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Patch code", output.Task.Title)
	core.AssertEqual(t, "pending", output.Task.Status)
	core.AssertEqual(t, "Do this first", output.Task.Notes)
	core.AssertEqual(t, "high", output.Task.Priority)
	core.AssertEqual(t, "implementation", output.Task.Category)
	core.AssertEqual(t, "pkg/agentic/task.go", output.Task.File)
	core.AssertEqual(t, 153, output.Task.Line)
}

func TestCommands_TaskCommand_Good_CreateFileRefAliases(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task Command File Ref",
		Description: "Create task through CLI command with RFC aliases",
		Phases: []Phase{
			{Name: "Setup"},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	r := s.cmdTaskCreate(core.NewOptions(
		core.Option{Key: "plan_slug", Value: plan.Slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "title", Value: "Patch code"},
		core.Option{Key: "file_ref", Value: "pkg/agentic/task.go"},
		core.Option{Key: "line_ref", Value: 153},
	))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(TaskCreateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "pkg/agentic/task.go", output.Task.FileRef)
	core.AssertEqual(t, 153, output.Task.LineRef)
	core.AssertEqual(t, "pkg/agentic/task.go", output.Task.File)
	core.AssertEqual(t, 153, output.Task.Line)
}

func TestCommands_TaskCommand_Bad_MissingRequiredFields(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdTaskUpdate(core.NewOptions(
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: "1"},
	))

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "required")
}

func TestCommands_TaskCommand_Ugly_ToggleCriteriaFallback(t *testing.T) {
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

	r := s.cmdTaskToggle(core.NewOptions(
		core.Option{Key: "plan_slug", Value: plan.Slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: 1},
	))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(TaskOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", output.Task.Status)
	core.AssertEqual(t, "Review RFC", output.Task.Title)
}

func TestCommands_TaskCommand_Bad_CreateMissingTitle(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdTaskCreate(core.NewOptions(
		core.Option{Key: "plan_slug", Value: "my-plan"},
		core.Option{Key: "phase_order", Value: 1},
	))

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "required")
}
