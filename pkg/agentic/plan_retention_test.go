// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanRetention_PlanCleanup_Good_DeletesExpiredArchivedPlans(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

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
	require.NoError(t, err)
	_, err = writePlan(PlansRoot(), recentPlan)
	require.NoError(t, err)
	_, err = writePlan(PlansRoot(), activePlan)
	require.NoError(t, err)

	result := s.planCleanup(core.NewOptions(core.Option{Key: "days", Value: 90}))
	require.True(t, result.OK)

	output, ok := result.Value.(PlanCleanupOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Deleted)
	assert.Equal(t, 1, output.Matched)
	assert.False(t, fs.Exists(core.JoinPath(PlansRoot(), "old-plan-abc123.json")))
	assert.True(t, fs.Exists(core.JoinPath(PlansRoot(), "recent-plan-abc123.json")))
	assert.True(t, fs.Exists(core.JoinPath(PlansRoot(), "active-plan-abc123.json")))
}

func TestPlanRetention_PlanCleanup_Bad_DryRunKeepsFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)

	plan := &Plan{
		ID:         "dry-run-plan-abc123",
		Title:      "Dry Run Plan",
		Status:     "archived",
		Objective:  "Keep me for now",
		ArchivedAt: time.Now().AddDate(0, 0, -100),
	}

	_, err := writePlan(PlansRoot(), plan)
	require.NoError(t, err)

	result := s.planCleanup(core.NewOptions(
		core.Option{Key: "days", Value: 90},
		core.Option{Key: "dry-run", Value: true},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(PlanCleanupOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.True(t, output.DryRun)
	assert.Equal(t, 1, output.Matched)
	assert.Equal(t, 0, output.Deleted)
	assert.True(t, fs.Exists(core.JoinPath(PlansRoot(), "dry-run-plan-abc123.json")))
}

func TestPlanRetention_PlanCleanup_Ugly_DisabledCleanupKeepsFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)

	plan := &Plan{
		ID:         "disabled-plan-abc123",
		Title:      "Disabled Plan",
		Status:     "archived",
		Objective:  "Should remain",
		ArchivedAt: time.Now().AddDate(0, 0, -100),
	}

	_, err := writePlan(PlansRoot(), plan)
	require.NoError(t, err)

	result := s.planCleanup(core.NewOptions(core.Option{Key: "days", Value: 0}))
	require.True(t, result.OK)

	output, ok := result.Value.(PlanCleanupOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.True(t, output.Disabled)
	assert.Equal(t, 0, output.Deleted)
	assert.True(t, fs.Exists(core.JoinPath(PlansRoot(), "disabled-plan-abc123.json")))
}

func TestPlanRetention_PlanArchivedAt_Good_FallsBackToFileModifiedTime(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	path := core.JoinPath(PlansRoot(), "fallback-plan-abc123.json")
	require.True(t, fs.Write(path, `{"id":"fallback-plan-abc123","title":"Fallback","status":"archived","objective":"Fallback"}`).OK)

	stat := fs.Stat(path)
	require.True(t, stat.OK)

	plan := &Plan{
		ID:        "fallback-plan-abc123",
		Title:     "Fallback",
		Status:    "archived",
		Objective: "Fallback",
	}

	archivedAt := planArchivedAt(path, plan)
	assert.False(t, archivedAt.IsZero())

	_, ok := stat.Value.(interface{ ModTime() time.Time })
	assert.True(t, ok)
}
