// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

// --- planPath ---

func TestPlan_PlanPath_Good_BasicFormat(t *testing.T) {
	result := planPath("/tmp/plans", "my-plan-abc123")
	core.AssertEqual(t, "/tmp/plans/my-plan-abc123.json", result)
	core.AssertContains(t, result, "my-plan-abc123.json")
}

func TestPlan_PlanPath_Good_NestedIDStripped(t *testing.T) {
	// SanitisePath strips directory components — prevents path traversal
	result := planPath("/plans", "../../../etc/passwd")
	core.AssertEqual(t, "/plans/passwd.json", result)
	core.AssertNotContains(t, result, "..")
}

func TestPlan_PlanPath_Good_SimpleID(t *testing.T) {
	result := planPath("/data", "test")
	core.AssertEqual(t, "/data/test.json", result)
	core.AssertContains(t, result, "test.json")
}

func TestPlan_PlanPath_Good_SlugWithDashes(t *testing.T) {
	result := planPath("/root", "migrate-core-abc123")
	core.AssertEqual(t, "/root/migrate-core-abc123.json", result)
	core.AssertContains(t, result, "migrate-core-abc123.json")
}

func TestPlan_PlanPath_Bad_DotID(t *testing.T) {
	// "." is sanitised to "invalid" to prevent exploiting the root directory
	result := planPath("/plans", ".")
	core.AssertEqual(t, "/plans/invalid.json", result)
	core.AssertContains(t, result, "invalid.json")
}

func TestPlan_PlanPath_Bad_DoubleDotID(t *testing.T) {
	result := planPath("/plans", "..")
	core.AssertEqual(t, "/plans/invalid.json", result)
	core.AssertContains(t, result, "invalid.json")
}

func TestPlan_PlanPath_Bad_EmptyID(t *testing.T) {
	result := planPath("/plans", "")
	core.AssertEqual(t, "/plans/invalid.json", result)
	core.AssertContains(t, result, "invalid.json")
}

// --- readPlan / writePlan ---

func TestPlan_ReadWrite_Good_BasicRoundtrip(t *testing.T) {
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
	core.RequireNoError(t, err)
	core.AssertEqual(t, core.JoinPath(dir, "basic-plan-abc.json"), path)

	read, err := readPlan(dir, "basic-plan-abc")
	core.RequireNoError(t, err)

	core.AssertEqual(t, plan.ID, read.ID)
	core.AssertEqual(t, plan.Title, read.Title)
	core.AssertEqual(t, plan.Status, read.Status)
	core.AssertEqual(t, plan.Repo, read.Repo)
	core.AssertEqual(t, plan.Org, read.Org)
	core.AssertEqual(t, plan.Objective, read.Objective)
	core.AssertEqual(t, plan.Agent, read.Agent)
}

func TestPlan_ReadWrite_Good_WithPhases(t *testing.T) {
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
	core.RequireNoError(t, err)

	read, err := readPlan(dir, "phase-plan-abc")
	core.RequireNoError(t, err)

	core.AssertLen(t, read.Phases, 3)
	core.AssertEqual(t, "Setup", read.Phases[0].Name)
	core.AssertEqual(t, "done", read.Phases[0].Status)
	core.AssertEqual(t, []string{"repo cloned", "deps installed"}, read.Phases[0].Criteria)
	core.AssertEqual(t, 3, read.Phases[0].Tests)
	core.AssertEqual(t, "WIP", read.Phases[1].Notes)
	core.AssertEqual(t, "pending", read.Phases[2].Status)
}

func TestPlan_ReadPlan_Bad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readPlan(dir, "nonexistent-plan")
	core.AssertError(t, err)
}

func TestPlan_ReadPlan_Bad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "bad.json"), `{broken`).OK)

	_, err := readPlan(dir, "bad")
	core.AssertError(t, err)
}

func TestPlan_WritePlan_Good_CreatesNestedDir(t *testing.T) {
	base := t.TempDir()
	nested := core.JoinPath(base, "deep", "nested", "plans")

	plan := &Plan{
		ID:        "deep-plan-xyz",
		Title:     "Deep",
		Status:    "draft",
		Objective: "Test nested dir creation",
	}

	path, err := writePlan(nested, plan)
	core.RequireNoError(t, err)
	core.AssertEqual(t, core.JoinPath(nested, "deep-plan-xyz.json"), path)
	core.AssertTrue(t, fs.IsFile(path))
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
	core.RequireNoError(t, err)

	plan.Title = "Second Title"
	plan.Status = "approved"
	_, err = writePlan(dir, plan)
	core.RequireNoError(t, err)

	read, err := readPlan(dir, "overwrite-plan-abc")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Second Title", read.Title)
	core.AssertEqual(t, "approved", read.Status)
}

func TestPlan_ReadPlan_Ugly_EmptyFileLogic(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "empty.json"), "").OK)

	_, err := readPlan(dir, "empty")
	core.AssertError(t, err)
}

func TestPlan_PhaseValue_Good_CompletionCriteriaAlias(t *testing.T) {
	phase, ok := phaseValue(map[string]any{
		"name":                "Setup",
		"completion_criteria": []any{"repo cloned", "dependencies installed"},
	})

	core.RequireTrue(t, ok)
	core.AssertEqual(t, []string{"repo cloned", "dependencies installed"}, phase.Criteria)
	core.AssertEqual(t, []string{"repo cloned", "dependencies installed"}, phase.CompletionCriteria)

	normalised := normalisePhase(phase, 1)
	core.AssertEqual(t, []string{"repo cloned", "dependencies installed"}, normalised.Criteria)
	core.AssertEqual(t, []string{"repo cloned", "dependencies installed"}, normalised.CompletionCriteria)

	tasks := phaseTaskList(normalised)
	core.AssertLen(t, tasks, 2)
	core.AssertEqual(t, "repo cloned", tasks[0].Title)
	core.AssertEqual(t, "dependencies installed", tasks[1].Title)
}
