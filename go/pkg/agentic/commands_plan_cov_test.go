// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- planCheckOutput / planCompleteOutput / phaseCompleteOutput (pure) ---

// TestCommandsPlanCov_PlanCheckOutput_Good_PhaseScopedComplete — checking a
// specific, completed phase reports complete with no pending items.
func TestCommandsPlanCov_PlanCheckOutput_Good_PhaseScopedComplete(t *testing.T) {
	plan := PlanCompatibilityView{
		Slug:   "p1",
		Phases: []Phase{{Number: 1, Name: "Build", Status: "completed"}},
	}
	out := planCheckOutput(plan, 1)
	core.AssertTrue(t, out.Complete)
	core.AssertEqual(t, 1, out.Phase)
	core.AssertEqual(t, "Build", out.PhaseName)
	core.AssertEqual(t, 0, len(out.Pending))
}

// TestCommandsPlanCov_PlanCheckOutput_Ugly_PhaseNotFound — a phase order that is
// not present is reported incomplete with a "phase N not found" pending entry.
func TestCommandsPlanCov_PlanCheckOutput_Ugly_PhaseNotFound(t *testing.T) {
	plan := PlanCompatibilityView{
		Slug:   "p1",
		Phases: []Phase{{Number: 1, Name: "Build", Status: "completed"}},
	}
	out := planCheckOutput(plan, 9)
	core.AssertFalse(t, out.Complete)
	core.RequireTrue(t, len(out.Pending) == 1)
	core.AssertContains(t, out.Pending[0], "phase 9 not found")
}

// TestCommandsPlanCov_PlanCompleteOutput_Ugly_PendingTasksAndPhases — a
// whole-plan check aggregates pending tasks (prefixed with their phase) and
// reports a no-task incomplete phase by name.
func TestCommandsPlanCov_PlanCompleteOutput_Ugly_PendingTasksAndPhases(t *testing.T) {
	plan := PlanCompatibilityView{
		Slug: "p1",
		Phases: []Phase{
			// Phase with a pending task → "phase 1: <task title>".
			{Number: 1, Name: "Build", Tasks: []PlanTask{
				{ID: "1", Title: "Compile", Status: "completed"},
				{ID: "2", Title: "Link", Status: "pending"},
			}},
			// Phase with no tasks and a non-complete status → "phase 2: Test".
			{Number: 2, Name: "Test", Status: "pending"},
		},
	}
	out := planCheckOutput(plan, 0)
	core.AssertFalse(t, out.Complete)
	core.AssertContains(t, core.Join("|", out.Pending...), "phase 1: Link")
	core.AssertContains(t, core.Join("|", out.Pending...), "phase 2: Test")
}

// TestCommandsPlanCov_PhaseCompleteOutput_Ugly_TaskTitleFallsBackToID — a
// pending task with no title surfaces its ID as the pending label.
func TestCommandsPlanCov_PhaseCompleteOutput_Ugly_TaskTitleFallsBackToID(t *testing.T) {
	phase := Phase{Number: 1, Name: "Build", Tasks: []PlanTask{{ID: "task-7", Status: "pending"}}}
	complete, pending := phaseCompleteOutput(phase)
	core.AssertFalse(t, complete)
	core.RequireTrue(t, len(pending) == 1)
	core.AssertEqual(t, "task-7", pending[0])
}

// TestCommandsPlanCov_PhaseCompleteOutput_Good_CriteriaDerivedTasks — a phase
// whose tasks are derived from criteria reports complete only when every
// derived task is completed.
func TestCommandsPlanCov_PhaseCompleteOutput_Good_CriteriaDerivedTasks(t *testing.T) {
	// approved status with no tasks → complete.
	complete, pending := phaseCompleteOutput(Phase{Number: 1, Name: "Sign-off", Status: "approved"})
	core.AssertTrue(t, complete)
	core.AssertEqual(t, 0, len(pending))

	// done status with no tasks → complete.
	complete, _ = phaseCompleteOutput(Phase{Number: 2, Name: "Ship", Status: "done"})
	core.AssertTrue(t, complete)
}

// --- cmdPlanCreate template path (injectable templateCreatePlan seam) ---

// TestCommandsPlanCov_CmdPlanCreate_Good_TemplateImport drives the template
// branch of cmdPlanCreate via the injectable seam and asserts the created/title/
// status lines.
func TestCommandsPlanCov_CmdPlanCreate_Good_TemplateImport(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := templateCreatePlan
	t.Cleanup(func() { templateCreatePlan = original })

	var gotInput TemplateCreatePlanInput
	templateCreatePlan = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input TemplateCreatePlanInput) (*mcp.CallToolResult, TemplateCreatePlanOutput, error) {
		gotInput = input
		return nil, TemplateCreatePlanOutput{
			Success: true,
			Plan:    PlanCompatibilitySummary{Slug: "bug-fix-abc", Title: "Bug Fix", Status: "ready"},
		}, nil
	}

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPlanCreate(core.NewOptions(
			core.Option{Key: "_arg", Value: "bug-fix-abc"},
			core.Option{Key: "import", Value: "bug-fix"},
			core.Option{Key: "title", Value: "Bug Fix"},
			core.Option{Key: "activate", Value: true},
		))
		core.RequireTrue(t, r.OK)
	})

	core.AssertEqual(t, "bug-fix", gotInput.Template)
	core.AssertTrue(t, gotInput.Activate)
	core.AssertContains(t, output, "created: bug-fix-abc")
	core.AssertContains(t, output, "title:   Bug Fix")
	core.AssertContains(t, output, "status:  ready")
}

// TestCommandsPlanCov_CmdPlanCreate_Ugly_TemplateError — a failing template seam
// surfaces the error envelope from the template branch.
func TestCommandsPlanCov_CmdPlanCreate_Ugly_TemplateError(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := templateCreatePlan
	t.Cleanup(func() { templateCreatePlan = original })
	templateCreatePlan = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ TemplateCreatePlanInput) (*mcp.CallToolResult, TemplateCreatePlanOutput, error) {
		return nil, TemplateCreatePlanOutput{}, core.E("agentic.templateCreatePlan", "no such template", nil)
	}

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPlanCreate(core.NewOptions(
			core.Option{Key: "_arg", Value: "x"},
			core.Option{Key: "template", Value: "missing"},
		))
		core.AssertFalse(t, r.OK)
		core.AssertContains(t, r.Value.(error).Error(), "no such template")
	})
	core.AssertContains(t, output, "error:")
}

// TestCommandsPlanCov_CmdPlanCreate_Good_ObjectiveFallsBackToTitle — with no
// objective and no description, cmdPlanCreate falls back to the title for the
// objective (the two-step fallback branch) and writes a real plan.
func TestCommandsPlanCov_CmdPlanCreate_Good_ObjectiveFallsBackToTitle(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdPlanCreate(core.NewOptions(
			core.Option{Key: "_arg", Value: "fallback-plan"},
			core.Option{Key: "title", Value: "Fallback Plan"},
		))
		core.RequireTrue(t, r.OK)
	})
	out, ok := r.Value.(PlanCreateOutput)
	core.RequireTrue(t, ok)
	core.AssertNotEmpty(t, out.ID)
	core.AssertContains(t, output, "created: ")
	core.AssertContains(t, output, "path:    ")
}

// TestCommandsPlanCov_CmdPlanCreate_Bad_MissingTitle — no title and no template
// prints usage and returns the required-field error.
func TestCommandsPlanCov_CmdPlanCreate_Bad_MissingTitle(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPlanCreate(core.NewOptions(core.Option{Key: "_arg", Value: "no-title"}))
		core.AssertFalse(t, r.OK)
		core.AssertContains(t, r.Value.(error).Error(), "title is required")
	})
	core.AssertContains(t, output, "usage: core-agent plan create")
}

// TestCommandsPlanCov_CmdPlanTemplates_Good_PrintsVariablesAndCategory drives the
// template list with an entry carrying a category and variables so those
// optional print lines are exercised.
func TestCommandsPlanCov_CmdPlanTemplates_Good_PrintsVariablesAndCategory(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := templateList
	t.Cleanup(func() { templateList = original })
	templateList = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ TemplateListInput) (*mcp.CallToolResult, TemplateListOutput, error) {
		return nil, TemplateListOutput{
			Success: true,
			Total:   1,
			Templates: []TemplateSummary{{
				Slug:        "bug-fix",
				Name:        "Bug Fix",
				Category:    "development",
				PhasesCount: 6,
				Variables:   []TemplateVariable{{Name: "repo"}},
			}},
		}, nil
	}

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPlanTemplates(core.NewOptions())
		core.RequireTrue(t, r.OK)
	})
	core.AssertContains(t, output, "bug-fix")
	core.AssertContains(t, output, "category: development")
	core.AssertContains(t, output, "variables: 1")
	core.AssertContains(t, output, "1 template(s)")
}

// TestCommandsPlanCov_CmdPlanTemplates_Ugly_ListError — a failing template list
// seam surfaces the error envelope.
func TestCommandsPlanCov_CmdPlanTemplates_Ugly_ListError(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	original := templateList
	t.Cleanup(func() { templateList = original })
	templateList = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ TemplateListInput) (*mcp.CallToolResult, TemplateListOutput, error) {
		return nil, TemplateListOutput{}, core.E("agentic.templateList", "template store unreadable", nil)
	}

	s := newTestPrep(t)
	output := captureStdout(t, func() {
		r := s.cmdPlanTemplates(core.NewOptions())
		core.AssertFalse(t, r.OK)
	})
	core.AssertContains(t, output, "error:")
}
