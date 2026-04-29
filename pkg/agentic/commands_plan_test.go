// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestCommandsPlan_CmdPlanCheck_Good_CompletePlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Check Plan",
		Description: "Confirm the plan check command reports completion",
		Phases: []Phase{
			{
				Name: "Setup",
				Tasks: []PlanTask{
					{ID: "1", Title: "Review RFC", Status: "completed"},
				},
			},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	r := s.cmdPlanCheck(core.NewOptions(core.Option{Key: "_arg", Value: plan.Slug}))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(PlanCheckOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertTrue(t, output.Complete)
	core.AssertEmpty(t, output.Pending)
	core.AssertEqual(t, plan.Slug, output.Plan.Slug)
}

func TestCommandsPlan_CmdPlanCheck_Bad_MissingSlug(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdPlanCheck(core.NewOptions())

	core.AssertFalse(t, r.OK)
	core.AssertError(t, r.Value.(error))
	core.AssertContains(t, r.Value.(error).Error(), "slug is required")
}

func TestCommandsPlan_CmdPlanCheck_Ugly_IncompletePhase(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Incomplete Plan",
		Description: "Leave one task pending",
		Phases: []Phase{
			{
				Number: 1,
				Name:   "Setup",
				Tasks: []PlanTask{
					{ID: "1", Title: "Review RFC", Status: "completed"},
					{ID: "2", Title: "Patch code", Status: "pending"},
				},
			},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	r := s.cmdPlanCheck(core.NewOptions(
		core.Option{Key: "slug", Value: plan.Slug},
		core.Option{Key: "phase", Value: 1},
	))

	core.AssertFalse(t, r.OK)
	output, ok := r.Value.(PlanCheckOutput)
	core.RequireTrue(t, ok)
	core.AssertFalse(t, output.Complete)
	core.AssertEqual(t, 1, output.Phase)
	core.AssertEqual(t, "Setup", output.PhaseName)
	core.AssertEqual(t, []string{"Patch code"}, output.Pending)
}

func TestCommandsPlan_CmdPlan_Good_RoutesCreate(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	r := s.cmdPlan(core.NewOptions(
		core.Option{Key: "action", Value: "create"},
		core.Option{Key: "slug", Value: "root-route-plan"},
		core.Option{Key: "title", Value: "Root Route Plan"},
		core.Option{Key: "objective", Value: "Exercise the root plan router"},
	))

	core.RequireTrue(t, r.OK)
	output, ok := r.Value.(PlanCreateOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertNotEmpty(t, output.ID)
	core.AssertNotEmpty(t, output.Path)
}

func TestCommandsPlan_CmdPlan_Good_RoutesStatus(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Status Route Plan",
		Description: "Exercise the root plan router status action",
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	r := s.cmdPlan(core.NewOptions(
		core.Option{Key: "action", Value: "status"},
		core.Option{Key: "slug", Value: plan.Slug},
	))

	core.RequireTrue(t, r.OK)
	output, ok := r.Value.(PlanCompatibilityGetOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, plan.Slug, output.Plan.Slug)
}

func TestCommandsPlan_CmdPlan_Bad_UnknownAction(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdPlan(core.NewOptions(
		core.Option{Key: "action", Value: "does-not-exist"},
	))

	core.AssertFalse(t, r.OK)
	core.AssertError(t, r.Value.(error))
	core.AssertContains(t, r.Value.(error).Error(), "unknown plan command")
}

func TestCommandsPlan_CmdPlanUpdate_Good_StatusAndAgent(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Update Command",
		Objective: "Verify the plan update command",
	})
	core.RequireNoError(t, err)

	r := s.cmdPlanUpdate(core.NewOptions(
		core.Option{Key: "_arg", Value: created.ID},
		core.Option{Key: "status", Value: "ready"},
		core.Option{Key: "agent", Value: "codex"},
	))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(PlanUpdateOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, created.ID, output.Plan.ID)
	core.AssertEqual(t, "ready", output.Plan.Status)
	core.AssertEqual(t, "codex", output.Plan.Agent)
}

func TestCommandsPlan_CmdPlanUpdate_Bad_MissingFields(t *testing.T) {
	s := newTestPrep(t)

	r := s.cmdPlanUpdate(core.NewOptions(
		core.Option{Key: "_arg", Value: "plan-123"},
	))

	core.AssertFalse(t, r.OK)
	core.AssertError(t, r.Value.(error))
	core.AssertContains(t, r.Value.(error).Error(), "at least one update field is required")
}

func TestCommandsPlan_HandlePlanCheck_Good_CompletePlan(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Action Check Plan",
		Description: "Confirm the plan check action reports completion",
		Phases: []Phase{
			{
				Name: "Setup",
				Tasks: []PlanTask{
					{ID: "1", Title: "Review RFC", Status: "completed"},
				},
			},
		},
	})
	core.RequireNoError(t, err)

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)

	r := s.handlePlanCheck(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: plan.Slug},
	))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(PlanCheckOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertTrue(t, output.Complete)
	core.AssertEqual(t, plan.Slug, output.Plan.Slug)
}

func TestCommandsPlan_CmdPlanTemplates_Good_Case(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "")

	r := s.cmdPlanTemplates(core.NewOptions(
		core.Option{Key: "category", Value: "development"},
	))

	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(TemplateListOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	assertNotZero(t, output.Total)
}

func TestCommandsPlan_CmdPlanTemplates_Ugly_NoMatchingCategory(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "")

	r := s.cmdPlanTemplates(core.NewOptions(
		core.Option{Key: "category", Value: "does-not-exist"},
	))

	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(TemplateListOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	assertZero(t, output.Total)
	core.AssertEmpty(t, output.Templates)
}

func TestCommandsPlan_RegisterPlanCommands_Good_SpecAliasRegistered(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerPlanCommands()

	core.AssertContains(t, c.Commands(), "agentic:plan")
	core.AssertContains(t, c.Commands(), "plan")
	core.AssertContains(t, c.Commands(), "agentic:plan/templates")
	core.AssertContains(t, c.Commands(), "plan/templates")
	core.AssertContains(t, c.Commands(), "agentic:plan/create")
	core.AssertContains(t, c.Commands(), "agentic:plan/get")
	core.AssertContains(t, c.Commands(), "plan/get")
	core.AssertContains(t, c.Commands(), "agentic:plan/list")
	core.AssertContains(t, c.Commands(), "agentic:plan/read")
	core.AssertContains(t, c.Commands(), "plan/read")
	core.AssertContains(t, c.Commands(), "agentic:plan/show")
	core.AssertContains(t, c.Commands(), "plan/show")
	core.AssertContains(t, c.Commands(), "agentic:plan/status")
	core.AssertContains(t, c.Commands(), "plan/update")
	core.AssertContains(t, c.Commands(), "agentic:plan/update")
	core.AssertContains(t, c.Commands(), "plan/status")
	core.AssertContains(t, c.Commands(), "plan/update_status")
	core.AssertContains(t, c.Commands(), "agentic:plan/update_status")
	core.AssertContains(t, c.Commands(), "agentic:plan/check")
	core.AssertContains(t, c.Commands(), "plan/check")
	core.AssertContains(t, c.Commands(), "agentic:plan/archive")
	core.AssertContains(t, c.Commands(), "plan/archive")
	core.AssertContains(t, c.Commands(), "agentic:plan/delete")
	core.AssertContains(t, c.Commands(), "plan/delete")
}
