// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- planPath ---

func TestPlan_PlanPath_Good_BasicFormat(t *testing.T) {
	result := planPath("/tmp/plans", "my-plan-abc123")
	assert.Equal(t, "/tmp/plans/my-plan-abc123.json", result)
}

func TestPlan_PlanPath_Good_NestedIDStripped(t *testing.T) {
	// PathBase strips directory component — prevents path traversal
	result := planPath("/plans", "../../../etc/passwd")
	assert.Equal(t, "/plans/passwd.json", result)
}

func TestPlan_PlanPath_Good_SimpleID(t *testing.T) {
	assert.Equal(t, "/data/test.json", planPath("/data", "test"))
}

func TestPlan_PlanPath_Good_SlugWithDashes(t *testing.T) {
	assert.Equal(t, "/root/migrate-core-abc123.json", planPath("/root", "migrate-core-abc123"))
}

func TestPlan_PlanPath_Bad_DotID(t *testing.T) {
	// "." is sanitised to "invalid" to prevent exploiting the root directory
	result := planPath("/plans", ".")
	assert.Equal(t, "/plans/invalid.json", result)
}

func TestPlan_PlanPath_Bad_DoubleDotID(t *testing.T) {
	result := planPath("/plans", "..")
	assert.Equal(t, "/plans/invalid.json", result)
}

func TestPlan_PlanPath_Bad_EmptyID(t *testing.T) {
	result := planPath("/plans", "")
	assert.Equal(t, "/plans/invalid.json", result)
}

// --- readPlan / writePlan ---

func TestReadWritePlan_Good_BasicRoundtrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	plan := &Plan{
		ID:        "basic-plan-abc",
		Title:     "Basic Plan",
		Status:    "draft",
		Repo:      "go-io",
		Org:       "core",
		Objective: "Verify round-trip works",
		Agent:     "claude:opus",
		CreatedAt: now,
		UpdatedAt: now,
	}

	path, err := writePlan(dir, plan)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "basic-plan-abc.json"), path)

	read, err := readPlan(dir, "basic-plan-abc")
	require.NoError(t, err)

	assert.Equal(t, plan.ID, read.ID)
	assert.Equal(t, plan.Title, read.Title)
	assert.Equal(t, plan.Status, read.Status)
	assert.Equal(t, plan.Repo, read.Repo)
	assert.Equal(t, plan.Org, read.Org)
	assert.Equal(t, plan.Objective, read.Objective)
	assert.Equal(t, plan.Agent, read.Agent)
}

func TestReadWritePlan_Good_WithPhases(t *testing.T) {
	dir := t.TempDir()

	plan := &Plan{
		ID:        "phase-plan-abc",
		Title:     "Phased Work",
		Status:    "in_progress",
		Objective: "Multi-phase plan",
		Phases: []Phase{
			{Number: 1, Name: "Setup", Status: "done", Criteria: []string{"repo cloned", "deps installed"}, Tests: 3},
			{Number: 2, Name: "Implement", Status: "in_progress", Notes: "WIP"},
			{Number: 3, Name: "Verify", Status: "pending"},
		},
	}

	_, err := writePlan(dir, plan)
	require.NoError(t, err)

	read, err := readPlan(dir, "phase-plan-abc")
	require.NoError(t, err)

	require.Len(t, read.Phases, 3)
	assert.Equal(t, "Setup", read.Phases[0].Name)
	assert.Equal(t, "done", read.Phases[0].Status)
	assert.Equal(t, []string{"repo cloned", "deps installed"}, read.Phases[0].Criteria)
	assert.Equal(t, 3, read.Phases[0].Tests)
	assert.Equal(t, "WIP", read.Phases[1].Notes)
	assert.Equal(t, "pending", read.Phases[2].Status)
}

func TestPlan_ReadPlan_Bad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readPlan(dir, "nonexistent-plan")
	assert.Error(t, err)
}

func TestPlan_ReadPlan_Bad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "bad.json"), `{broken`).OK)

	_, err := readPlan(dir, "bad")
	assert.Error(t, err)
}

func TestPlan_WritePlan_Good_CreatesNestedDir(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "deep", "nested", "plans")

	plan := &Plan{
		ID:        "deep-plan-xyz",
		Title:     "Deep",
		Status:    "draft",
		Objective: "Test nested dir creation",
	}

	path, err := writePlan(nested, plan)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(nested, "deep-plan-xyz.json"), path)
	assert.True(t, fs.IsFile(path))
}

func TestPlan_WritePlan_Good_OverwriteExistingLogic(t *testing.T) {
	dir := t.TempDir()

	plan := &Plan{
		ID:        "overwrite-plan-abc",
		Title:     "First Title",
		Status:    "draft",
		Objective: "Initial",
	}
	_, err := writePlan(dir, plan)
	require.NoError(t, err)

	plan.Title = "Second Title"
	plan.Status = "approved"
	_, err = writePlan(dir, plan)
	require.NoError(t, err)

	read, err := readPlan(dir, "overwrite-plan-abc")
	require.NoError(t, err)
	assert.Equal(t, "Second Title", read.Title)
	assert.Equal(t, "approved", read.Status)
}

func TestPlan_ReadPlan_Ugly_EmptyFileLogic(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "empty.json"), "").OK)

	_, err := readPlan(dir, "empty")
	assert.Error(t, err)
}
