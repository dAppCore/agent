// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerPlanCommands() {
	c := s.Core()
	c.Command("plan", core.Command{Description: "Manage implementation plans", Action: s.cmdPlan})
	c.Command("plan/create", core.Command{Description: "Create an implementation plan or create one from a template", Action: s.cmdPlanCreate})
	c.Command("plan/list", core.Command{Description: "List implementation plans", Action: s.cmdPlanList})
	c.Command("plan/show", core.Command{Description: "Show an implementation plan", Action: s.cmdPlanShow})
	c.Command("plan/status", core.Command{Description: "Read or update an implementation plan status", Action: s.cmdPlanStatus})
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
