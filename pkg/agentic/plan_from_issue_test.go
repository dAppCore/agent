// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestPlanFromIssue_PlanFromIssue_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/issues/fix-auth", r.URL.Path)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":{"issue":{"id":17,"slug":"fix-auth","title":"Fix auth middleware","description":"Stop anonymous access to the admin route\n\n## Checklist\n- [ ] Keep CLI output stable","type":"bug","status":"open","priority":"high","labels":["security","backend"],"metadata":{"source":"forge"}}}}`))
	}))
	defer server.Close()

	s := newTestPrep(t)
	s.brainURL = server.URL

	result := s.handlePlanFromIssue(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(PlanFromIssueOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "Fix auth middleware", output.Issue.Title)
	core.AssertEqual(t, "issue-fix-auth", output.Plan.Slug)
	core.AssertEqual(t, "Stop anonymous access to the admin route\n\n## Checklist\n- [ ] Keep CLI output stable", output.Plan.Objective)
	core.AssertNotEmpty(t, output.Path)
	core.AssertTrue(t, fs.Exists(output.Path))

	plan, err := readPlan(PlansRoot(), output.Plan.ID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, output.Plan.Slug, plan.Slug)
	core.AssertEqual(t, output.Issue.Slug, plan.Context["source_issue_slug"])
	core.AssertEqual(t, output.Issue.Status, plan.Context["source_issue_status"])
	core.AssertEqual(t, output.Issue.Metadata, plan.Context["source_issue_metadata"])
	core.AssertLen(t, plan.Phases, 1)
	core.AssertLen(t, plan.Phases[0].Tasks, 1)
	core.AssertEqual(t, "Keep CLI output stable", plan.Phases[0].Tasks[0].Title)
}

func TestPlanFromIssue_PlanFromIssue_Bad_MissingIdentifier(t *testing.T) {
	s := newTestPrep(t)

	result := s.handlePlanFromIssue(context.Background(), core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "issue slug or id is required")
}

func TestPlanFromIssue_PlanFromIssue_Ugly_FallsBackToTitleObjective(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"issue":{"id":22,"slug":"refine-logging","title":"Refine logging"}}}`))
	}))
	defer server.Close()

	s := newTestPrep(t)
	s.brainURL = server.URL

	result := s.handlePlanFromIssue(context.Background(), core.NewOptions(
		core.Option{Key: "_arg", Value: "refine-logging"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(PlanFromIssueOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Refine logging", output.Plan.Objective)
	core.AssertEqual(t, "issue-refine-logging", output.Plan.Slug)
	core.AssertEqual(t, "Refine logging", output.Plan.Title)
}

func TestPlanFromIssue_PlanFromIssue_Good_NoChecklistKeepsTasksEmpty(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"issue":{"id":31,"slug":"investigate-latency","title":"Investigate latency","description":"The dashboard is slow. Please investigate."}}}`))
	}))
	defer server.Close()

	s := newTestPrep(t)
	s.brainURL = server.URL

	result := s.handlePlanFromIssue(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "investigate-latency"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(PlanFromIssueOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, output.Plan.Phases, 1)
	core.AssertEmpty(t, output.Plan.Phases[0].Tasks)
}

func TestPlanFromIssue_CmdPlanFromIssue_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"issue":{"id":5,"slug":"fix-build","title":"Fix build output","description":"Keep CLI output stable"}}}`))
	}))
	defer server.Close()

	s := newTestPrep(t)
	s.brainURL = server.URL

	output := captureStdout(t, func() {
		result := s.cmdPlanFromIssue(core.NewOptions(core.Option{Key: "_arg", Value: "fix-build"}))
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "created:")
	core.AssertContains(t, output, "issue:")
	core.AssertContains(t, output, "path:")
}
