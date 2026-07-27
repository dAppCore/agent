// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// covPhasePlan seeds a plan whose first phase carries a description, notes,
// tasks and a checkpoint so the optional phase print lines can be exercised.
func covPhasePlan(t *testing.T) *PrepSubsystem {
	t.Helper()
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, _, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Phase coverage plan",
		Slug:      "phase-cov-plan",
		Objective: "Exercise phase command output",
		Phases: []Phase{{
			Number:      1,
			Name:        "Setup",
			Status:      "pending",
			Description: "Get the tree ready",
			Notes:       "Watch the imports",
			Tasks:       []PlanTask{{ID: "1", Title: "Read RFC", Status: "pending"}},
			Checkpoints: []PhaseCheckpoint{{Note: "kickoff"}},
		}},
	})
	core.RequireNoError(t, err)
	return s
}

// TestCommandsPhaseCov_CmdPhaseGet_Good_AllOptionalFields prints every optional
// phase field (desc/notes/tasks/checkpoints) for a richly-populated phase.
func TestCommandsPhaseCov_CmdPhaseGet_Good_AllOptionalFields(t *testing.T) {
	s := covPhasePlan(t)

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPhaseGet(core.NewOptions(
			core.Option{Key: "_arg", Value: "phase-cov-plan"},
			core.Option{Key: "phase", Value: 1},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, "phase:  1")
	core.AssertContains(t, output, "name:   Setup")
	core.AssertContains(t, output, "status: pending")
	core.AssertContains(t, output, "desc:   Get the tree ready")
	core.AssertContains(t, output, "notes:  Watch the imports")
	core.AssertContains(t, output, "tasks:  1")
	core.AssertContains(t, output, "checkpoints: 1")
}

// TestCommandsPhaseCov_CmdPhaseGet_Ugly_MissingPlan — an unknown plan slug
// surfaces the handler error.
func TestCommandsPhaseCov_CmdPhaseGet_Ugly_MissingPlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPhaseGet(core.NewOptions(
			core.Option{Key: "_arg", Value: "no-such-plan"},
			core.Option{Key: "phase", Value: 1},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsPhaseCov_CmdPhaseUpdateStatus_Good_PrintsStatus — updating a phase
// status prints the new status line.
func TestCommandsPhaseCov_CmdPhaseUpdateStatus_Good_PrintsStatus(t *testing.T) {
	s := covPhasePlan(t)

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPhaseUpdateStatus(core.NewOptions(
			core.Option{Key: "_arg", Value: "phase-cov-plan"},
			core.Option{Key: "phase", Value: 1},
			core.Option{Key: "status", Value: "completed"},
			core.Option{Key: "reason", Value: "all done"},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, "status: completed")
}

// TestCommandsPhaseCov_CmdPhaseUpdateStatus_Ugly_MissingPlan — an unknown plan
// surfaces the handler error.
func TestCommandsPhaseCov_CmdPhaseUpdateStatus_Ugly_MissingPlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPhaseUpdateStatus(core.NewOptions(
			core.Option{Key: "_arg", Value: "no-such-plan"},
			core.Option{Key: "phase", Value: 1},
			core.Option{Key: "status", Value: "completed"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsPhaseCov_CmdPhaseAddCheckpoint_Ugly_MissingPlan — an unknown plan
// surfaces the handler error.
func TestCommandsPhaseCov_CmdPhaseAddCheckpoint_Ugly_MissingPlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPhaseAddCheckpoint(core.NewOptions(
			core.Option{Key: "_arg", Value: "no-such-plan"},
			core.Option{Key: "phase", Value: 1},
			core.Option{Key: "note", Value: "build passes"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsPhaseCov_RegisterPhaseCommands_Ugly_DuplicateConflict — a second
// registration fails on the first duplicate command.
func TestCommandsPhaseCov_RegisterPhaseCommands_Ugly_DuplicateConflict(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireTrue(t, s.registerPhaseCommands().OK)
	core.AssertFalse(t, s.registerPhaseCommands().OK)
}
