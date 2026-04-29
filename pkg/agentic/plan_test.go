// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func assertCoreIDFormat(t *testing.T, id string) {
	t.Helper()
	parts := core.Split(id, "-")
	core.AssertLen(t, parts, 3)
	if len(parts) != 3 {
		return
	}
	core.AssertEqual(t, "id", parts[0])
	assertRegexp(t, "^[1-9][0-9]*$", parts[1])
	core.AssertLen(t, parts[2], 6)
	assertRegexp(t, "^[0-9a-f]{6}$", parts[2])
}

func TestPlan_PlanPath_Good_Case(t *testing.T) {
	first := planPath("/tmp/plans", "my-plan-abc123")
	second := planPath("/data", "test")
	core.AssertEqual(t, "/tmp/plans/my-plan-abc123.json", first)
	core.AssertEqual(t, "/data/test.json", second)
}

func TestPlan_WritePlan_Good_Case(t *testing.T) {
	dir := t.TempDir()
	plan := &Plan{
		ID:        "test-plan-abc123",
		Title:     "Test Plan",
		Status:    "draft",
		Objective: "Test the plan system",
	}

	path, err := writePlan(dir, plan)
	core.RequireNoError(t, err)
	core.AssertEqual(t, core.JoinPath(dir, "test-plan-abc123.json"), path)

	// Verify file exists
	core.AssertTrue(t, fs.IsFile(path))
}

func TestPlan_WritePlan_Good_CreatesDirectory(t *testing.T) {
	base := t.TempDir()
	dir := core.JoinPath(base, "nested", "plans")

	plan := &Plan{
		ID:        "nested-plan-abc123",
		Title:     "Nested",
		Status:    "draft",
		Objective: "Test nested directory creation",
	}

	path, err := writePlan(dir, plan)
	core.RequireNoError(t, err)
	core.AssertContains(t, path, "nested-plan-abc123.json")
}

func TestPlan_ReadPlan_Good_Case(t *testing.T) {
	dir := t.TempDir()
	original := &Plan{
		ID:        "read-test-abc123",
		Title:     "Read Test",
		Status:    "ready",
		Repo:      "go-io",
		Org:       "core",
		Objective: "Verify plan reading works",
		Phases: []Phase{
			{Number: 1, Name: "Setup", Status: "done", Tasks: []PlanTask{{ID: "1", Title: "Review imports", File: "pkg/agentic/plan.go", Line: 46}}},
			{Number: 2, Name: "Implement", Status: "pending"},
		},
		Notes: "Some notes",
		Agent: "claude:opus",
	}

	_, err := writePlan(dir, original)
	core.RequireNoError(t, err)

	read, err := readPlan(dir, "read-test-abc123")
	core.RequireNoError(t, err)

	core.AssertEqual(t, original.ID, read.ID)
	core.AssertEqual(t, original.Title, read.Title)
	core.AssertEqual(t, original.Status, read.Status)
	core.AssertEqual(t, original.Repo, read.Repo)
	core.AssertEqual(t, original.Org, read.Org)
	core.AssertEqual(t, original.Objective, read.Objective)
	core.AssertLen(t, read.Phases, 2)
	core.AssertEqual(t, "Setup", read.Phases[0].Name)
	core.AssertEqual(t, "done", read.Phases[0].Status)
	core.AssertLen(t, read.Phases[0].Tasks, 1)
	core.AssertEqual(t, "Review imports", read.Phases[0].Tasks[0].Title)
	core.AssertEqual(t, "pkg/agentic/plan.go", read.Phases[0].Tasks[0].File)
	core.AssertEqual(t, 46, read.Phases[0].Tasks[0].Line)
	core.AssertEqual(t, "Implement", read.Phases[1].Name)
	core.AssertEqual(t, "pending", read.Phases[1].Status)
	core.AssertEqual(t, "Some notes", read.Notes)
	core.AssertEqual(t, "claude:opus", read.Agent)
}

func TestPlan_ReadPlan_Bad_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := readPlan(dir, "nonexistent-plan")
	core.AssertError(t, err)
}

func TestPlan_ReadPlan_Bad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "bad-json.json"), "{broken").OK)

	_, err := readPlan(dir, "bad-json")
	core.AssertError(t, err)
}

func TestPlan_WriteRead_Good_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	plan := &Plan{
		ID:        "roundtrip-abc123",
		Title:     "Roundtrip Test",
		Status:    "in_progress",
		Repo:      "agent",
		Org:       "core",
		Objective: "Ensure write-read roundtrip works",
		Phases: []Phase{
			{Number: 1, Name: "Phase One", Status: "done", Criteria: []string{"tests pass", "coverage > 80%"}, Tests: 5, Tasks: []PlanTask{{ID: "1", Title: "tests pass", File: "pkg/agentic/plan_test.go", Line: 100}}},
			{Number: 2, Name: "Phase Two", Status: "in_progress", Notes: "Working on it"},
			{Number: 3, Name: "Phase Three", Status: "pending"},
		},
		Notes: "Important plan",
		Agent: "gemini",
	}

	_, err := writePlan(dir, plan)
	core.RequireNoError(t, err)

	read, err := readPlan(dir, "roundtrip-abc123")
	core.RequireNoError(t, err)

	core.AssertEqual(t, plan.Title, read.Title)
	core.AssertEqual(t, plan.Status, read.Status)
	core.AssertLen(t, read.Phases, 3)
	core.AssertEqual(t, []string{"tests pass", "coverage > 80%"}, read.Phases[0].Criteria)
	core.AssertEqual(t, 5, read.Phases[0].Tests)
	core.AssertLen(t, read.Phases[0].Tasks, 1)
	core.AssertEqual(t, "pkg/agentic/plan_test.go", read.Phases[0].Tasks[0].File)
	core.AssertEqual(t, 100, read.Phases[0].Tasks[0].Line)
	core.AssertEqual(t, "Working on it", read.Phases[1].Notes)
}

func TestPlan_ValidPlanStatus_Good_AllValid(t *testing.T) {
	validStatuses := []string{"draft", "ready", "in_progress", "needs_verification", "verified", "approved"}
	for _, s := range validStatuses {
		core.AssertTrue(t, validPlanStatus(s), "expected %q to be valid", s)
	}
}

func TestPlan_ValidPlanStatus_Bad_Invalid(t *testing.T) {
	invalidStatuses := []string{"", "running", "completed", "cancelled", "archived", "DRAFT", "Draft"}
	for _, s := range invalidStatuses {
		core.AssertFalse(t, validPlanStatus(s), "expected %q to be invalid", s)
	}
}

func TestPlan_WritePlan_Good_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()

	plan := &Plan{
		ID:        "overwrite-abc123",
		Title:     "Original",
		Status:    "draft",
		Objective: "Original objective",
	}

	_, err := writePlan(dir, plan)
	core.RequireNoError(t, err)

	plan.Title = "Updated"
	plan.Status = "ready"
	_, err = writePlan(dir, plan)
	core.RequireNoError(t, err)

	read, err := readPlan(dir, "overwrite-abc123")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Updated", read.Title)
	core.AssertEqual(t, "ready", read.Status)
}

func TestPlan_ReadPlan_Ugly_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "empty.json"), "").OK)

	_, err := readPlan(dir, "empty")
	core.AssertError(t, err)
}

func TestPlan_RegisterPlanTools_Good_RegistersAgenticCompatibilityAliases(t *testing.T) {
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(svc)

	server := svc.Server()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	core.RequireNoError(t, err)

	var toolNames []string
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	core.AssertContains(t, toolNames, "agentic_plan_get")
	core.AssertContains(t, toolNames, "agentic_plan_check")
	core.AssertContains(t, toolNames, "agentic_plan_update_status")
	core.AssertContains(t, toolNames, "agentic_plan_archive")
	core.AssertContains(t, toolNames, "agentic_plan_from_issue")
}
