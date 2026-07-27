// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestPlanRetention_PlanCleanup_Good_DeletesExpiredArchivedPlans(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	oldPlan := &Plan{
		ID:         "old-plan-abc123",
		Title:      "Old Plan",
		Status:     "archived",
		Objective:  "Delete me",
		ArchivedAt: time.Now().AddDate(0, 0, -100),
	}
	recentPlan := &Plan{
		ID:         "recent-plan-abc123",
		Title:      "Recent Plan",
		Status:     "archived",
		Objective:  "Keep me",
		ArchivedAt: time.Now().AddDate(0, 0, -10),
	}
	activePlan := &Plan{
		ID:        "active-plan-abc123",
		Title:     "Active Plan",
		Status:    "ready",
		Objective: "Keep me too",
	}

	_, err := writePlan(PlansRoot(), oldPlan)
	core.RequireNoError(t, err)
	_, err = writePlan(PlansRoot(), recentPlan)
	core.RequireNoError(t, err)
	_, err = writePlan(PlansRoot(), activePlan)
	core.RequireNoError(t, err)

	result := s.planCleanup(core.NewOptions(core.Option{Key: "days", Value: 90}))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(PlanCleanupOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Deleted)
	core.AssertEqual(t, 1, output.Matched)
	core.AssertFalse(t, fs.Exists(core.JoinPath(PlansRoot(), "old-plan-abc123.json")).OK)
	core.AssertTrue(t, fs.Exists(core.JoinPath(PlansRoot(), "recent-plan-abc123.json")).OK)
	core.AssertTrue(t, fs.Exists(core.JoinPath(PlansRoot(), "active-plan-abc123.json")).OK)
}

func TestPlanRetention_PlanCleanup_Good_ArchivesExpiredCompletedPlans(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	plan := &Plan{
		ID:        "completed-plan-abc123",
		Title:     "Completed Plan",
		Status:    "approved",
		Objective: "Archive me",
		UpdatedAt: time.Now().AddDate(0, 0, -100),
	}

	_, err := writePlan(PlansRoot(), plan)
	core.RequireNoError(t, err)

	result := s.planCleanup(core.NewOptions(core.Option{Key: "days", Value: 90}))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(PlanCleanupOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Archived)
	core.AssertEqual(t, 0, output.Deleted)
	core.AssertEqual(t, 1, output.Matched)

	updated, err := readPlan(PlansRoot(), plan.ID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "archived", updated.Status)
	core.AssertFalse(t, updated.ArchivedAt.IsZero())
	core.AssertTrue(t, fs.Exists(core.JoinPath(PlansRoot(), "completed-plan-abc123.json")).OK)
}

func TestPlanRetention_PlanCleanup_Bad_DryRunKeepsFiles(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	plan := &Plan{
		ID:         "dry-run-plan-abc123",
		Title:      "Dry Run Plan",
		Status:     "archived",
		Objective:  "Keep me for now",
		ArchivedAt: time.Now().AddDate(0, 0, -100),
	}

	_, err := writePlan(PlansRoot(), plan)
	core.RequireNoError(t, err)

	result := s.planCleanup(core.NewOptions(
		core.Option{Key: "days", Value: 90},
		core.Option{Key: "dry-run", Value: true},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(PlanCleanupOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertTrue(t, output.DryRun)
	core.AssertEqual(t, 1, output.Matched)
	core.AssertEqual(t, 0, output.Deleted)
	core.AssertTrue(t, fs.Exists(core.JoinPath(PlansRoot(), "dry-run-plan-abc123.json")).OK)
}

func TestPlanRetention_PlanCleanup_Ugly_DisabledCleanupKeepsFiles(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	plan := &Plan{
		ID:         "disabled-plan-abc123",
		Title:      "Disabled Plan",
		Status:     "archived",
		Objective:  "Should remain",
		ArchivedAt: time.Now().AddDate(0, 0, -100),
	}

	_, err := writePlan(PlansRoot(), plan)
	core.RequireNoError(t, err)

	result := s.planCleanup(core.NewOptions(core.Option{Key: "days", Value: 0}))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(PlanCleanupOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertTrue(t, output.Disabled)
	core.AssertEqual(t, 0, output.Deleted)
	core.AssertTrue(t, fs.Exists(core.JoinPath(PlansRoot(), "disabled-plan-abc123.json")).OK)
}

func TestPlanRetention_PlanArchivedAt_Good_FallsBackToFileModifiedTime(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	path := core.JoinPath(PlansRoot(), "fallback-plan-abc123.json")
	core.RequireTrue(t, fs.Write(path, `{"id":"fallback-plan-abc123","title":"Fallback","status":"archived","objective":"Fallback"}`).OK)

	stat := fs.Stat(path)
	core.RequireTrue(t, stat.OK)

	plan := &Plan{
		ID:        "fallback-plan-abc123",
		Title:     "Fallback",
		Status:    "archived",
		Objective: "Fallback",
	}

	archivedAt := planArchivedAt(path, plan)
	core.AssertFalse(t, archivedAt.IsZero())

	_, ok := stat.Value.(interface{ ModTime() time.Time })
	core.AssertTrue(t, ok)
}

func TestPlanRetention_RunPlanCleanupLoop_Good_DeletesExpiredPlans(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	plan := &Plan{
		ID:         "scheduled-plan-abc123",
		Title:      "Scheduled Plan",
		Status:     "archived",
		Objective:  "Remove me on the next retention pass",
		ArchivedAt: time.Now().AddDate(0, 0, -100),
	}

	_, err := writePlan(PlansRoot(), plan)
	core.RequireNoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		s.runPlanCleanupLoop(ctx, time.Millisecond)
		close(done)
	}()

	requireEventually(t, func() bool {
		return !fs.Exists(core.JoinPath(PlansRoot(), "scheduled-plan-abc123.json")).OK
	}, time.Second, 5*time.Millisecond)

	cancel()
	requireEventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 5*time.Millisecond)
}
