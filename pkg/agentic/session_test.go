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

func TestSession_HandleSessionGet_Good_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"session":{"session_id":"ses_nested","plan":"ax-follow-up","agent_type":"codex","status":"active"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionGet(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_nested"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	assert.Equal(t, "ses_nested", output.Session.SessionID)
	assert.Equal(t, "active", output.Session.Status)
}

func TestSession_HandleSessionList_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions", r.URL.Path)
		require.Equal(t, "ax-follow-up", r.URL.Query().Get("plan_slug"))
		require.Equal(t, "codex", r.URL.Query().Get("agent_type"))
		require.Equal(t, "active", r.URL.Query().Get("status"))
		require.Equal(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":[{"session_id":"ses_1","agent_type":"codex","status":"active"},{"session_id":"ses_2","agent_type":"claude","status":"completed"}],"count":2}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionList(context.Background(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "agent_type", Value: "codex"},
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

func TestSession_HandleSessionList_Good_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"sessions":[{"session_id":"ses_1","agent_type":"codex","status":"active"}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionList(context.Background(), core.NewOptions())
	require.True(t, result.OK)

	output, ok := result.Value.(SessionListOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Count)
	require.Len(t, output.Sessions, 1)
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

func TestSession_HandleSessionLog_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writeSessionCache(&Session{
		SessionID: "ses_log",
		AgentType: "codex",
		Status:    "active",
	}))

	result := subsystem.handleSessionLog(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_log"},
		core.Option{Key: "message", Value: "Checked build"},
		core.Option{Key: "type", Value: "checkpoint"},
		core.Option{Key: "data", Value: `{"repo":"core/go"}`},
	))
	require.True(t, result.OK)

	session, err := readSessionCache("ses_log")
	require.NoError(t, err)
	require.Len(t, session.WorkLog, 1)
	assert.Equal(t, "Checked build", session.WorkLog[0]["message"])
	assert.Equal(t, "checkpoint", session.WorkLog[0]["type"])
}

func TestSession_HandleSessionLog_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionLog(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_log"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionLog_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionLog(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_missing"},
		core.Option{Key: "message", Value: "Checked build"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionArtifact_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writeSessionCache(&Session{
		SessionID: "ses_artifact",
		AgentType: "codex",
		Status:    "active",
	}))

	result := subsystem.handleSessionArtifact(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_artifact"},
		core.Option{Key: "path", Value: "pkg/agentic/session.go"},
		core.Option{Key: "action", Value: "modified"},
		core.Option{Key: "metadata", Value: `{"insertions":12}`},
	))
	require.True(t, result.OK)

	session, err := readSessionCache("ses_artifact")
	require.NoError(t, err)
	require.Len(t, session.Artifacts, 1)
	assert.Equal(t, "pkg/agentic/session.go", session.Artifacts[0]["path"])
	assert.Equal(t, "modified", session.Artifacts[0]["action"])
}

func TestSession_HandleSessionArtifact_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionArtifact(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_artifact"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionArtifact_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionArtifact(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_artifact"},
		core.Option{Key: "path", Value: "pkg/agentic/session.go"},
		core.Option{Key: "action", Value: "modified"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionHandoff_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writeSessionCache(&Session{
		SessionID: "ses_handoff",
		AgentType: "codex",
		Status:    "active",
		WorkLog: []map[string]any{
			{"message": "Checked build", "type": "checkpoint", "timestamp": "2026-03-31T10:00:00Z"},
		},
	}))

	result := subsystem.handleSessionHandoff(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_handoff"},
		core.Option{Key: "summary", Value: "Ready for review"},
		core.Option{Key: "next_steps", Value: `["Run verify"]`},
		core.Option{Key: "blockers", Value: `["Need CI"]`},
	))
	require.True(t, result.OK)

	session, err := readSessionCache("ses_handoff")
	require.NoError(t, err)
	assert.Equal(t, "paused", session.Status)
	assert.Equal(t, "Ready for review", session.Handoff["summary"])
}

func TestSession_HandleSessionHandoff_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionHandoff(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_handoff"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionHandoff_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionHandoff(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_handoff"},
		core.Option{Key: "summary", Value: "Ready for review"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionResume_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writeSessionCache(&Session{
		SessionID: "ses_resume",
		AgentType: "codex",
		Status:    "paused",
		Handoff: map[string]any{
			"summary": "Ready for review",
		},
		WorkLog: []map[string]any{
			{"message": "Checked build", "type": "checkpoint", "timestamp": "2026-03-31T10:00:00Z"},
		},
		Artifacts: []map[string]any{
			{"path": "pkg/agentic/session.go", "action": "modified"},
		},
	}))

	result := subsystem.handleSessionResume(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_resume"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionResumeOutput)
	require.True(t, ok)
	assert.Equal(t, "active", output.Session.Status)
	assert.Equal(t, "Ready for review", output.HandoffContext["summary"])
	require.Len(t, output.RecentActions, 1)
}

func TestSession_HandleSessionResume_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionResume(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
}

func TestSession_HandleSessionResume_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionResume(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_resume"},
	))
	assert.False(t, result.OK)
}

func TestSession_HandleSessionReplay_Good(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	require.NoError(t, writeSessionCache(&Session{
		SessionID: "ses_replay",
		AgentType: "codex",
		Status:    "completed",
		PlanSlug:  "ax-follow-up",
		WorkLog: []map[string]any{
			{"message": "Checked build", "type": "checkpoint", "timestamp": "2026-03-31T10:00:00Z"},
			{"message": "Chose pattern", "type": "decision", "timestamp": "2026-03-31T10:10:00Z"},
			{"message": "CI failed", "type": "error", "timestamp": "2026-03-31T10:15:00Z"},
		},
		Artifacts: []map[string]any{
			{"path": "pkg/agentic/session.go", "action": "modified"},
		},
		Handoff: map[string]any{
			"summary": "Ready for review",
		},
		Summary: "Completed work",
	}))

	result := subsystem.handleSessionReplay(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_replay"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionReplayOutput)
	require.True(t, ok)
	assert.Equal(t, "ses_replay", output.ReplayContext["session_id"])
	assert.Equal(t, 3, output.ReplayContext["total_actions"])
	require.Len(t, output.ReplayContext["work_log"].([]map[string]any), 3)
	require.Len(t, output.ReplayContext["checkpoints"].([]map[string]any), 1)
	require.Len(t, output.ReplayContext["work_log_by_type"].(map[string]any)["error"].([]map[string]any), 1)
}

func TestSession_HandleSessionReplay_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionReplay(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
}

func TestSession_HandleSessionReplay_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionReplay(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_replay"},
	))
	assert.False(t, result.OK)
}
