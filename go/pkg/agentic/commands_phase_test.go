// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestCommandsPhase_RegisterPhaseCommands_Good_AllRegistered(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	s.registerPhaseCommands()

	cmds := c.Commands()
	core.AssertContains(t, cmds, "phase")
	core.AssertContains(t, cmds, "agentic:phase")
	core.AssertContains(t, cmds, "phase/get")
	core.AssertContains(t, cmds, "agentic:phase/get")
	core.AssertContains(t, cmds, "phase/update_status")
	core.AssertContains(t, cmds, "agentic:phase/update_status")
	core.AssertContains(t, cmds, "phase/update-status")
	core.AssertContains(t, cmds, "agentic:phase/update-status")
	core.AssertContains(t, cmds, "phase/add_checkpoint")
	core.AssertContains(t, cmds, "agentic:phase/add_checkpoint")
	core.AssertContains(t, cmds, "phase/add-checkpoint")
	core.AssertContains(t, cmds, "agentic:phase/add-checkpoint")
}

func TestCommandsPhase_CmdPhase_Good_GetUpdateCheckpoint(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	_, _, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Phase command plan",
		Slug:      "phase-command-plan",
		Objective: "Exercise phase commands",
		Phases: []Phase{
			{Number: 1, Name: "Setup", Status: "pending"},
		},
	})
	core.RequireNoError(t, err)

	output := captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions(
			core.Option{Key: "action", Value: "get"},
			core.Option{Key: "_arg", Value: "phase-command-plan"},
			core.Option{Key: "phase", Value: 1},
		))
		core.AssertTrue(t, r.OK)
	})
	core.AssertContains(t, output, "phase:  1")
	core.AssertContains(t, output, "name:   Setup")
	core.AssertContains(t, output, "status: pending")

	output = captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions(
			core.Option{Key: "action", Value: "update-status"},
			core.Option{Key: "_arg", Value: "phase-command-plan"},
			core.Option{Key: "phase", Value: 1},
			core.Option{Key: "status", Value: "completed"},
		))
		core.AssertTrue(t, r.OK)
	})
	core.AssertContains(t, output, "status: completed")

	output = captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions(
			core.Option{Key: "action", Value: "add-checkpoint"},
			core.Option{Key: "_arg", Value: "phase-command-plan"},
			core.Option{Key: "phase", Value: 1},
			core.Option{Key: "note", Value: "Build passes"},
		))
		core.AssertTrue(t, r.OK)
	})
	core.AssertContains(t, output, "checkpoints: 1")
}

func TestCommandsPhase_CmdPhase_Bad_MissingActionStillShowsUsage(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	output := captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions())
		core.AssertTrue(t, r.OK)
	})
	core.AssertContains(t, output, "core-agent phase get")
}

func TestCommandsPhase_CmdPhase_Ugly_UnknownAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPhase(core.NewOptions(core.Option{Key: "action", Value: "explode"}))
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "unknown phase command")
}
