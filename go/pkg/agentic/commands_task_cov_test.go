// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// covTaskPlan seeds a plan with a single phase + task and returns the prep and
// the plan slug for the task commands to operate on.
func covTaskPlan(t *testing.T) (*PrepSubsystem, string) {
	t.Helper()
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Task routing plan",
		Description: "Exercise the task command router",
		Phases:      []Phase{{Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review RFC"}}}},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)
	return s, plan.Slug
}

// TestCommandsTaskCov_CmdTask_Good_RoutesUpdate — the "update" action routes to
// cmdTaskUpdate and applies the change.
func TestCommandsTaskCov_CmdTask_Good_RoutesUpdate(t *testing.T) {
	s, slug := covTaskPlan(t)

	r := s.cmdTask(core.NewOptions(
		core.Option{Key: "action", Value: "update"},
		core.Option{Key: "plan_slug", Value: slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: "1"},
		core.Option{Key: "status", Value: "completed"},
	))
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(TaskOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", out.Task.Status)
}

// TestCommandsTaskCov_CmdTask_Good_RoutesToggle — the "toggle" action routes to
// cmdTaskToggle and flips the task status.
func TestCommandsTaskCov_CmdTask_Good_RoutesToggle(t *testing.T) {
	s, slug := covTaskPlan(t)

	r := s.cmdTask(core.NewOptions(
		core.Option{Key: "action", Value: "toggle"},
		core.Option{Key: "plan_slug", Value: slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "task_identifier", Value: "1"},
	))
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(TaskOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", out.Task.Status)
}

// TestCommandsTaskCov_CmdTask_Good_RoutesCreate — the "create" action routes to
// cmdTaskCreate and adds the task.
func TestCommandsTaskCov_CmdTask_Good_RoutesCreate(t *testing.T) {
	s, slug := covTaskPlan(t)

	r := s.cmdTask(core.NewOptions(
		core.Option{Key: "action", Value: "create"},
		core.Option{Key: "plan_slug", Value: slug},
		core.Option{Key: "phase_order", Value: 1},
		core.Option{Key: "title", Value: "New task"},
	))
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(TaskCreateOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "New task", out.Task.Title)
}

// TestCommandsTaskCov_CmdTask_Bad_MissingActionShowsUsage — no action prints the
// usage block and returns OK.
func TestCommandsTaskCov_CmdTask_Bad_MissingActionShowsUsage(t *testing.T) {
	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdTask(core.NewOptions())
		core.AssertTrue(t, r.OK)
	})
	core.AssertContains(t, output, "core-agent task update")
}

// TestCommandsTaskCov_CmdTask_Ugly_UnknownAction — an unrecognised action prints
// usage and returns the unknown-command error.
func TestCommandsTaskCov_CmdTask_Ugly_UnknownAction(t *testing.T) {
	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdTask(core.NewOptions(core.Option{Key: "action", Value: "explode"}))
		core.AssertFalse(t, r.OK)
		core.AssertContains(t, r.Value.(error).Error(), "unknown task command")
	})
	core.AssertContains(t, output, "core-agent task toggle")
}

// TestCommandsTaskCov_CmdTaskUpdate_Ugly_UnknownPlan — updating a task in a
// non-existent plan surfaces the handler error.
func TestCommandsTaskCov_CmdTaskUpdate_Ugly_UnknownPlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdTaskUpdate(core.NewOptions(
			core.Option{Key: "plan_slug", Value: "no-such-plan"},
			core.Option{Key: "phase_order", Value: 1},
			core.Option{Key: "task_identifier", Value: "1"},
			core.Option{Key: "status", Value: "completed"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsTaskCov_CmdTaskToggle_Ugly_UnknownPlan — toggling a task in a
// non-existent plan surfaces the handler error.
func TestCommandsTaskCov_CmdTaskToggle_Ugly_UnknownPlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdTaskToggle(core.NewOptions(
			core.Option{Key: "plan_slug", Value: "no-such-plan"},
			core.Option{Key: "phase_order", Value: 1},
			core.Option{Key: "task_identifier", Value: "1"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsTaskCov_CmdTaskToggle_Bad_MissingFields — missing required fields
// prints usage and returns the required-field error.
func TestCommandsTaskCov_CmdTaskToggle_Bad_MissingFields(t *testing.T) {
	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdTaskToggle(core.NewOptions(core.Option{Key: "phase_order", Value: 1}))
		core.AssertFalse(t, r.OK)
		core.AssertContains(t, r.Value.(error).Error(), "required")
	})
	core.AssertContains(t, output, "core-agent task toggle")
}

// TestCommandsTaskCov_RegisterTaskCommands_Ugly_DuplicateConflict — a second
// registration fails on the first duplicate command.
func TestCommandsTaskCov_RegisterTaskCommands_Ugly_DuplicateConflict(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireTrue(t, s.registerTaskCommands().OK)
	core.AssertFalse(t, s.registerTaskCommands().OK)
}
