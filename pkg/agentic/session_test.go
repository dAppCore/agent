// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestSession_HandleSessionStart_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "opus", payload["agent_type"])
		core.AssertEqual(t, "ax-follow-up", payload["plan_slug"])

		_, _ = w.Write([]byte(`{"data":{"id":1,"session_id":"ses_abc123","plan_slug":"ax-follow-up","agent_type":"opus","status":"active","context_summary":{"repo":"core/go"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionStart(context.Background(), core.NewOptions(
		core.Option{Key: "agent_type", Value: "opus"},
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "context", Value: `{"repo":"core/go"}`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses_abc123", output.Session.SessionID)
	core.AssertEqual(t, "active", output.Session.Status)
	core.AssertEqual(t, "opus", output.Session.AgentType)
}

func TestSession_HandleSessionStart_Good_CanonicalAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "opus", payload["agent_type"])
		core.AssertEqual(t, "ax-follow-up", payload["plan_slug"])

		_, _ = w.Write([]byte(`{"data":{"id":1,"session_id":"ses_abc123","plan_slug":"ax-follow-up","agent_type":"opus","status":"active","context_summary":{"repo":"core/go"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionStart(context.Background(), core.NewOptions(
		core.Option{Key: "agent_type", Value: "claude:opus"},
		core.Option{Key: "plan_slug", Value: "ax-follow-up"},
		core.Option{Key: "context", Value: `{"repo":"core/go"}`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses_abc123", output.Session.SessionID)
	core.AssertEqual(t, "active", output.Session.Status)
	core.AssertEqual(t, "opus", output.Session.AgentType)
}

func TestSession_HandleSessionStart_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleSessionStart(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionStart_Bad_InvalidAgentType(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.handleSessionStart(context.Background(), core.NewOptions(
		core.Option{Key: "agent_type", Value: "codex"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Value.(error).Error(), "claude:opus")
}

func TestSession_HandleSessionStart_Ugly_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionStart(context.Background(), core.NewOptions(
		core.Option{Key: "agent_type", Value: "codex"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionGet_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions/ses_abc123", r.URL.Path)
		core.AssertEqual(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{"data":{"session_id":"ses_abc123","plan":"ax-follow-up","agent_type":"codex","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionGet(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_abc123"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses_abc123", output.Session.SessionID)
	core.AssertEqual(t, "ax-follow-up", output.Session.Plan)
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
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses_nested", output.Session.SessionID)
	core.AssertEqual(t, "active", output.Session.Status)
}

func TestSession_HandleSessionList_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions", r.URL.Path)
		core.AssertEqual(t, "ax-follow-up", r.URL.Query().Get("plan_slug"))
		core.AssertEqual(t, "codex", r.URL.Query().Get("agent_type"))
		core.AssertEqual(t, "active", r.URL.Query().Get("status"))
		core.AssertEqual(t, "5", r.URL.Query().Get("limit"))
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
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 2, output.Count)
	core.AssertLen(t, output.Sessions, 2)
	core.AssertEqual(t, "ses_1", output.Sessions[0].SessionID)
}

func TestSession_HandleSessionList_Good_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"sessions":[{"session_id":"ses_1","agent_type":"codex","status":"active"}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionList(context.Background(), core.NewOptions())
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, output.Count)
	core.AssertLen(t, output.Sessions, 1)
	core.AssertEqual(t, "ses_1", output.Sessions[0].SessionID)
}

func TestSession_HandleSessionContinue_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions/ses_abc123/continue", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "codex", payload["agent_type"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses_abc123","agent_type":"codex","status":"active","work_log":[{"type":"checkpoint","message":"continue"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionContinue(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_abc123"},
		core.Option{Key: "agent_type", Value: "codex"},
		core.Option{Key: "work_log", Value: `[{"type":"checkpoint","message":"continue"}]`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, output.Session.WorkLog, 1)
	core.AssertEqual(t, "active", output.Session.Status)
}

func TestSession_HandleSessionEnd_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions/ses_abc123/end", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "completed", payload["status"])
		core.AssertEqual(t, "All green", payload["summary"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses_abc123","agent_type":"codex","status":"completed","summary":"All green","ended_at":"2026-03-31T12:00:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionEnd(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_abc123"},
		core.Option{Key: "status", Value: "completed"},
		core.Option{Key: "summary", Value: "All green"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", output.Session.Status)
	core.AssertEqual(t, "All green", output.Session.Summary)
}

func TestSession_HandleSessionEnd_Good_HandoffNotes(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/v1/sessions/ses_handoff/end":
			core.AssertEqual(t, http.MethodPost, r.Method)

			bodyResult := core.ReadAll(r.Body)
			core.RequireTrue(t, bodyResult.OK)

			var payload map[string]any
			parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
			core.RequireTrue(t, parseResult.OK)
			core.AssertEqual(t, "handed_off", payload["status"])
			core.AssertEqual(t, "Ready for review", payload["summary"])

			handoffNotes, ok := payload["handoff_notes"].(map[string]any)
			core.RequireTrue(t, ok)
			core.AssertEqual(t, "Ready for review", handoffNotes["summary"])
			core.AssertEqual(t, []any{"Run the verifier"}, handoffNotes["next_steps"])
			core.AssertEqual(t, []any{"Needs input"}, handoffNotes["blockers"])

			_, _ = w.Write([]byte(`{"data":{"session_id":"ses_handoff","agent_type":"codex","status":"handed_off","summary":"Ready for review","handoff_notes":{"summary":"Ready for review","next_steps":["Run the verifier"],"blockers":["Needs input"]}}}`))
		case "/v1/brain/remember":
			core.AssertEqual(t, http.MethodPost, r.Method)
			core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

			bodyResult := core.ReadAll(r.Body)
			core.RequireTrue(t, bodyResult.OK)

			var payload map[string]any
			parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
			core.RequireTrue(t, parseResult.OK)
			core.AssertEqual(t, "observation", payload["type"])
			core.AssertEqual(t, "codex", payload["agent_id"])

			content, _ := payload["content"].(string)
			core.AssertContains(t, content, "Session handoff: ses_handoff")
			core.AssertContains(t, content, "Ready for review")
			core.AssertContains(t, content, "Run the verifier")
			core.AssertContains(t, content, "Needs input")

			_, _ = w.Write([]byte(`{"data":{"id":"mem_handoff"}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionEnd(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_handoff"},
		core.Option{Key: "status", Value: "handed_off"},
		core.Option{Key: "summary", Value: "Ready for review"},
		core.Option{Key: "handoff_notes", Value: `{"summary":"Ready for review","next_steps":["Run the verifier"],"blockers":["Needs input"]}`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "handed_off", output.Session.Status)
	core.AssertEqual(t, "Ready for review", output.Session.Summary)
	core.AssertNotNil(t, output.Session.Handoff)
	core.AssertEqual(t, "Ready for review", output.Session.Handoff["summary"])
	core.AssertEqual(t, []any{"Run the verifier"}, output.Session.Handoff["next_steps"])
	core.AssertEqual(t, []any{"Needs input"}, output.Session.Handoff["blockers"])
	core.AssertEqual(t, 2, callCount)
}

func TestSession_HandleSessionEnd_Bad_MissingSessionID(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleSessionEnd(context.Background(), core.NewOptions(
		core.Option{Key: "status", Value: "completed"},
		core.Option{Key: "summary", Value: "All green"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionEnd_Ugly_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSessionEnd(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_abc123"},
		core.Option{Key: "status", Value: "completed"},
		core.Option{Key: "summary", Value: "All green"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionLog_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireNoError(t, writeSessionCache(&Session{
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
	core.RequireTrue(t, result.OK)

	session, err := readSessionCache("ses_log")
	core.RequireNoError(t, err)
	core.AssertLen(t, session.WorkLog, 1)
	core.AssertEqual(t, "Checked build", session.WorkLog[0]["message"])
	core.AssertEqual(t, "checkpoint", session.WorkLog[0]["type"])
}

func TestSession_HandleSessionLog_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionLog(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_log"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionLog_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionLog(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_missing"},
		core.Option{Key: "message", Value: "Checked build"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionArtifact_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID: "ses_artifact",
		AgentType: "codex",
		Status:    "active",
	}))

	result := subsystem.handleSessionArtifact(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_artifact"},
		core.Option{Key: `path`, Value: "pkg/agentic/session.go"},
		core.Option{Key: "action", Value: "modified"},
		core.Option{Key: "metadata", Value: `{"insertions":12}`},
	))
	core.RequireTrue(t, result.OK)

	session, err := readSessionCache("ses_artifact")
	core.RequireNoError(t, err)
	core.AssertLen(t, session.Artifacts, 1)
	core.AssertEqual(t, "pkg/agentic/session.go", session.Artifacts[0][`path`])
	core.AssertEqual(t, "modified", session.Artifacts[0]["action"])
}

func TestSession_HandleSessionArtifact_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionArtifact(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_artifact"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionArtifact_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionArtifact(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_artifact"},
		core.Option{Key: `path`, Value: "pkg/agentic/session.go"},
		core.Option{Key: "action", Value: "modified"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionHandoff_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireNoError(t, writeSessionCache(&Session{
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
	core.RequireTrue(t, result.OK)

	session, err := readSessionCache("ses_handoff")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "handed_off", session.Status)
	core.AssertEqual(t, "Ready for review", session.Handoff["summary"])
}

func TestSession_HandleSessionHandoff_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionHandoff(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_handoff"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionHandoff_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionHandoff(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_handoff"},
		core.Option{Key: "summary", Value: "Ready for review"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionResume_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID: "ses_resume",
		AgentType: "codex",
		Status:    "handed_off",
		Handoff: map[string]any{
			"summary": "Ready for review",
		},
		WorkLog: []map[string]any{
			{"message": "Checked build", "type": "checkpoint", "timestamp": "2026-03-31T10:00:00Z"},
		},
		Artifacts: []map[string]any{
			{`path`: "pkg/agentic/session.go", "action": "modified"},
		},
	}))

	result := subsystem.handleSessionResume(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_resume"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "active", output.Session.Status)
	core.AssertEqual(t, "ses_resume", output.HandoffContext["session_id"])
	handoffNotes, ok := output.HandoffContext["handoff_notes"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Ready for review", handoffNotes["summary"])
	core.AssertLen(t, output.RecentActions, 1)
}

func TestSession_HandleSessionResume_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionResume(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionResume_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionResume(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_resume"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionReplay_Good_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	core.RequireNoError(t, writeSessionCache(&Session{
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
			{`path`: "pkg/agentic/session.go", "action": "modified"},
		},
		Handoff: map[string]any{
			"summary": "Ready for review",
		},
		Summary: "Completed work",
	}))

	result := subsystem.handleSessionReplay(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_replay"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionReplayOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses_replay", output.ReplayContext["session_id"])
	core.AssertEqual(t, 3, output.ReplayContext["total_actions"])
	core.AssertLen(t, output.ReplayContext["work_log"].([]map[string]any), 3)
	core.AssertLen(t, output.ReplayContext["checkpoints"].([]map[string]any), 1)
	core.AssertLen(t, output.ReplayContext["work_log_by_type"].(map[string]any)["error"].([]map[string]any), 1)
}

func TestSession_HandleSessionReplay_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionReplay(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestSession_HandleSessionReplay_Ugly_MissingSession(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleSessionReplay(context.Background(), core.NewOptions(
		core.Option{Key: "session_id", Value: "ses_replay"},
	))
	core.AssertFalse(t, result.OK)
}

func TestSession_normaliseSessionAgentType_Good(t *testing.T) {
	agentType, ok := normaliseSessionAgentType("  CLAUDE:Sonnet  ")
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "sonnet", agentType)
}

func TestSession_normaliseSessionAgentType_Bad(t *testing.T) {
	agentType, ok := normaliseSessionAgentType("codex")
	core.AssertFalse(t, ok)
	core.AssertEmpty(t, agentType)
}

func TestSession_normaliseSessionAgentType_Ugly(t *testing.T) {
	agentType, ok := normaliseSessionAgentType("   ")
	core.AssertFalse(t, ok)
	core.AssertEmpty(t, agentType)
}
