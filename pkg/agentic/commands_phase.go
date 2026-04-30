// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
)

func (s *PrepSubsystem) registerPhaseCommands() {
	c := s.Core()
	c.Command("phase", core.Command{Description: "Manage plan phases", Action: s.cmdPhase})
	c.Command("agentic:phase", core.Command{Description: "Manage plan phases", Action: s.cmdPhase})
	c.Command("phase/get", core.Command{Description: "Read a plan phase by slug and order", Action: s.cmdPhaseGet})
	c.Command("agentic:phase/get", core.Command{Description: "Read a plan phase by slug and order", Action: s.cmdPhaseGet})
	c.Command("phase/update_status", core.Command{Description: "Update a plan phase status by slug and order", Action: s.cmdPhaseUpdateStatus})
	c.Command("agentic:phase/update_status", core.Command{Description: "Update a plan phase status by slug and order", Action: s.cmdPhaseUpdateStatus})
	c.Command("phase/update-status", core.Command{Description: "Update a plan phase status by slug and order", Action: s.cmdPhaseUpdateStatus})
	c.Command("agentic:phase/update-status", core.Command{Description: "Update a plan phase status by slug and order", Action: s.cmdPhaseUpdateStatus})
	c.Command("phase/add_checkpoint", core.Command{Description: "Append a checkpoint note to a plan phase", Action: s.cmdPhaseAddCheckpoint})
	c.Command("agentic:phase/add_checkpoint", core.Command{Description: "Append a checkpoint note to a plan phase", Action: s.cmdPhaseAddCheckpoint})
	c.Command("phase/add-checkpoint", core.Command{Description: "Append a checkpoint note to a plan phase", Action: s.cmdPhaseAddCheckpoint})
	c.Command("agentic:phase/add-checkpoint", core.Command{Description: "Append a checkpoint note to a plan phase", Action: s.cmdPhaseAddCheckpoint})
}

func (s *PrepSubsystem) cmdPhase(options core.Options) core.Result {
	switch action := optionStringValue(options, "action"); action {
	case "get":
		return s.cmdPhaseGet(options)
	case "update_status", "update-status", "update":
		return s.cmdPhaseUpdateStatus(options)
	case "add_checkpoint", "add-checkpoint", "checkpoint":
		return s.cmdPhaseAddCheckpoint(options)
	case "":
		core.Print(nil, "usage: core-agent phase get <plan> --phase=1")
		core.Print(nil, "       core-agent phase update-status <plan> --phase=1 --status=completed [--reason=\"...\"]")
		core.Print(nil, "       core-agent phase add-checkpoint <plan> --phase=1 --note=\"Build passes\" [--context='{\"build\":\"ok\"}']")
		return core.Result{OK: true}
	default:
		core.Print(nil, "usage: core-agent phase get <plan> --phase=1")
		core.Print(nil, "       core-agent phase update-status <plan> --phase=1 --status=completed [--reason=\"...\"]")
		core.Print(nil, "       core-agent phase add-checkpoint <plan> --phase=1 --note=\"Build passes\" [--context='{\"build\":\"ok\"}']")
		return core.Result{Value: core.E("agentic.cmdPhase", core.Concat("unknown phase command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdPhaseGet(options core.Options) core.Result {
	result := s.handlePhaseGet(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan", "slug", "_arg")},
		core.Option{Key: "phase_order", Value: optionIntValue(options, "phase_order", "phase")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdPhaseGet", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(PhaseOutput)
	if !ok {
		err := core.E("agentic.cmdPhaseGet", "invalid phase get output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "phase:  %d", output.Phase.Number)
	core.Print(nil, "name:   %s", output.Phase.Name)
	core.Print(nil, "status: %s", output.Phase.Status)
	if output.Phase.Description != "" {
		core.Print(nil, "desc:   %s", output.Phase.Description)
	}
	if output.Phase.Notes != "" {
		core.Print(nil, "notes:  %s", output.Phase.Notes)
	}
	if len(output.Phase.Tasks) > 0 {
		core.Print(nil, "tasks:  %d", len(output.Phase.Tasks))
	}
	if len(output.Phase.Checkpoints) > 0 {
		core.Print(nil, "checkpoints: %d", len(output.Phase.Checkpoints))
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPhaseUpdateStatus(options core.Options) core.Result {
	result := s.handlePhaseUpdateStatus(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan", "slug", "_arg")},
		core.Option{Key: "phase_order", Value: optionIntValue(options, "phase_order", "phase")},
		core.Option{Key: "status", Value: optionStringValue(options, "status")},
		core.Option{Key: "reason", Value: optionStringValue(options, "reason")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdPhaseUpdateStatus", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(PhaseOutput)
	if !ok {
		err := core.E("agentic.cmdPhaseUpdateStatus", "invalid phase update output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "phase:  %d", output.Phase.Number)
	core.Print(nil, "name:   %s", output.Phase.Name)
	core.Print(nil, "status: %s", output.Phase.Status)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPhaseAddCheckpoint(options core.Options) core.Result {
	result := s.handlePhaseAddCheckpoint(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan", "slug", "_arg")},
		core.Option{Key: "phase_order", Value: optionIntValue(options, "phase_order", "phase")},
		core.Option{Key: "note", Value: optionStringValue(options, "note")},
		core.Option{Key: "context", Value: optionAnyMapValue(options, "context")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdPhaseAddCheckpoint", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(PhaseOutput)
	if !ok {
		err := core.E("agentic.cmdPhaseAddCheckpoint", "invalid phase checkpoint output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "phase:  %d", output.Phase.Number)
	core.Print(nil, "name:   %s", output.Phase.Name)
	core.Print(nil, "status: %s", output.Phase.Status)
	core.Print(nil, "checkpoints: %d", len(output.Phase.Checkpoints))
	return core.Result{Value: output, OK: true}
}
