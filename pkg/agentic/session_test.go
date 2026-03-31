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

func TestSession_HandleSessionStart_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "codex", payload["agent_type"])
		require.Equal(t, "ax-follow-up", payload["plan_slug"])

		_, _ = w.Write([]byte(`{"data":{"id":1,"session_id":"ses_abc123","plan_slug":"ax-follow-up","agent_type":"codex","status":"active","context_summary":{"repo":"core/go"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionStart(context.Background(), core.NewOptions(
		core.Option{Key: "agent_type", Value: "codex"},
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "context", Value: `{"repo":"core/go"}`},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	assert.Equal(t, "ses_abc123", output.Session.SessionID)
	assert.Equal(t, "active", output.Session.Status)
	assert.Equal(t, "codex", output.Session.AgentType)
}

func TestSession_HandleSessionStart_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleSessionStart(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
}

func TestSession_HandleSessionStart_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionStart(context.Background(), core.NewOptions(
		core.Option{Key: "agent_type", Value: "codex"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionGet_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions/ses_abc123", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{"data":{"session_id":"ses_abc123","plan":"ax-follow-up","agent_type":"codex","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionGet(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_abc123"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	assert.Equal(t, "ses_abc123", output.Session.SessionID)
	assert.Equal(t, "ax-follow-up", output.Session.Plan)
}

func TestSession_HandleSessionList_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions", r.URL.Path)
		require.Equal(t, "ax-follow-up", r.URL.Query().Get("plan_slug"))
		require.Equal(t, "active", r.URL.Query().Get("status"))
		require.Equal(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":[{"session_id":"ses_1","agent_type":"codex","status":"active"},{"session_id":"ses_2","agent_type":"claude","status":"completed"}],"count":2}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionList(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "status", Value: "active"},
		core.Option{Key: "limit", Value: 5},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionListOutput)
	require.True(t, ok)
	assert.Equal(t, 2, output.Count)
	require.Len(t, output.Sessions, 2)
	assert.Equal(t, "ses_1", output.Sessions[0].SessionID)
}

func TestSession_HandleSessionContinue_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions/ses_abc123/continue", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "codex", payload["agent_type"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses_abc123","agent_type":"codex","status":"active","work_log":[{"type":"checkpoint","message":"continue"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionContinue(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_abc123"},
		core.Option{Key: "agent_type", Value: "codex"},
		core.Option{Key: "work_log", Value: `[{"type":"checkpoint","message":"continue"}]`},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	require.Len(t, output.Session.WorkLog, 1)
	assert.Equal(t, "active", output.Session.Status)
}

func TestSession_HandleSessionEnd_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions/ses_abc123/end", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "completed", payload["status"])
		require.Equal(t, "All green", payload["summary"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses_abc123","agent_type":"codex","status":"completed","summary":"All green","ended_at":"2026-03-31T12:00:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionEnd(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_abc123"},
		core.Option{Key: "status", Value: "completed"},
		core.Option{Key: "summary", Value: "All green"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	assert.Equal(t, "completed", output.Session.Status)
	assert.Equal(t, "All green", output.Session.Summary)
}
