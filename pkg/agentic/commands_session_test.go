// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestCommandsSession_RegisterSessionCommands_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerSessionCommands()

	core.AssertContains(t, c.Commands(), "session/get")
	core.AssertContains(t, c.Commands(), "agentic:session/get")
	core.AssertContains(t, c.Commands(), "session/list")
	core.AssertContains(t, c.Commands(), "agentic:session/list")
	core.AssertContains(t, c.Commands(), "session/handoff")
	core.AssertContains(t, c.Commands(), "agentic:session/handoff")
	core.AssertContains(t, c.Commands(), "session/start")
	core.AssertContains(t, c.Commands(), "agentic:session/start")
	core.AssertContains(t, c.Commands(), "session/continue")
	core.AssertContains(t, c.Commands(), "agentic:session/continue")
	core.AssertContains(t, c.Commands(), "session/end")
	core.AssertContains(t, c.Commands(), "agentic:session/end")
	core.AssertContains(t, c.Commands(), "session/complete")
	core.AssertContains(t, c.Commands(), "agentic:session/complete")
	core.AssertContains(t, c.Commands(), "session/log")
	core.AssertContains(t, c.Commands(), "agentic:session/log")
	core.AssertContains(t, c.Commands(), "session/artifact")
	core.AssertContains(t, c.Commands(), "agentic:session/artifact")
	core.AssertContains(t, c.Commands(), "session/resume")
	core.AssertContains(t, c.Commands(), "agentic:session/resume")
	core.AssertContains(t, c.Commands(), "session/replay")
	core.AssertContains(t, c.Commands(), "agentic:session/replay")
}

func TestCommandsSession_CmdSessionGet_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions/ses-get", r.URL.Path)
		core.AssertEqual(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{"data":{"session_id":"ses-get","plan_slug":"ax-follow-up","agent_type":"codex","status":"active","summary":"Working","created_at":"2026-03-31T12:00:00Z","updated_at":"2026-03-31T12:30:00Z","work_log":[{"type":"checkpoint","message":"started"}],"artifacts":[{"path":"pkg/agentic/session.go","action":"modified"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")

	output := captureStdout(t, func() {
		result := subsystem.cmdSessionGet(core.NewOptions(core.Option{Key: "_arg", Value: "ses-get"}))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "session: ses-get")
	core.AssertContains(t, output, "plan:    ax-follow-up")
	core.AssertContains(t, output, "work log: 1 item(s)")
	core.AssertContains(t, output, "artifacts: 1 item(s)")
}

func TestCommandsSession_CmdSessionList_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions", r.URL.Path)
		core.AssertEqual(t, "ax-follow-up", r.URL.Query().Get("plan_slug"))
		core.AssertEqual(t, "codex", r.URL.Query().Get("agent_type"))
		core.AssertEqual(t, "active", r.URL.Query().Get("status"))
		core.AssertEqual(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":[{"session_id":"ses-1","plan_slug":"ax-follow-up","agent_type":"codex","status":"active"},{"session_id":"ses-2","agent_type":"claude","status":"paused"}],"count":2}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")

	output := captureStdout(t, func() {
		result := subsystem.cmdSessionList(core.NewOptions(
			core.Option{Key: "plan_slug", Value: "ax-follow-up"},
			core.Option{Key: "agent_type", Value: "codex"},
			core.Option{Key: "status", Value: "active"},
			core.Option{Key: "limit", Value: 5},
		))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "ses-1")
	core.AssertContains(t, output, "2 session(s)")
}

func TestCommandsSession_CmdSessionStart_Good(t *testing.T) {
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

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses-start","plan_slug":"ax-follow-up","agent_type":"opus","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionStart(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "agent_type", Value: "opus"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses-start", output.Session.SessionID)
	core.AssertEqual(t, "ax-follow-up", output.Session.PlanSlug)
	core.AssertEqual(t, "opus", output.Session.AgentType)
}

func TestCommandsSession_CmdSessionStart_Good_CanonicalAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "opus", payload["agent_type"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses-start","plan_slug":"ax-follow-up","agent_type":"opus","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionStart(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "agent_type", Value: "claude:opus"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses-start", output.Session.SessionID)
	core.AssertEqual(t, "ax-follow-up", output.Session.PlanSlug)
	core.AssertEqual(t, "opus", output.Session.AgentType)
}

func TestCommandsSession_CmdSessionStart_Bad_MissingPlanSlug(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSessionStart(core.NewOptions(core.Option{Key: "agent_type", Value: "opus"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "plan_slug is required")
}

func TestCommandsSession_CmdSessionStart_Bad_InvalidAgentType(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSessionStart(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "agent_type", Value: "codex"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "claude:opus")
}

func TestCommandsSession_CmdSessionStart_Ugly_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionStart(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "agent_type", Value: "codex"},
	))
	core.AssertFalse(t, result.OK)
}

func TestCommandsSession_CmdSessionContinue_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sessions/ses-continue/continue", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "codex", payload["agent_type"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses-continue","agent_type":"codex","status":"active","work_log":[{"type":"checkpoint","message":"continue"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionContinue(core.NewOptions(
		core.Option{Key: "_arg", Value: "ses-continue"},
		core.Option{Key: "agent_type", Value: "codex"},
		core.Option{Key: "work_log", Value: []map[string]any{{"type": "checkpoint", "message": "continue"}}},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "ses-continue", output.Session.SessionID)
	core.AssertEqual(t, "codex", output.Session.AgentType)
	core.AssertLen(t, output.Session.WorkLog, 1)
}

func TestCommandsSession_CmdSessionContinue_Bad_MissingSessionID(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSessionContinue(core.NewOptions(core.Option{Key: "agent_type", Value: "codex"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "session_id is required")
}

func TestCommandsSession_CmdSessionContinue_Ugly_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionContinue(core.NewOptions(
		core.Option{Key: "_arg", Value: "ses-continue"},
		core.Option{Key: "agent_type", Value: "codex"},
	))
	core.AssertFalse(t, result.OK)
}

func TestCommandsSession_CmdSessionHandoff_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID: "ses-handoff",
		AgentType: "codex",
		Status:    "active",
		WorkLog: []map[string]any{
			{"type": "checkpoint", "message": "build passed"},
			{"type": "decision", "message": "hand off for review"},
		},
	}))

	result := s.cmdSessionHandoff(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-handoff"},
		core.Option{Key: "summary", Value: "Ready for review"},
		core.Option{Key: "next_steps", Value: []string{"Run the verifier", "Merge if clean"}},
		core.Option{Key: "blockers", Value: []string{"Need final approval"}},
		core.Option{Key: "context_for_next", Value: map[string]any{"repo": "go-io"}},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionHandoffOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "ses-handoff", output.HandoffContext["session_id"])
	handoffNotes, ok := output.HandoffContext["handoff_notes"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Ready for review", handoffNotes["summary"])

	cached, err := readSessionCache("ses-handoff")
	core.RequireNoError(t, err)
	core.AssertNotNil(t, cached)
	core.AssertEqual(t, "handed_off", cached.Status)
	core.AssertNotEmpty(t, cached.Handoff)
}

func TestCommandsSession_CmdSessionHandoff_Bad_MissingSummary(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionHandoff(core.NewOptions(core.Option{Key: "session_id", Value: "ses-handoff"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "summary is required")
}

func TestCommandsSession_CmdSessionHandoff_Ugly_CorruptedCacheFallsBackToRemoteError(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireTrue(t, fs.EnsureDir(sessionCacheRoot()).OK)
	core.RequireTrue(t, fs.WriteAtomic(sessionCachePath("ses-bad"), "{not-json").OK)

	result := s.cmdSessionHandoff(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-bad"},
		core.Option{Key: "summary", Value: "Ready for review"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "no platform API key configured")
}

func TestCommandsSession_CmdSessionEnd_Good(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/v1/sessions/ses-end/end":
			core.AssertEqual(t, http.MethodPost, r.Method)

			bodyResult := core.ReadAll(r.Body)
			core.RequireTrue(t, bodyResult.OK)

			var payload map[string]any
			parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
			core.RequireTrue(t, parseResult.OK)
			core.AssertEqual(t, "completed", payload["status"])
			core.AssertEqual(t, "Ready for review", payload["summary"])

			handoffNotes, ok := payload["handoff_notes"].(map[string]any)
			core.RequireTrue(t, ok)
			core.AssertEqual(t, "Ready for review", handoffNotes["summary"])
			core.AssertEqual(t, []any{"Run the verifier"}, handoffNotes["next_steps"])

			_, _ = w.Write([]byte(`{"data":{"session_id":"ses-end","agent_type":"codex","status":"completed","summary":"Ready for review","handoff":{"summary":"Ready for review","next_steps":["Run the verifier"]},"ended_at":"2026-03-31T12:00:00Z"}}`))
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
			core.AssertContains(t, content, "Session handoff: ses-end")
			core.AssertContains(t, content, "Ready for review")
			core.AssertContains(t, content, "Run the verifier")

			_, _ = w.Write([]byte(`{"data":{"id":"mem_end"}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionEnd(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-end"},
		core.Option{Key: "summary", Value: "Ready for review"},
		core.Option{Key: "handoff_notes", Value: `{"summary":"Ready for review","next_steps":["Run the verifier"]}`},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", output.Session.Status)
	core.AssertEqual(t, "Ready for review", output.Session.Summary)
	core.AssertNotNil(t, output.Session.Handoff)
	core.AssertEqual(t, "Ready for review", output.Session.Handoff["summary"])
	core.AssertEqual(t, 2, callCount)
}

func TestCommandsSession_CmdSessionEnd_Bad_MissingSummary(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdSessionEnd(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-end"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "summary is required")
}

func TestCommandsSession_CmdSessionEnd_Ugly_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionEnd(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-end"},
		core.Option{Key: "summary", Value: "Ready for review"},
	))
	core.AssertFalse(t, result.OK)
}

func TestCommandsSession_CmdSessionLog_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID: "ses-log",
		AgentType: "codex",
		Status:    "active",
		WorkLog: []map[string]any{
			{"type": "checkpoint", "message": "build passed"},
		},
	}))

	result := s.cmdSessionLog(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-log"},
		core.Option{Key: "message", Value: "Checked build"},
		core.Option{Key: "type", Value: "checkpoint"},
		core.Option{Key: "data", Value: map[string]any{"repo": "go-io"}},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionLogOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "Checked build", output.Logged)

	cached, err := readSessionCache("ses-log")
	core.RequireNoError(t, err)
	core.AssertNotNil(t, cached)
	core.AssertLen(t, cached.WorkLog, 2)
	core.AssertEqual(t, "checkpoint", cached.WorkLog[1]["type"])
	core.AssertEqual(t, "Checked build", cached.WorkLog[1]["message"])
}

func TestCommandsSession_CmdSessionLog_Bad_MissingMessage(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionLog(core.NewOptions(core.Option{Key: "session_id", Value: "ses-log"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "message is required")
}

func TestCommandsSession_CmdSessionLog_Ugly_CorruptedCacheFallsBackToRemoteError(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireTrue(t, fs.EnsureDir(sessionCacheRoot()).OK)
	core.RequireTrue(t, fs.WriteAtomic(sessionCachePath("ses-bad"), "{not-json").OK)

	result := s.cmdSessionLog(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-bad"},
		core.Option{Key: "message", Value: "Checked build"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "no platform API key configured")
}

func TestCommandsSession_CmdSessionArtifact_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID: "ses-artifact",
		AgentType: "codex",
		Status:    "active",
	}))

	result := s.cmdSessionArtifact(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-artifact"},
		core.Option{Key: "path", Value: "pkg/agentic/session.go"},
		core.Option{Key: "action", Value: "modified"},
		core.Option{Key: "description", Value: "Tracked session metadata"},
		core.Option{Key: "metadata", Value: map[string]any{"repo": "go-agent"}},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionArtifactOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "pkg/agentic/session.go", output.Artifact)

	cached, err := readSessionCache("ses-artifact")
	core.RequireNoError(t, err)
	core.AssertNotNil(t, cached)
	core.AssertLen(t, cached.Artifacts, 1)
	core.AssertEqual(t, "modified", cached.Artifacts[0]["action"])
	core.AssertEqual(t, "pkg/agentic/session.go", cached.Artifacts[0]["path"])
	metadata, ok := cached.Artifacts[0]["metadata"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Tracked session metadata", metadata["description"])
	core.AssertEqual(t, "go-agent", metadata["repo"])
}

func TestCommandsSession_CmdSessionArtifact_Bad_MissingPath(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionArtifact(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-artifact"},
		core.Option{Key: "action", Value: "modified"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "path is required")
}

func TestCommandsSession_CmdSessionResume_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID:      "ses-abc123",
		AgentType:      "codex",
		Status:         "paused",
		ContextSummary: map[string]any{"repo": "go-io"},
		WorkLog: []map[string]any{
			{"type": "checkpoint", "message": "build passed"},
			{"type": "decision", "message": "open PR"},
		},
		Artifacts: []map[string]any{
			{"path": "pkg/agentic/session.go", "action": "modified"},
		},
		Handoff: map[string]any{
			"summary": "Ready for review",
		},
	}))

	result := s.cmdSessionResume(core.NewOptions(core.Option{Key: "session_id", Value: "ses-abc123"}))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "ses-abc123", output.Session.SessionID)
	core.AssertEqual(t, "active", output.Session.Status)
	core.AssertEqual(t, "ses-abc123", output.HandoffContext["session_id"])
	handoffNotes, ok := output.HandoffContext["handoff_notes"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "Ready for review", handoffNotes["summary"])
	core.AssertLen(t, output.RecentActions, 2)
	core.AssertLen(t, output.Artifacts, 1)
}

func TestCommandsSession_CmdSessionResume_Bad_MissingSessionID(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionResume(core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "session_id is required")
}

func TestCommandsSession_CmdSessionResume_Ugly_CorruptedCacheFallsBackToRemoteError(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireTrue(t, fs.EnsureDir(sessionCacheRoot()).OK)
	core.RequireTrue(t, fs.WriteAtomic(sessionCachePath("ses-bad"), "{not-json").OK)

	result := s.cmdSessionResume(core.NewOptions(core.Option{Key: "session_id", Value: "ses-bad"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "no platform API key configured")
}

func TestCommandsSession_CmdSessionReplay_Good(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID: "ses-replay",
		AgentType: "codex",
		Status:    "active",
		WorkLog: []map[string]any{
			{"type": "checkpoint", "message": "started", "timestamp": time.Now().Format(time.RFC3339)},
			{"type": "decision", "message": "kept scope small", "timestamp": time.Now().Format(time.RFC3339)},
			{"type": "error", "message": "flaky test", "timestamp": time.Now().Format(time.RFC3339)},
		},
		Artifacts: []map[string]any{
			{"path": "pkg/agentic/commands_session.go", "action": "created"},
		},
	}))

	result := s.cmdSessionReplay(core.NewOptions(core.Option{Key: "session_id", Value: "ses-replay"}))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(SessionReplayOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "ses-replay", output.ReplayContext["session_id"])
	core.AssertContains(t, output.ReplayContext, "checkpoints")
	core.AssertContains(t, output.ReplayContext, "decisions")
	core.AssertContains(t, output.ReplayContext, "errors")
}

func TestCommandsSession_CmdSessionReplay_Bad_MissingSessionID(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionReplay(core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "session_id is required")
}
