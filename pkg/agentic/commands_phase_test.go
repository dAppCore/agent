// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsPhase_RegisterPhaseCommands_Good_AllRegistered(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	s.registerPhaseCommands()

	cmds := c.Commands()
	assert.Contains(t, cmds, "phase")
	assert.Contains(t, cmds, "agentic:phase")
	assert.Contains(t, cmds, "phase/get")
	assert.Contains(t, cmds, "agentic:phase/get")
	assert.Contains(t, cmds, "phase/update_status")
	assert.Contains(t, cmds, "agentic:phase/update_status")
	assert.Contains(t, cmds, "phase/update-status")
	assert.Contains(t, cmds, "agentic:phase/update-status")
	assert.Contains(t, cmds, "phase/add_checkpoint")
	assert.Contains(t, cmds, "agentic:phase/add_checkpoint")
	assert.Contains(t, cmds, "phase/add-checkpoint")
	assert.Contains(t, cmds, "agentic:phase/add-checkpoint")
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
	require.NoError(t, err)

	output := captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions(
			core.Option{Key: "action", Value: "get"},
			core.Option{Key: "_arg", Value: "phase-command-plan"},
			core.Option{Key: "phase", Value: 1},
		))
		assert.True(t, r.OK)
	})
	assert.Contains(t, output, "phase:  1")
	assert.Contains(t, output, "name:   Setup")
	assert.Contains(t, output, "status: pending")

	output = captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions(
			core.Option{Key: "action", Value: "update-status"},
			core.Option{Key: "_arg", Value: "phase-command-plan"},
			core.Option{Key: "phase", Value: 1},
			core.Option{Key: "status", Value: "completed"},
		))
		assert.True(t, r.OK)
	})
	assert.Contains(t, output, "status: completed")

	output = captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions(
			core.Option{Key: "action", Value: "add-checkpoint"},
			core.Option{Key: "_arg", Value: "phase-command-plan"},
			core.Option{Key: "phase", Value: 1},
			core.Option{Key: "note", Value: "Build passes"},
		))
		assert.True(t, r.OK)
	})
	assert.Contains(t, output, "checkpoints: 1")
}

func TestCommandsPhase_CmdPhase_Bad_MissingActionStillShowsUsage(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	output := captureStdout(t, func() {
		r := s.cmdPhase(core.NewOptions())
		assert.True(t, r.OK)
	})
	assert.Contains(t, output, "core-agent phase get")
}

func TestCommandsPhase_CmdPhase_Ugly_UnknownAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPhase(core.NewOptions(core.Option{Key: "action", Value: "explode"}))
	require.False(t, r.OK)
	assert.Contains(t, r.Value.(error).Error(), "unknown phase command")
}
