// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerTaskCommands() {
	c := s.Core()
	c.Command("task", core.Command{Description: "Manage plan tasks", Action: s.cmdTask})
	c.Command("agentic:task", core.Command{Description: "Manage plan tasks", Action: s.cmdTask})
	c.Command("task/create", core.Command{Description: "Create a task in a plan phase", Action: s.cmdTaskCreate})
	c.Command("agentic:task/create", core.Command{Description: "Create a task in a plan phase", Action: s.cmdTaskCreate})
	c.Command("task/update", core.Command{Description: "Update a plan task status, notes, priority, or category", Action: s.cmdTaskUpdate})
	c.Command("agentic:task/update", core.Command{Description: "Update a plan task status, notes, priority, or category", Action: s.cmdTaskUpdate})
	c.Command("task/toggle", core.Command{Description: "Toggle a plan task between pending and completed", Action: s.cmdTaskToggle})
	c.Command("agentic:task/toggle", core.Command{Description: "Toggle a plan task between pending and completed", Action: s.cmdTaskToggle})
}

func (s *PrepSubsystem) cmdTask(options core.Options) core.Result {
	action := optionStringValue(options, "action", "_arg")
	switch action {
	case "create":
		return s.cmdTaskCreate(options)
	case "toggle":
		return s.cmdTaskToggle(options)
	case "update":
		return s.cmdTaskUpdate(options)
	case "":
		core.Print(nil, "usage: core-agent task update <plan> --phase=1 --task=1 [--status=completed] [--notes=\"Done\"] [--priority=high] [--category=security] [--file=pkg/agentic/task.go|--file-ref=pkg/agentic/task.go] [--line=42|--line-ref=42]")
		core.Print(nil, "       core-agent task create <plan> --phase=1 --title=\"Review RFC\" [--description=\"...\"] [--status=pending] [--notes=\"...\"] [--priority=high] [--category=security] [--file=pkg/agentic/task.go|--file-ref=pkg/agentic/task.go] [--line=42|--line-ref=42]")
		core.Print(nil, "       core-agent task toggle <plan> --phase=1 --task=1")
		return core.Result{OK: true}
	default:
		core.Print(nil, "usage: core-agent task update <plan> --phase=1 --task=1 [--status=completed] [--notes=\"Done\"] [--priority=high] [--category=security] [--file=pkg/agentic/task.go|--file-ref=pkg/agentic/task.go] [--line=42|--line-ref=42]")
		core.Print(nil, "       core-agent task create <plan> --phase=1 --title=\"Review RFC\" [--description=\"...\"] [--status=pending] [--notes=\"...\"] [--priority=high] [--category=security] [--file=pkg/agentic/task.go|--file-ref=pkg/agentic/task.go] [--line=42|--line-ref=42]")
		core.Print(nil, "       core-agent task toggle <plan> --phase=1 --task=1")
		return core.Result{Value: core.E("agentic.cmdTask", core.Concat("unknown task command: ", action), nil), OK: false}
	}
}

func (s *PrepSubsystem) cmdTaskCreate(options core.Options) core.Result {
	planSlug := optionStringValue(options, "plan_slug", "plan", "slug", "_arg")
	phaseOrder := optionIntValue(options, "phase_order", "phase")
	title := optionStringValue(options, "title", "task")

	if planSlug == "" || phaseOrder == 0 || title == "" {
		core.Print(nil, "usage: core-agent task create <plan> --phase=1 --title=\"Review RFC\" [--description=\"...\"] [--status=pending] [--notes=\"...\"] [--priority=high] [--category=security] [--file=pkg/agentic/task.go] [--line=42]")
		return core.Result{Value: core.E("agentic.cmdTaskCreate", "plan_slug, phase_order, and title are required", nil), OK: false}
	}

	result := s.handleTaskCreate(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: planSlug},
		core.Option{Key: "phase_order", Value: phaseOrder},
		core.Option{Key: "title", Value: title},
		core.Option{Key: "description", Value: optionStringValue(options, "description")},
		core.Option{Key: "status", Value: optionStringValue(options, "status")},
		core.Option{Key: "notes", Value: optionStringValue(options, "notes")},
		core.Option{Key: "priority", Value: optionStringValue(options, "priority")},
		core.Option{Key: "category", Value: optionStringValue(options, "category")},
		core.Option{Key: "file", Value: optionStringValue(options, "file")},
		core.Option{Key: "line", Value: optionIntValue(options, "line")},
		core.Option{Key: "file_ref", Value: optionStringValue(options, "file_ref", "file-ref")},
		core.Option{Key: "line_ref", Value: optionIntValue(options, "line_ref", "line-ref")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdTaskCreate", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(TaskCreateOutput)
	if !ok {
		err := core.E("agentic.cmdTaskCreate", "invalid task create output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "task:   %s", output.Task.Title)
	core.Print(nil, "status: %s", output.Task.Status)
	if output.Task.Priority != "" {
		core.Print(nil, "priority: %s", output.Task.Priority)
	}
	if output.Task.Category != "" {
		core.Print(nil, "category: %s", output.Task.Category)
	}
	if output.Task.File != "" {
		core.Print(nil, "file:   %s", output.Task.File)
	}
	if output.Task.Line > 0 {
		core.Print(nil, "line:   %d", output.Task.Line)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdTaskUpdate(options core.Options) core.Result {
	planSlug := optionStringValue(options, "plan_slug", "plan", "slug", "_arg")
	phaseOrder := optionIntValue(options, "phase_order", "phase")
	taskIdentifier := optionAnyValue(options, "task_identifier", "task")

	if planSlug == "" || phaseOrder == 0 || taskIdentifierValue(taskIdentifier) == "" {
		core.Print(nil, "usage: core-agent task update <plan> --phase=1 --task=1 [--status=completed] [--notes=\"Done\"] [--priority=high] [--category=security] [--file=pkg/agentic/task.go] [--line=42]")
		return core.Result{Value: core.E("agentic.cmdTaskUpdate", "plan_slug, phase_order, and task_identifier are required", nil), OK: false}
	}

	result := s.handleTaskUpdate(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: planSlug},
		core.Option{Key: "phase_order", Value: phaseOrder},
		core.Option{Key: "task_identifier", Value: taskIdentifier},
		core.Option{Key: "status", Value: optionStringValue(options, "status")},
		core.Option{Key: "notes", Value: optionStringValue(options, "notes")},
		core.Option{Key: "priority", Value: optionStringValue(options, "priority")},
		core.Option{Key: "category", Value: optionStringValue(options, "category")},
		core.Option{Key: "file", Value: optionStringValue(options, "file")},
		core.Option{Key: "line", Value: optionIntValue(options, "line")},
		core.Option{Key: "file_ref", Value: optionStringValue(options, "file_ref", "file-ref")},
		core.Option{Key: "line_ref", Value: optionIntValue(options, "line_ref", "line-ref")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdTaskUpdate", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(TaskOutput)
	if !ok {
		err := core.E("agentic.cmdTaskUpdate", "invalid task update output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "task:   %s", output.Task.Title)
	core.Print(nil, "status: %s", output.Task.Status)
	if output.Task.Notes != "" {
		core.Print(nil, "notes:  %s", output.Task.Notes)
	}
	if output.Task.Priority != "" {
		core.Print(nil, "priority: %s", output.Task.Priority)
	}
	if output.Task.Category != "" {
		core.Print(nil, "category: %s", output.Task.Category)
	}
	if output.Task.File != "" {
		core.Print(nil, "file:   %s", output.Task.File)
	}
	if output.Task.Line > 0 {
		core.Print(nil, "line:   %d", output.Task.Line)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdTaskToggle(options core.Options) core.Result {
	planSlug := optionStringValue(options, "plan_slug", "plan", "slug", "_arg")
	phaseOrder := optionIntValue(options, "phase_order", "phase")
	taskIdentifier := optionAnyValue(options, "task_identifier", "task")

	if planSlug == "" || phaseOrder == 0 || taskIdentifierValue(taskIdentifier) == "" {
		core.Print(nil, "usage: core-agent task toggle <plan> --phase=1 --task=1")
		return core.Result{Value: core.E("agentic.cmdTaskToggle", "plan_slug, phase_order, and task_identifier are required", nil), OK: false}
	}

	result := s.handleTaskToggle(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: planSlug},
		core.Option{Key: "phase_order", Value: phaseOrder},
		core.Option{Key: "task_identifier", Value: taskIdentifier},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdTaskToggle", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(TaskOutput)
	if !ok {
		err := core.E("agentic.cmdTaskToggle", "invalid task toggle output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "task:   %s", output.Task.Title)
	core.Print(nil, "status: %s", output.Task.Status)
	return core.Result{Value: output, OK: true}
}
