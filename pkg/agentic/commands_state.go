// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerStateCommands() {
	c := s.Core()
	c.Command("state", core.Command{Description: "Manage shared plan state", Action: s.cmdState})
	c.Command("agentic:state", core.Command{Description: "Manage shared plan state", Action: s.cmdState})
	c.Command("state/set", core.Command{Description: "Store shared plan state", Action: s.cmdStateSet})
	c.Command("agentic:state/set", core.Command{Description: "Store shared plan state", Action: s.cmdStateSet})
	c.Command("state/get", core.Command{Description: "Read shared plan state by key", Action: s.cmdStateGet})
	c.Command("agentic:state/get", core.Command{Description: "Read shared plan state by key", Action: s.cmdStateGet})
	c.Command("state/list", core.Command{Description: "List shared plan state for a plan", Action: s.cmdStateList})
	c.Command("agentic:state/list", core.Command{Description: "List shared plan state for a plan", Action: s.cmdStateList})
	c.Command("state/delete", core.Command{Description: "Delete shared plan state by key", Action: s.cmdStateDelete})
	c.Command("agentic:state/delete", core.Command{Description: "Delete shared plan state by key", Action: s.cmdStateDelete})
}

func (s *PrepSubsystem) cmdState(options core.Options) core.Result {
	switch action := optionStringValue(options, "action"); action {
	case "set":
		return s.cmdStateSet(options)
	case "get":
		return s.cmdStateGet(options)
	case "list":
		return s.cmdStateList(options)
	case "delete":
		return s.cmdStateDelete(options)
	case "":
		core.Print(nil, "usage: core-agent state set <plan> --key=pattern --value=observer [--type=general] [--description=\"Shared across sessions\"]")
		core.Print(nil, "       core-agent state get <plan> --key=pattern")
		core.Print(nil, "       core-agent state list <plan> [--type=general] [--category=general]")
		core.Print(nil, "       core-agent state delete <plan> --key=pattern")
		return core.Result{OK: true}
	default:
		core.Print(nil, "usage: core-agent state set <plan> --key=pattern --value=observer [--type=general] [--description=\"Shared across sessions\"]")
		core.Print(nil, "       core-agent state get <plan> --key=pattern")
		core.Print(nil, "       core-agent state list <plan> [--type=general] [--category=general]")
		core.Print(nil, "       core-agent state delete <plan> --key=pattern")
		return core.Result{Value: core.E("agentic.cmdState", core.Concat("unknown state command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdStateSet(options core.Options) core.Result {
	result := s.handleStateSet(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan", "slug", "_arg")},
		core.Option{Key: "key", Value: optionStringValue(options, "key")},
		core.Option{Key: "value", Value: optionAnyValue(options, "value")},
		core.Option{Key: "type", Value: optionStringValue(options, "type")},
		core.Option{Key: "description", Value: optionStringValue(options, "description")},
		core.Option{Key: "category", Value: optionStringValue(options, "category")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdStateSet", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(StateOutput)
	if !ok {
		err := core.E("agentic.cmdStateSet", "invalid state set output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "key:   %s", output.State.Key)
	core.Print(nil, "type:  %s", output.State.Type)
	if output.State.Description != "" {
		core.Print(nil, "desc:  %s", output.State.Description)
	}
	core.Print(nil, "value: %s", stateValueString(output.State.Value))
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdStateGet(options core.Options) core.Result {
	result := s.handleStateGet(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan", "slug", "_arg")},
		core.Option{Key: "key", Value: optionStringValue(options, "key")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdStateGet", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(StateOutput)
	if !ok {
		err := core.E("agentic.cmdStateGet", "invalid state get output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "key:   %s", output.State.Key)
	core.Print(nil, "type:  %s", output.State.Type)
	if output.State.Description != "" {
		core.Print(nil, "desc:  %s", output.State.Description)
	}
	if output.State.UpdatedAt != "" {
		core.Print(nil, "updated: %s", output.State.UpdatedAt)
	}
	core.Print(nil, "value: %s", stateValueString(output.State.Value))
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdStateList(options core.Options) core.Result {
	result := s.handleStateList(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan", "slug", "_arg")},
		core.Option{Key: "type", Value: optionStringValue(options, "type")},
		core.Option{Key: "category", Value: optionStringValue(options, "category")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdStateList", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(StateListOutput)
	if !ok {
		err := core.E("agentic.cmdStateList", "invalid state list output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Total == 0 {
		core.Print(nil, "no states")
		return core.Result{Value: output, OK: true}
	}

	for _, state := range output.States {
		core.Print(nil, "  %-20s %-12s %s", state.Key, state.Type, stateValueString(state.Value))
	}
	core.Print(nil, "%d state(s)", output.Total)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdStateDelete(options core.Options) core.Result {
	result := s.handleStateDelete(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan", "slug", "_arg")},
		core.Option{Key: "key", Value: optionStringValue(options, "key")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdStateDelete", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(StateDeleteOutput)
	if !ok {
		err := core.E("agentic.cmdStateDelete", "invalid state delete output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "deleted: %s", output.Deleted.Key)
	core.Print(nil, "type:    %s", output.Deleted.Type)
	return core.Result{Value: output, OK: true}
}

func stateValueString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}

	jsonValue := core.JSONMarshalString(value)
	if jsonValue != "" {
		return jsonValue
	}

	return core.Sprint(value)
}
