// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanFromIssue_PlanFromIssue_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/issues/fix-auth", r.URL.Path)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":{"issue":{"id":17,"slug":"fix-auth","title":"Fix auth middleware","description":"Stop anonymous access to the admin route\n\n## Checklist\n- [ ] Keep CLI output stable","type":"bug","status":"open","priority":"high","labels":["security","backend"],"metadata":{"source":"forge"}}}}`))
	}))
	defer server.Close()

	s := newTestPrep(t)
	s.brainURL = server.URL

	result := s.handlePlanFromIssue(context.Background(), core.NewOptions(
		core.Option{Key: "slug", Value: "fix-auth"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(PlanFromIssueOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "Fix auth middleware", output.Issue.Title)
	assert.Equal(t, "issue-fix-auth", output.Plan.Slug)
	assert.Equal(t, "Stop anonymous access to the admin route\n\n## Checklist\n- [ ] Keep CLI output stable", output.Plan.Objective)
	assert.NotEmpty(t, output.Path)
	assert.True(t, fs.Exists(output.Path))

	plan, err := readPlan(PlansRoot(), output.Plan.ID)
	require.NoError(t, err)
	assert.Equal(t, output.Plan.Slug, plan.Slug)
	assert.Equal(t, output.Issue.Slug, plan.Context["source_issue_slug"])
	require.Len(t, plan.Phases, 1)
	require.Len(t, plan.Phases[0].Tasks, 1)
	assert.Equal(t, "Keep CLI output stable", plan.Phases[0].Tasks[0].Title)
}

func TestPlanFromIssue_PlanFromIssue_Bad_MissingIdentifier(t *testing.T) {
	s := newTestPrep(t)

	result := s.handlePlanFromIssue(context.Background(), core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "issue slug or id is required")
}

func TestPlanFromIssue_PlanFromIssue_Ugly_FallsBackToTitleObjective(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)
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
	require.True(t, result.OK)

	output, ok := result.Value.(PlanFromIssueOutput)
	require.True(t, ok)
	assert.Equal(t, "Refine logging", output.Plan.Objective)
	assert.Equal(t, "issue-refine-logging", output.Plan.Slug)
	assert.Equal(t, "Refine logging", output.Plan.Title)
}

func TestPlanFromIssue_PlanFromIssue_Good_NoChecklistKeepsTasksEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)
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
	require.True(t, result.OK)

	output, ok := result.Value.(PlanFromIssueOutput)
	require.True(t, ok)
	require.Len(t, output.Plan.Phases, 1)
	assert.Empty(t, output.Plan.Phases[0].Tasks)
}

func TestPlanFromIssue_CmdPlanFromIssue_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"issue":{"id":5,"slug":"fix-build","title":"Fix build output","description":"Keep CLI output stable"}}}`))
	}))
	defer server.Close()

	s := newTestPrep(t)
	s.brainURL = server.URL

	output := captureStdout(t, func() {
		result := s.cmdPlanFromIssue(core.NewOptions(core.Option{Key: "_arg", Value: "fix-build"}))
		assert.True(t, result.OK)
	})

	assert.Contains(t, output, "created:")
	assert.Contains(t, output, "issue:")
	assert.Contains(t, output, "path:")
}
