// SPDX-License-Identifier: EUPL-1.2

// Extra coverage for the plan command surface in commands_plan.go. Plans are
// local-file-backed (PlansRoot under the test workspace), so a plan created
// in-process can be driven through the show / status / update / list /
// archive / delete wrappers and the cmdPlan dispatcher with no network. Each
// test seeds a plan, reads its slug, then exercises the success-print and
// guard branches the happy-path suite skipped.

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// seedPlan creates a plan in the test workspace and returns the subsystem +
// the created plan's slug + id.
func seedPlan(t *testing.T) (*PrepSubsystem, string, string) {
	t.Helper()
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	s := newTestPrep(t)
	_, out, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:       "Coverage Plan",
		Objective:   "Exercise the plan command wrappers",
		Description: "seed plan",
		Repo:        "go-io",
		Phases: []Phase{
			{Number: 1, Name: "Setup", Tasks: []PlanTask{{ID: "1", Title: "Review", Status: "completed"}}},
		},
	})
	core.RequireNoError(t, err)
	plan, err := readPlan(PlansRoot(), out.ID)
	core.RequireNoError(t, err)
	return s, plan.Slug, out.ID
}

// --- cmdPlanShow -----------------------------------------------------

func TestCommandsPlan_CmdPlanShow_Good_PrintsPlan(t *testing.T) {
	s, slug, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanShow(core.NewOptions(core.Option{Key: "_arg", Value: slug}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "slug:        "+slug)
	core.AssertContains(t, out, "description: seed plan")
}

func TestCommandsPlan_CmdPlanShow_Bad_NotFound(t *testing.T) {
	s, _, _ := seedPlan(t)
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdPlanShow(core.NewOptions(core.Option{Key: "_arg", Value: "no-such-plan"}))
	})
	core.AssertFalse(t, r.OK)
}

// --- cmdPlanStatus ---------------------------------------------------

func TestCommandsPlan_CmdPlanStatus_Good_Read(t *testing.T) {
	s, slug, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanStatus(core.NewOptions(core.Option{Key: "_arg", Value: slug}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "slug:   "+slug)
	core.AssertContains(t, out, "status:")
}

func TestCommandsPlan_CmdPlanStatus_Good_Set(t *testing.T) {
	s, slug, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanStatus(core.NewOptions(
			core.Option{Key: "_arg", Value: slug},
			core.Option{Key: "set", Value: "ready"},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "status: ready")
}

func TestCommandsPlan_CmdPlanStatus_Bad_MissingSlug(t *testing.T) {
	s, _, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPlanStatus(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent plan status")
}

// --- cmdPlanUpdate (via the cmd wrapper) -----------------------------

func TestCommandsPlan_CmdPlanUpdate_Good_PrintsUpdated(t *testing.T) {
	s, slug, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanUpdate(core.NewOptions(
			core.Option{Key: "slug", Value: slug},
			core.Option{Key: "status", Value: "ready"},
			core.Option{Key: "agent", Value: "codex"},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "status: ready")
	core.AssertContains(t, out, "agent:  codex")
}

func TestCommandsPlan_CmdPlanUpdate_Bad_NoChanges(t *testing.T) {
	s, slug, _ := seedPlan(t)
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdPlanUpdate(core.NewOptions(core.Option{Key: "slug", Value: slug}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "at least one update field is required")
}

// --- cmdPlanList (populated) -----------------------------------------

func TestCommandsPlan_CmdPlanList_Good_Populated(t *testing.T) {
	s, slug, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPlanList(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, slug)
}

// --- cmdPlanArchive --------------------------------------------------

func TestCommandsPlan_CmdPlanArchive_Good_PrintsArchived(t *testing.T) {
	s, slug, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanArchive(core.NewOptions(core.Option{Key: "_arg", Value: slug}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "archived:")
}

func TestCommandsPlan_CmdPlanArchive_Bad_MissingSlug(t *testing.T) {
	s, _, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPlanArchive(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent plan archive")
}

// --- cmdPlanDelete ---------------------------------------------------

func TestCommandsPlan_CmdPlanDelete_Good_PrintsDeleted(t *testing.T) {
	s, _, id := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdPlanDelete(core.NewOptions(core.Option{Key: "_arg", Value: id}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "deleted:")
}

func TestCommandsPlan_CmdPlanDelete_Bad_MissingID(t *testing.T) {
	s, _, _ := seedPlan(t)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPlanDelete(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent plan delete")
}

// --- cmdPlan dispatcher: remaining routes ----------------------------

// TestCommandsPlan_CmdPlan_RoutesShowAndArchiveAndDelete — the dispatcher
// reaches the show / archive / delete sub-handlers via --action.
func TestCommandsPlan_CmdPlan_RoutesShowAndArchiveAndDelete(t *testing.T) {
	s, slug, id := seedPlan(t)

	captureStdout(t, func() {
		core.AssertTrue(t, s.cmdPlan(core.NewOptions(
			core.Option{Key: "action", Value: "show"},
			core.Option{Key: "_arg", Value: slug},
		)).OK)
		core.AssertTrue(t, s.cmdPlan(core.NewOptions(
			core.Option{Key: "action", Value: "templates"},
		)).OK)
		core.AssertTrue(t, s.cmdPlan(core.NewOptions(
			core.Option{Key: "action", Value: "archive"},
			core.Option{Key: "_arg", Value: slug},
		)).OK)
		core.AssertTrue(t, s.cmdPlan(core.NewOptions(
			core.Option{Key: "action", Value: "delete"},
			core.Option{Key: "_arg", Value: id},
		)).OK)
	})
}

// TestCommandsPlan_CmdPlanFromIssue_Bad_MissingIdentifier — from-issue with
// no slug/id prints usage and returns non-OK.
func TestCommandsPlan_CmdPlanFromIssue_Bad_MissingIdentifier(t *testing.T) {
	s := newTestPrep(t)
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdPlanFromIssue(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent plan from-issue")
}
