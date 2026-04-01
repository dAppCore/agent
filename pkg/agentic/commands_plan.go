// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerPlanCommands() {
	c := s.Core()
	c.Command("plan", core.Command{Description: "Manage implementation plans", Action: s.cmdPlan})
	c.Command("agentic:plan", core.Command{Description: "Manage implementation plans", Action: s.cmdPlan})
	c.Command("plan/create", core.Command{Description: "Create an implementation plan or create one from a template", Action: s.cmdPlanCreate})
	c.Command("plan/from-issue", core.Command{Description: "Create an implementation plan from a tracked issue", Action: s.cmdPlanFromIssue})
	c.Command("plan/list", core.Command{Description: "List implementation plans", Action: s.cmdPlanList})
	c.Command("agentic:plan/read", core.Command{Description: "Read an implementation plan", Action: s.cmdPlanShow})
	c.Command("plan/read", core.Command{Description: "Read an implementation plan", Action: s.cmdPlanShow})
	c.Command("plan/show", core.Command{Description: "Show an implementation plan", Action: s.cmdPlanShow})
	c.Command("plan/status", core.Command{Description: "Read or update an implementation plan status", Action: s.cmdPlanStatus})
	c.Command("plan/check", core.Command{Description: "Check whether a plan or phase is complete", Action: s.cmdPlanCheck})
	c.Command("plan/archive", core.Command{Description: "Archive an implementation plan by slug or ID", Action: s.cmdPlanArchive})
	c.Command("plan/delete", core.Command{Description: "Delete an implementation plan by ID", Action: s.cmdPlanDelete})
}

func (s *PrepSubsystem) cmdPlan(options core.Options) core.Result {
	return s.cmdPlanList(options)
}

func (s *PrepSubsystem) cmdPlanCreate(options core.Options) core.Result {
	ctx := s.commandContext()
	slug := optionStringValue(options, "slug", "_arg")
	title := optionStringValue(options, "title")
	objective := optionStringValue(options, "objective")
	description := optionStringValue(options, "description")
	templateName := templateNameValue(optionStringValue(options, "template"), optionStringValue(options, "template_slug", "template-slug"), optionStringValue(options, "import"))

	if templateName != "" {
		variables := optionStringMapValue(options, "variables")
		if variables == nil {
			variables = map[string]string{}
		}

		_, output, err := s.templateCreatePlan(ctx, nil, TemplateCreatePlanInput{
			Template:     templateName,
			Variables:    variables,
			Slug:         slug,
			Title:        title,
			Activate:     optionBoolValue(options, "activate"),
			TemplateSlug: templateName,
		})
		if err != nil {
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}

		core.Print(nil, "created: %s", output.Plan.Slug)
		core.Print(nil, "title:   %s", output.Plan.Title)
		core.Print(nil, "status:  %s", output.Plan.Status)
		return core.Result{Value: output, OK: true}
	}

	if title == "" {
		core.Print(nil, "usage: core-agent plan create <slug> --title=\"My Plan\" [--objective=\"...\"] [--description=\"...\"] [--import=bug-fix] [--activate]")
		return core.Result{Value: core.E("agentic.cmdPlanCreate", "title is required", nil), OK: false}
	}

	if objective == "" {
		objective = description
	}
	if objective == "" {
		objective = title
	}

	_, output, err := s.planCreate(ctx, nil, PlanCreateInput{
		Title:       title,
		Slug:        slug,
		Objective:   objective,
		Description: description,
		Context:     optionAnyMapValue(options, "context"),
		Repo:        optionStringValue(options, "repo"),
		Org:         optionStringValue(options, "org"),
		Phases:      planPhasesValue(options, "phases"),
		Notes:       optionStringValue(options, "notes"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "created: %s", output.ID)
	core.Print(nil, "path:    %s", output.Path)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPlanFromIssue(options core.Options) core.Result {
	ctx := s.commandContext()
	identifier := optionStringValue(options, "slug", "_arg")
	if identifier == "" {
		identifier = optionStringValue(options, "id")
	}
	if identifier == "" {
		core.Print(nil, "usage: core-agent plan from-issue <slug> [--id=N]")
		return core.Result{Value: core.E("agentic.cmdPlanFromIssue", "issue slug or id is required", nil), OK: false}
	}

	_, output, err := s.planFromIssue(ctx, nil, PlanFromIssueInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "created: %s", output.Plan.Slug)
	core.Print(nil, "issue:   #%d %s", output.Issue.ID, output.Issue.Title)
	core.Print(nil, "path:    %s", output.Path)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPlanList(options core.Options) core.Result {
	ctx := s.commandContext()
	_, output, err := s.planList(ctx, nil, PlanListInput{
		Status: optionStringValue(options, "status"),
		Repo:   optionStringValue(options, "repo"),
		Limit:  optionIntValue(options, "limit"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Count == 0 {
		core.Print(nil, "no plans")
		return core.Result{Value: output, OK: true}
	}

	for _, plan := range output.Plans {
		core.Print(nil, "  %-10s %-24s %s", plan.Status, plan.Slug, plan.Title)
	}
	core.Print(nil, "%d plan(s)", output.Count)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPlanShow(options core.Options) core.Result {
	ctx := s.commandContext()
	slug := optionStringValue(options, "slug", "_arg")
	if slug == "" {
		core.Print(nil, "usage: core-agent plan show <slug>")
		return core.Result{Value: core.E("agentic.cmdPlanShow", "slug is required", nil), OK: false}
	}

	_, output, err := s.planGetCompat(ctx, nil, PlanReadInput{Slug: slug})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "slug:        %s", output.Plan.Slug)
	core.Print(nil, "title:       %s", output.Plan.Title)
	core.Print(nil, "status:      %s", output.Plan.Status)
	core.Print(nil, "progress:    %d/%d (%d%%)", output.Plan.Progress.Completed, output.Plan.Progress.Total, output.Plan.Progress.Percentage)
	if output.Plan.TemplateVersion.Slug != "" {
		core.Print(nil, "template:    %s v%d", output.Plan.TemplateVersion.Slug, output.Plan.TemplateVersion.Version)
		if output.Plan.TemplateVersion.ContentHash != "" {
			core.Print(nil, "template id: %s", output.Plan.TemplateVersion.ContentHash)
		}
	}
	if output.Plan.Description != "" {
		core.Print(nil, "description: %s", output.Plan.Description)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPlanStatus(options core.Options) core.Result {
	ctx := s.commandContext()
	slug := optionStringValue(options, "slug", "_arg")
	if slug == "" {
		core.Print(nil, "usage: core-agent plan status <slug> [--set=ready]")
		return core.Result{Value: core.E("agentic.cmdPlanStatus", "slug is required", nil), OK: false}
	}

	set := optionStringValue(options, "set", "status")
	if set == "" {
		_, output, err := s.planGetCompat(ctx, nil, PlanReadInput{Slug: slug})
		if err != nil {
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}
		core.Print(nil, "slug:   %s", output.Plan.Slug)
		core.Print(nil, "status: %s", output.Plan.Status)
		return core.Result{Value: output, OK: true}
	}

	_, output, err := s.planUpdateStatusCompat(ctx, nil, PlanStatusUpdateInput{Slug: slug, Status: set})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "slug:   %s", output.Plan.Slug)
	core.Print(nil, "status: %s", output.Plan.Status)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPlanCheck(options core.Options) core.Result {
	ctx := s.commandContext()
	slug := optionStringValue(options, "slug", "_arg")
	if slug == "" {
		core.Print(nil, "usage: core-agent plan check <slug> [--phase=1]")
		return core.Result{Value: core.E("agentic.cmdPlanCheck", "slug is required", nil), OK: false}
	}

	result := s.handlePlanCheck(ctx, core.NewOptions(
		core.Option{Key: "slug", Value: slug},
		core.Option{Key: "phase", Value: optionIntValue(options, "phase", "phase_order")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdPlanCheck", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	check, ok := result.Value.(PlanCheckOutput)
	if !ok {
		err := core.E("agentic.cmdPlanCheck", "invalid plan check output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "slug:     %s", check.Plan.Slug)
	core.Print(nil, "status:   %s", check.Plan.Status)
	core.Print(nil, "progress: %d/%d (%d%%)", check.Plan.Progress.Completed, check.Plan.Progress.Total, check.Plan.Progress.Percentage)
	if check.Phase > 0 {
		core.Print(nil, "phase:    %d %s", check.Phase, check.PhaseName)
	}
	if len(check.Pending) > 0 {
		core.Print(nil, "pending:")
		for _, item := range check.Pending {
			core.Print(nil, "  - %s", item)
		}
	}
	if check.Complete {
		core.Print(nil, "complete")
	} else {
		core.Print(nil, "incomplete")
	}

	if !check.Complete {
		return core.Result{Value: check, OK: false}
	}
	return core.Result{Value: check, OK: true}
}

func (s *PrepSubsystem) cmdPlanArchive(options core.Options) core.Result {
	ctx := s.commandContext()
	id := optionStringValue(options, "id", "slug", "_arg")
	if id == "" {
		core.Print(nil, "usage: core-agent plan archive <slug> [--reason=\"...\"]")
		return core.Result{Value: core.E("agentic.cmdPlanArchive", "slug or id is required", nil), OK: false}
	}

	result := s.handlePlanArchive(ctx, core.NewOptions(
		core.Option{Key: "slug", Value: id},
		core.Option{Key: "reason", Value: optionStringValue(options, "reason")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdPlanArchive", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(PlanArchiveOutput)
	if !ok {
		err := core.E("agentic.cmdPlanArchive", "invalid plan archive output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "archived: %s", output.Archived)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPlanDelete(options core.Options) core.Result {
	ctx := s.commandContext()
	id := optionStringValue(options, "id", "_arg")
	if id == "" {
		core.Print(nil, "usage: core-agent plan delete <id> [--reason=\"...\"]")
		return core.Result{Value: core.E("agentic.cmdPlanDelete", "id is required", nil), OK: false}
	}

	result := s.handlePlanDelete(ctx, core.NewOptions(
		core.Option{Key: "id", Value: id},
		core.Option{Key: "reason", Value: optionStringValue(options, "reason")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdPlanDelete", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(PlanDeleteOutput)
	if !ok {
		err := core.E("agentic.cmdPlanDelete", "invalid plan delete output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "deleted: %s", output.Deleted)
	return core.Result{Value: output, OK: true}
}

func planCheckOutput(plan PlanCompatibilityView, phaseOrder int) PlanCheckOutput {
	output := PlanCheckOutput{
		Success: true,
		Plan:    plan,
	}

	if phaseOrder <= 0 {
		output.Complete, output.Pending = planCompleteOutput(plan.Phases)
		return output
	}

	for _, phase := range plan.Phases {
		if phase.Number != phaseOrder {
			continue
		}
		output.Phase = phase.Number
		output.PhaseName = phase.Name
		output.Complete, output.Pending = phaseCompleteOutput(phase)
		return output
	}

	output.Complete = false
	output.Pending = []string{core.Concat("phase ", core.Sprint(phaseOrder), " not found")}
	return output
}

func planCompleteOutput(phases []Phase) (bool, []string) {
	var pending []string
	for _, phase := range phases {
		phaseComplete, phasePending := phaseCompleteOutput(phase)
		if phaseComplete {
			continue
		}
		if len(phasePending) == 0 {
			pending = append(pending, core.Concat("phase ", core.Sprint(phase.Number), ": ", phase.Name))
			continue
		}
		for _, item := range phasePending {
			pending = append(pending, core.Concat("phase ", core.Sprint(phase.Number), ": ", item))
		}
	}
	return len(pending) == 0, pending
}

func phaseCompleteOutput(phase Phase) (bool, []string) {
	tasks := phaseTaskList(phase)
	if len(tasks) == 0 {
		switch phase.Status {
		case "completed", "done", "approved":
			return true, nil
		default:
			return false, []string{phase.Name}
		}
	}

	var pending []string
	for _, task := range tasks {
		if task.Status == "completed" {
			continue
		}
		label := task.Title
		if label == "" {
			label = task.ID
		}
		pending = append(pending, label)
	}
	return len(pending) == 0, pending
}
