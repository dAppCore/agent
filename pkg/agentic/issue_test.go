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

func TestIssue_HandleIssueRecordCreate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/issues", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "Fix auth", payload["title"])
		require.Equal(t, "bug", payload["type"])

		_, _ = w.Write([]byte(`{"data":{"slug":"fix-auth","title":"Fix auth","type":"bug","status":"open","priority":"high","labels":["auth"]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordCreate(context.Background(), core.NewOptions(
		core.Option{Key: "title", Value: "Fix auth"},
		core.Option{Key: "type", Value: "bug"},
		core.Option{Key: "priority", Value: "high"},
		core.Option{Key: "labels", Value: "auth"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(IssueOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "fix-auth", output.Issue.Slug)
	assert.Equal(t, "open", output.Issue.Status)
	assert.Equal(t, []string{"auth"}, output.Issue.Labels)
}

func TestIssue_HandleIssueRecordGet_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleIssueRecordGet(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
	assert.EqualError(t, result.Value.(error), "issueGet: slug is required")
}

func TestIssue_HandleIssueRecordList_Ugly_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/issues", r.URL.Path)
		require.Equal(t, "open", r.URL.Query().Get("status"))
		_, _ = w.Write([]byte(`{"data":{"issues":[{"id":7,"workspace_id":3,"sprint_id":5,"slug":"fix-auth","title":"Fix auth","labels":["auth","backend"]}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleIssueRecordList(context.Background(), core.NewOptions(
		core.Option{Key: "status", Value: "open"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(IssueListOutput)
	require.True(t, ok)
	require.Len(t, output.Issues, 1)
	assert.Equal(t, 1, output.Count)
	assert.Equal(t, 3, output.Issues[0].WorkspaceID)
	assert.Equal(t, 5, output.Issues[0].SprintID)
	assert.Equal(t, []string{"auth", "backend"}, output.Issues[0].Labels)
}
