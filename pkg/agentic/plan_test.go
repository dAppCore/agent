// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"path/filepath"
	"strings"
	"testing"

	coreio "forge.lthn.ai/core/go-io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanPath_Good(t *testing.T) {
	assert.Equal(t, "/tmp/plans/my-plan-abc123.json", planPath("/tmp/plans", "my-plan-abc123"))
	assert.Equal(t, "/data/test.json", planPath("/data", "test"))
}

func TestWritePlan_Good(t *testing.T) {
	dir := t.TempDir()
	plan := &Plan{
		ID:        "test-plan-abc123",
		Title:     "Test Plan",
		Status:    "draft",
		Objective: "Test the plan system",
	}

	path, err := writePlan(dir, plan)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "test-plan-abc123.json"), path)

	// Verify file exists
	assert.True(t, coreio.Local.IsFile(path))
}

func TestWritePlan_Good_CreatesDirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "nested", "plans")

	plan := &Plan{
		ID:        "nested-plan-abc123",
		Title:     "Nested",
		Status:    "draft",
		Objective: "Test nested directory creation",
	}

	path, err := writePlan(dir, plan)
	require.NoError(t, err)
	assert.Contains(t, path, "nested-plan-abc123.json")
}

func TestReadPlan_Good(t *testing.T) {
	dir := t.TempDir()
	original := &Plan{
		ID:        "read-test-abc123",
		Title:     "Read Test",
		Status:    "ready",
		Repo:      "go-io",
		Org:       "core",
		Objective: "Verify plan reading works",
		Phases: []Phase{
			{Number: 1, Name: "Setup", Status: "done"},
			{Number: 2, Name: "Implement", Status: "pending"},
		},
		Notes: "Some notes",
		Agent: "claude:opus",
	}

	_, err := writePlan(dir, original)
	require.NoError(t, err)

	read, err := readPlan(dir, "read-test-abc123")
	require.NoError(t, err)

	assert.Equal(t, original.ID, read.ID)
	assert.Equal(t, original.Title, read.Title)
	assert.Equal(t, original.Status, read.Status)
	assert.Equal(t, original.Repo, read.Repo)
	assert.Equal(t, original.Org, read.Org)
	assert.Equal(t, original.Objective, read.Objective)
	assert.Len(t, read.Phases, 2)
	assert.Equal(t, "Setup", read.Phases[0].Name)
	assert.Equal(t, "done", read.Phases[0].Status)
	assert.Equal(t, "Implement", read.Phases[1].Name)
	assert.Equal(t, "pending", read.Phases[1].Status)
	assert.Equal(t, "Some notes", read.Notes)
	assert.Equal(t, "claude:opus", read.Agent)
}

func TestReadPlan_Bad_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := readPlan(dir, "nonexistent-plan")
	assert.Error(t, err)
}

func TestReadPlan_Bad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, coreio.Local.Write(filepath.Join(dir, "bad-json.json"), "{broken"))

	_, err := readPlan(dir, "bad-json")
	assert.Error(t, err)
}

func TestWriteReadPlan_Good_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	plan := &Plan{
		ID:        "roundtrip-abc123",
		Title:     "Roundtrip Test",
		Status:    "in_progress",
		Repo:      "agent",
		Org:       "core",
		Objective: "Ensure write-read roundtrip works",
		Phases: []Phase{
			{Number: 1, Name: "Phase One", Status: "done", Criteria: []string{"tests pass", "coverage > 80%"}, Tests: 5},
			{Number: 2, Name: "Phase Two", Status: "in_progress", Notes: "Working on it"},
			{Number: 3, Name: "Phase Three", Status: "pending"},
		},
		Notes: "Important plan",
		Agent: "gemini",
	}

	_, err := writePlan(dir, plan)
	require.NoError(t, err)

	read, err := readPlan(dir, "roundtrip-abc123")
	require.NoError(t, err)

	assert.Equal(t, plan.Title, read.Title)
	assert.Equal(t, plan.Status, read.Status)
	assert.Len(t, read.Phases, 3)
	assert.Equal(t, []string{"tests pass", "coverage > 80%"}, read.Phases[0].Criteria)
	assert.Equal(t, 5, read.Phases[0].Tests)
	assert.Equal(t, "Working on it", read.Phases[1].Notes)
}

func TestGeneratePlanID_Good_Slugifies(t *testing.T) {
	id := generatePlanID("Add Unit Tests for Agentic")
	assert.True(t, strings.HasPrefix(id, "add-unit-tests-for-agentic"), "got: %s", id)
	// Should have random suffix
	parts := strings.Split(id, "-")
	assert.True(t, len(parts) >= 5, "expected slug with random suffix, got: %s", id)
}

func TestGeneratePlanID_Good_TruncatesLong(t *testing.T) {
	id := generatePlanID("This is a very long title that should be truncated to a reasonable length for file naming purposes")
	// Slug part (before random suffix) should be <= 30 chars
	lastDash := strings.LastIndex(id, "-")
	slug := id[:lastDash]
	assert.True(t, len(slug) <= 36, "slug too long: %s (%d chars)", slug, len(slug))
}

func TestGeneratePlanID_Good_HandlesSpecialChars(t *testing.T) {
	id := generatePlanID("Fix bug #123: auth & session!")
	assert.True(t, strings.Contains(id, "fix-bug"), "got: %s", id)
	assert.NotContains(t, id, "#")
	assert.NotContains(t, id, "!")
	assert.NotContains(t, id, "&")
}

func TestGeneratePlanID_Good_Unique(t *testing.T) {
	id1 := generatePlanID("Same Title")
	id2 := generatePlanID("Same Title")
	assert.NotEqual(t, id1, id2, "IDs should differ due to random suffix")
}

func TestValidPlanStatus_Good_AllValid(t *testing.T) {
	validStatuses := []string{"draft", "ready", "in_progress", "needs_verification", "verified", "approved"}
	for _, s := range validStatuses {
		assert.True(t, validPlanStatus(s), "expected %q to be valid", s)
	}
}

func TestValidPlanStatus_Bad_Invalid(t *testing.T) {
	invalidStatuses := []string{"", "running", "completed", "cancelled", "archived", "DRAFT", "Draft"}
	for _, s := range invalidStatuses {
		assert.False(t, validPlanStatus(s), "expected %q to be invalid", s)
	}
}

func TestWritePlan_Good_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()

	plan := &Plan{
		ID:        "overwrite-abc123",
		Title:     "Original",
		Status:    "draft",
		Objective: "Original objective",
	}

	_, err := writePlan(dir, plan)
	require.NoError(t, err)

	plan.Title = "Updated"
	plan.Status = "ready"
	_, err = writePlan(dir, plan)
	require.NoError(t, err)

	read, err := readPlan(dir, "overwrite-abc123")
	require.NoError(t, err)
	assert.Equal(t, "Updated", read.Title)
	assert.Equal(t, "ready", read.Status)
}

func TestReadPlan_Ugly_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, coreio.Local.Write(filepath.Join(dir, "empty.json"), ""))

	_, err := readPlan(dir, "empty")
	assert.Error(t, err)
}
