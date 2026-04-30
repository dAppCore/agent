// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestSprint_HandleSprintCreate_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "AX Follow-up", payload["title"])
		core.AssertEqual(t, "Finish RFC parity", payload["goal"])

		_, _ = w.Write([]byte(`{"data":{"slug":"ax-follow-up","title":"AX Follow-up","goal":"Finish RFC parity","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSprintCreate(context.Background(), core.NewOptions(
		core.Option{Key: "title", Value: "AX Follow-up"},
		core.Option{Key: "goal", Value: "Finish RFC parity"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SprintOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "ax-follow-up", output.Sprint.Slug)
	core.AssertEqual(t, "active", output.Sprint.Status)
}

func TestSprint_HandleSprintGet_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleSprintGet(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error), "sprintGet: id or slug is required")
}

func TestSprint_HandleSprintGet_Good_IDAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints/7", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"id":7,"slug":"ax-follow-up","title":"AX Follow-up","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSprintGet(context.Background(), core.NewOptions(
		core.Option{Key: "id", Value: "7"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SprintOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 7, output.Sprint.ID)
	core.AssertEqual(t, "ax-follow-up", output.Sprint.Slug)
}

func TestSprint_HandleSprintList_Ugly_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints", r.URL.Path)
		core.AssertEqual(t, "active", r.URL.Query().Get("status"))
		_, _ = w.Write([]byte(`{"data":{"sprints":[{"id":4,"workspace_id":2,"slug":"ax-follow-up","title":"AX Follow-up","goal":"Finish RFC parity","status":"active"}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSprintList(context.Background(), core.NewOptions(
		core.Option{Key: "status", Value: "active"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SprintListOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, output.Sprints, 1)
	core.AssertEqual(t, 1, output.Count)
	core.AssertEqual(t, 2, output.Sprints[0].WorkspaceID)
	core.AssertEqual(t, "Finish RFC parity", output.Sprints[0].Goal)
}

func TestSprint_SprintStart_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints/ax-follow-up/start", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"data":{"sprint":{"slug":"ax-follow-up","title":"AX Follow-up","status":"active"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.sprintStart(context.Background(), SprintTransitionInput{Slug: "ax-follow-up"})
	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(SprintOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "ax-follow-up", output.Sprint.Slug)
	core.AssertEqual(t, "active", output.Sprint.Status)
}

func TestSprint_SprintComplete_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints/7/complete", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"data":{"sprint":{"id":7,"slug":"ax-follow-up","title":"AX Follow-up","status":"completed"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.sprintComplete(context.Background(), SprintTransitionInput{ID: "7"})
	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(SprintOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 7, output.Sprint.ID)
	core.AssertEqual(t, "completed", output.Sprint.Status)
}
