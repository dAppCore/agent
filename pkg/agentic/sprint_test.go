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

func TestSprint_HandleSprintCreate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sprints", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "AX Follow-up", payload["title"])
		require.Equal(t, "Finish RFC parity", payload["goal"])

		_, _ = w.Write([]byte(`{"data":{"slug":"ax-follow-up","title":"AX Follow-up","goal":"Finish RFC parity","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSprintCreate(context.Background(), core.NewOptions(
		core.Option{Key: "title", Value: "AX Follow-up"},
		core.Option{Key: "goal", Value: "Finish RFC parity"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SprintOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "ax-follow-up", output.Sprint.Slug)
	assert.Equal(t, "active", output.Sprint.Status)
}

func TestSprint_HandleSprintGet_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleSprintGet(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
	assert.EqualError(t, result.Value.(error), "sprintGet: id or slug is required")
}

func TestSprint_HandleSprintGet_Good_IDAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sprints/7", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"id":7,"slug":"ax-follow-up","title":"AX Follow-up","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSprintGet(context.Background(), core.NewOptions(
		core.Option{Key: "id", Value: "7"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SprintOutput)
	require.True(t, ok)
	assert.Equal(t, 7, output.Sprint.ID)
	assert.Equal(t, "ax-follow-up", output.Sprint.Slug)
}

func TestSprint_HandleSprintList_Ugly_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sprints", r.URL.Path)
		require.Equal(t, "active", r.URL.Query().Get("status"))
		_, _ = w.Write([]byte(`{"data":{"sprints":[{"id":4,"workspace_id":2,"slug":"ax-follow-up","title":"AX Follow-up","goal":"Finish RFC parity","status":"active"}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSprintList(context.Background(), core.NewOptions(
		core.Option{Key: "status", Value: "active"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SprintListOutput)
	require.True(t, ok)
	require.Len(t, output.Sprints, 1)
	assert.Equal(t, 1, output.Count)
	assert.Equal(t, 2, output.Sprints[0].WorkspaceID)
	assert.Equal(t, "Finish RFC parity", output.Sprints[0].Goal)
}
