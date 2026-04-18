// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerSprintCommands() {
	c := s.Core()
	c.Command("sprint", core.Command{Description: "Manage tracked platform sprints", Action: s.cmdSprint})
	c.Command("agentic:sprint", core.Command{Description: "Manage tracked platform sprints", Action: s.cmdSprint})
	c.Command("sprint/create", core.Command{Description: "Create a tracked platform sprint", Action: s.cmdSprintCreate})
	c.Command("agentic:sprint/create", core.Command{Description: "Create a tracked platform sprint", Action: s.cmdSprintCreate})
	c.Command("sprint/get", core.Command{Description: "Read a tracked platform sprint by slug or ID", Action: s.cmdSprintGet})
	c.Command("agentic:sprint/get", core.Command{Description: "Read a tracked platform sprint by slug or ID", Action: s.cmdSprintGet})
	c.Command("sprint/list", core.Command{Description: "List tracked platform sprints", Action: s.cmdSprintList})
	c.Command("agentic:sprint/list", core.Command{Description: "List tracked platform sprints", Action: s.cmdSprintList})
	c.Command("sprint/update", core.Command{Description: "Update a tracked platform sprint", Action: s.cmdSprintUpdate})
	c.Command("agentic:sprint/update", core.Command{Description: "Update a tracked platform sprint", Action: s.cmdSprintUpdate})
	c.Command("sprint/archive", core.Command{Description: "Archive a tracked platform sprint", Action: s.cmdSprintArchive})
	c.Command("agentic:sprint/archive", core.Command{Description: "Archive a tracked platform sprint", Action: s.cmdSprintArchive})
}

func (s *PrepSubsystem) cmdSprint(options core.Options) core.Result {
	action := optionStringValue(options, "action")
	switch action {
	case "create":
		return s.cmdSprintCreate(options)
	case "get", "show":
		return s.cmdSprintGet(options)
	case "list":
		return s.cmdSprintList(options)
	case "update":
		return s.cmdSprintUpdate(options)
	case "archive", "delete":
		return s.cmdSprintArchive(options)
	case "":
		core.Print(nil, "usage: core-agent sprint create --title=\"AX Follow-up\" [--goal=\"Finish RFC parity\"] [--status=active]")
		core.Print(nil, "       core-agent sprint get <slug-or-id>")
		core.Print(nil, "       core-agent sprint list [--status=active] [--limit=10]")
		core.Print(nil, "       core-agent sprint update <slug-or-id> [--title=\"...\"] [--goal=\"...\"] [--status=completed]")
		core.Print(nil, "       core-agent sprint archive <slug-or-id>")
		return core.Result{OK: true}
	default:
		core.Print(nil, "usage: core-agent sprint create --title=\"AX Follow-up\" [--goal=\"Finish RFC parity\"] [--status=active]")
		core.Print(nil, "       core-agent sprint get <slug-or-id>")
		core.Print(nil, "       core-agent sprint list [--status=active] [--limit=10]")
		core.Print(nil, "       core-agent sprint update <slug-or-id> [--title=\"...\"] [--goal=\"...\"] [--status=completed]")
		core.Print(nil, "       core-agent sprint archive <slug-or-id>")
		return core.Result{Value: core.E("agentic.cmdSprint", core.Concat("unknown sprint command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdSprintCreate(options core.Options) core.Result {
	title := optionStringValue(options, "title")
	if title == "" {
		core.Print(nil, "usage: core-agent sprint create --title=\"AX Follow-up\" [--goal=\"Finish RFC parity\"] [--status=active]")
		return core.Result{Value: core.E("agentic.cmdSprintCreate", "title is required", nil), OK: false}
	}

	result := s.handleSprintCreate(s.commandContext(), core.NewOptions(
		core.Option{Key: "title", Value: title},
		core.Option{Key: "goal", Value: optionStringValue(options, "goal")},
		core.Option{Key: "status", Value: optionStringValue(options, "status")},
		core.Option{Key: "metadata", Value: optionAnyMapValue(options, "metadata")},
		core.Option{Key: "started_at", Value: optionStringValue(options, "started_at", "started-at")},
		core.Option{Key: "ended_at", Value: optionStringValue(options, "ended_at", "ended-at")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSprintCreate", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SprintOutput)
	if !ok {
		err := core.E("agentic.cmdSprintCreate", "invalid sprint create output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "slug:  %s", output.Sprint.Slug)
	core.Print(nil, "title: %s", output.Sprint.Title)
	core.Print(nil, "status: %s", output.Sprint.Status)
	if output.Sprint.Goal != "" {
		core.Print(nil, "goal:  %s", output.Sprint.Goal)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdSprintGet(options core.Options) core.Result {
	identifier := optionStringValue(options, "slug", "id", "_arg")
	if identifier == "" {
		core.Print(nil, "usage: core-agent sprint get <slug-or-id>")
		return core.Result{Value: core.E("agentic.cmdSprintGet", "id or slug is required", nil), OK: false}
	}

	result := s.handleSprintGet(s.commandContext(), core.NewOptions(
		core.Option{Key: "slug", Value: optionStringValue(options, "slug")},
		core.Option{Key: "id", Value: optionStringValue(options, "id", "_arg")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSprintGet", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SprintOutput)
	if !ok {
		err := core.E("agentic.cmdSprintGet", "invalid sprint get output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "slug:  %s", output.Sprint.Slug)
	core.Print(nil, "title: %s", output.Sprint.Title)
	core.Print(nil, "status: %s", output.Sprint.Status)
	if output.Sprint.Goal != "" {
		core.Print(nil, "goal:  %s", output.Sprint.Goal)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdSprintList(options core.Options) core.Result {
	result := s.handleSprintList(s.commandContext(), core.NewOptions(
		core.Option{Key: "status", Value: optionStringValue(options, "status")},
		core.Option{Key: "limit", Value: optionIntValue(options, "limit")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSprintList", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SprintListOutput)
	if !ok {
		err := core.E("agentic.cmdSprintList", "invalid sprint list output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Count == 0 {
		core.Print(nil, "no sprints")
		return core.Result{Value: output, OK: true}
	}

	for _, sprint := range output.Sprints {
		core.Print(nil, "  %-10s %-24s %s", sprint.Status, sprint.Slug, sprint.Title)
	}
	core.Print(nil, "%d sprint(s)", output.Count)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdSprintUpdate(options core.Options) core.Result {
	identifier := optionStringValue(options, "slug", "id", "_arg")
	if identifier == "" {
		core.Print(nil, "usage: core-agent sprint update <slug-or-id> [--title=\"...\"] [--goal=\"...\"] [--status=completed]")
		return core.Result{Value: core.E("agentic.cmdSprintUpdate", "id or slug is required", nil), OK: false}
	}

	result := s.handleSprintUpdate(s.commandContext(), core.NewOptions(
		core.Option{Key: "slug", Value: optionStringValue(options, "slug")},
		core.Option{Key: "id", Value: optionStringValue(options, "id", "_arg")},
		core.Option{Key: "title", Value: optionStringValue(options, "title")},
		core.Option{Key: "goal", Value: optionStringValue(options, "goal")},
		core.Option{Key: "status", Value: optionStringValue(options, "status")},
		core.Option{Key: "metadata", Value: optionAnyMapValue(options, "metadata")},
		core.Option{Key: "started_at", Value: optionStringValue(options, "started_at", "started-at")},
		core.Option{Key: "ended_at", Value: optionStringValue(options, "ended_at", "ended-at")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSprintUpdate", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SprintOutput)
	if !ok {
		err := core.E("agentic.cmdSprintUpdate", "invalid sprint update output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "slug:  %s", output.Sprint.Slug)
	core.Print(nil, "title: %s", output.Sprint.Title)
	core.Print(nil, "status: %s", output.Sprint.Status)
	if output.Sprint.Goal != "" {
		core.Print(nil, "goal:  %s", output.Sprint.Goal)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdSprintArchive(options core.Options) core.Result {
	identifier := optionStringValue(options, "slug", "id", "_arg")
	if identifier == "" {
		core.Print(nil, "usage: core-agent sprint archive <slug-or-id>")
		return core.Result{Value: core.E("agentic.cmdSprintArchive", "id or slug is required", nil), OK: false}
	}

	result := s.handleSprintArchive(s.commandContext(), core.NewOptions(
		core.Option{Key: "slug", Value: optionStringValue(options, "slug")},
		core.Option{Key: "id", Value: optionStringValue(options, "id", "_arg")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSprintArchive", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SprintArchiveOutput)
	if !ok {
		err := core.E("agentic.cmdSprintArchive", "invalid sprint archive output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "archived: %s", output.Archived)
	return core.Result{Value: output, OK: true}
}
