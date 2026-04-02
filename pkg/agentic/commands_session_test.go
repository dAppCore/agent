// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsSession_RegisterSessionCommands_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerSessionCommands()

	assert.Contains(t, c.Commands(), "session/get")
	assert.Contains(t, c.Commands(), "agentic:session/get")
	assert.Contains(t, c.Commands(), "session/list")
	assert.Contains(t, c.Commands(), "agentic:session/list")
	assert.Contains(t, c.Commands(), "session/handoff")
	assert.Contains(t, c.Commands(), "agentic:session/handoff")
	assert.Contains(t, c.Commands(), "session/start")
	assert.Contains(t, c.Commands(), "agentic:session/start")
	assert.Contains(t, c.Commands(), "session/continue")
	assert.Contains(t, c.Commands(), "agentic:session/continue")
	assert.Contains(t, c.Commands(), "session/end")
	assert.Contains(t, c.Commands(), "agentic:session/end")
	assert.Contains(t, c.Commands(), "session/complete")
	assert.Contains(t, c.Commands(), "agentic:session/complete")
	assert.Contains(t, c.Commands(), "session/log")
	assert.Contains(t, c.Commands(), "agentic:session/log")
	assert.Contains(t, c.Commands(), "session/artifact")
	assert.Contains(t, c.Commands(), "agentic:session/artifact")
	assert.Contains(t, c.Commands(), "session/resume")
	assert.Contains(t, c.Commands(), "agentic:session/resume")
	assert.Contains(t, c.Commands(), "session/replay")
	assert.Contains(t, c.Commands(), "agentic:session/replay")
}

func TestCommandsSession_CmdSessionGet_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions/ses-get", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{"data":{"session_id":"ses-get","plan_slug":"ax-follow-up","agent_type":"codex","status":"active","summary":"Working","created_at":"2026-03-31T12:00:00Z","updated_at":"2026-03-31T12:30:00Z","work_log":[{"type":"checkpoint","message":"started"}],"artifacts":[{"path":"pkg/agentic/session.go","action":"modified"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")

	output := captureStdout(t, func() {
		result := subsystem.cmdSessionGet(core.NewOptions(core.Option{Key: "_arg", Value: "ses-get"}))
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "session: ses-get")
	assert.Contains(t, output, "plan:    ax-follow-up")
	assert.Contains(t, output, "work log: 1 item(s)")
	assert.Contains(t, output, "artifacts: 1 item(s)")
}

func TestCommandsSession_CmdSessionList_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions", r.URL.Path)
		require.Equal(t, "ax-follow-up", r.URL.Query().Get("plan_slug"))
		require.Equal(t, "codex", r.URL.Query().Get("agent_type"))
		require.Equal(t, "active", r.URL.Query().Get("status"))
		require.Equal(t, "5", r.URL.Query().Get("limit"))
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
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "ses-1")
	assert.Contains(t, output, "2 session(s)")
}

func TestCommandsSession_CmdSessionStart_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		assert.Equal(t, "opus", payload["agent_type"])
		assert.Equal(t, "ax-follow-up", payload["plan_slug"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses-start","plan_slug":"ax-follow-up","agent_type":"opus","status":"active"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionStart(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "agent_type", Value: "opus"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	assert.Equal(t, "ses-start", output.Session.SessionID)
	assert.Equal(t, "ax-follow-up", output.Session.PlanSlug)
	assert.Equal(t, "opus", output.Session.AgentType)
}

func TestCommandsSession_CmdSessionStart_Bad_MissingPlanSlug(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSessionStart(core.NewOptions(core.Option{Key: "agent_type", Value: "opus"}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "plan_slug is required")
}

func TestCommandsSession_CmdSessionStart_Bad_InvalidAgentType(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSessionStart(core.NewOptions(
		core.Option{Key: "_arg", Value: "ax-follow-up"},
		core.Option{Key: "agent_type", Value: "codex"},
	))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "opus, sonnet, or haiku")
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
	assert.False(t, result.OK)
}

func TestCommandsSession_CmdSessionContinue_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sessions/ses-continue/continue", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		assert.Equal(t, "codex", payload["agent_type"])

		_, _ = w.Write([]byte(`{"data":{"session_id":"ses-continue","agent_type":"codex","status":"active","work_log":[{"type":"checkpoint","message":"continue"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSessionContinue(core.NewOptions(
		core.Option{Key: "_arg", Value: "ses-continue"},
		core.Option{Key: "agent_type", Value: "codex"},
		core.Option{Key: "work_log", Value: []map[string]any{{"type": "checkpoint", "message": "continue"}}},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	assert.Equal(t, "ses-continue", output.Session.SessionID)
	assert.Equal(t, "codex", output.Session.AgentType)
	require.Len(t, output.Session.WorkLog, 1)
}

func TestCommandsSession_CmdSessionContinue_Bad_MissingSessionID(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSessionContinue(core.NewOptions(core.Option{Key: "agent_type", Value: "codex"}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "session_id is required")
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
	assert.False(t, result.OK)
}

func TestCommandsSession_CmdSessionHandoff_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.NoError(t, writeSessionCache(&Session{
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
	require.True(t, result.OK)

	output, ok := result.Value.(SessionHandoffOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "ses-handoff", output.HandoffContext["session_id"])
	handoffNotes, ok := output.HandoffContext["handoff_notes"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Ready for review", handoffNotes["summary"])

	cached, err := readSessionCache("ses-handoff")
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, "handed_off", cached.Status)
	assert.NotEmpty(t, cached.Handoff)
}

func TestCommandsSession_CmdSessionHandoff_Bad_MissingSummary(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionHandoff(core.NewOptions(core.Option{Key: "session_id", Value: "ses-handoff"}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "summary is required")
}

func TestCommandsSession_CmdSessionHandoff_Ugly_CorruptedCacheFallsBackToRemoteError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.True(t, fs.EnsureDir(sessionCacheRoot()).OK)
	require.True(t, fs.WriteAtomic(sessionCachePath("ses-bad"), "{not-json").OK)

	result := s.cmdSessionHandoff(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-bad"},
		core.Option{Key: "summary", Value: "Ready for review"},
	))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "no platform API key configured")
}

func TestCommandsSession_CmdSessionEnd_Good(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch r.URL.Path {
		case "/v1/sessions/ses-end/end":
			require.Equal(t, http.MethodPost, r.Method)

			bodyResult := core.ReadAll(r.Body)
			require.True(t, bodyResult.OK)

			var payload map[string]any
			parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
			require.True(t, parseResult.OK)
			require.Equal(t, "completed", payload["status"])
			require.Equal(t, "Ready for review", payload["summary"])

			handoffNotes, ok := payload["handoff_notes"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "Ready for review", handoffNotes["summary"])
			assert.Equal(t, []any{"Run the verifier"}, handoffNotes["next_steps"])

			_, _ = w.Write([]byte(`{"data":{"session_id":"ses-end","agent_type":"codex","status":"completed","summary":"Ready for review","handoff":{"summary":"Ready for review","next_steps":["Run the verifier"]},"ended_at":"2026-03-31T12:00:00Z"}}`))
		case "/v1/brain/remember":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))

			bodyResult := core.ReadAll(r.Body)
			require.True(t, bodyResult.OK)

			var payload map[string]any
			parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
			require.True(t, parseResult.OK)
			assert.Equal(t, "observation", payload["type"])
			assert.Equal(t, "codex", payload["agent_id"])

			content, _ := payload["content"].(string)
			assert.Contains(t, content, "Session handoff: ses-end")
			assert.Contains(t, content, "Ready for review")
			assert.Contains(t, content, "Run the verifier")

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
	require.True(t, result.OK)

	output, ok := result.Value.(SessionOutput)
	require.True(t, ok)
	assert.Equal(t, "completed", output.Session.Status)
	assert.Equal(t, "Ready for review", output.Session.Summary)
	require.NotNil(t, output.Session.Handoff)
	assert.Equal(t, "Ready for review", output.Session.Handoff["summary"])
	assert.Equal(t, 2, callCount)
}

func TestCommandsSession_CmdSessionEnd_Bad_MissingSummary(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.cmdSessionEnd(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-end"},
	))
	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "summary is required")
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
	assert.False(t, result.OK)
}

func TestCommandsSession_CmdSessionLog_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.NoError(t, writeSessionCache(&Session{
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
	require.True(t, result.OK)

	output, ok := result.Value.(SessionLogOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "Checked build", output.Logged)

	cached, err := readSessionCache("ses-log")
	require.NoError(t, err)
	require.NotNil(t, cached)
	require.Len(t, cached.WorkLog, 2)
	assert.Equal(t, "checkpoint", cached.WorkLog[1]["type"])
	assert.Equal(t, "Checked build", cached.WorkLog[1]["message"])
}

func TestCommandsSession_CmdSessionLog_Bad_MissingMessage(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionLog(core.NewOptions(core.Option{Key: "session_id", Value: "ses-log"}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "message is required")
}

func TestCommandsSession_CmdSessionLog_Ugly_CorruptedCacheFallsBackToRemoteError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.True(t, fs.EnsureDir(sessionCacheRoot()).OK)
	require.True(t, fs.WriteAtomic(sessionCachePath("ses-bad"), "{not-json").OK)

	result := s.cmdSessionLog(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-bad"},
		core.Option{Key: "message", Value: "Checked build"},
	))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "no platform API key configured")
}

func TestCommandsSession_CmdSessionArtifact_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.NoError(t, writeSessionCache(&Session{
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
	require.True(t, result.OK)

	output, ok := result.Value.(SessionArtifactOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "pkg/agentic/session.go", output.Artifact)

	cached, err := readSessionCache("ses-artifact")
	require.NoError(t, err)
	require.NotNil(t, cached)
	require.Len(t, cached.Artifacts, 1)
	assert.Equal(t, "modified", cached.Artifacts[0]["action"])
	assert.Equal(t, "pkg/agentic/session.go", cached.Artifacts[0]["path"])
	metadata, ok := cached.Artifacts[0]["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Tracked session metadata", metadata["description"])
	assert.Equal(t, "go-agent", metadata["repo"])
}

func TestCommandsSession_CmdSessionArtifact_Bad_MissingPath(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionArtifact(core.NewOptions(
		core.Option{Key: "session_id", Value: "ses-artifact"},
		core.Option{Key: "action", Value: "modified"},
	))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "path is required")
}

func TestCommandsSession_CmdSessionResume_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.NoError(t, writeSessionCache(&Session{
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
	require.True(t, result.OK)

	output, ok := result.Value.(SessionResumeOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "ses-abc123", output.Session.SessionID)
	assert.Equal(t, "active", output.Session.Status)
	assert.Equal(t, "ses-abc123", output.HandoffContext["session_id"])
	handoffNotes, ok := output.HandoffContext["handoff_notes"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Ready for review", handoffNotes["summary"])
	assert.Len(t, output.RecentActions, 2)
	assert.Len(t, output.Artifacts, 1)
}

func TestCommandsSession_CmdSessionResume_Bad_MissingSessionID(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionResume(core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "session_id is required")
}

func TestCommandsSession_CmdSessionResume_Ugly_CorruptedCacheFallsBackToRemoteError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.True(t, fs.EnsureDir(sessionCacheRoot()).OK)
	require.True(t, fs.WriteAtomic(sessionCachePath("ses-bad"), "{not-json").OK)

	result := s.cmdSessionResume(core.NewOptions(core.Option{Key: "session_id", Value: "ses-bad"}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "no platform API key configured")
}

func TestCommandsSession_CmdSessionReplay_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.NoError(t, writeSessionCache(&Session{
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
	require.True(t, result.OK)

	output, ok := result.Value.(SessionReplayOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "ses-replay", output.ReplayContext["session_id"])
	assert.Contains(t, output.ReplayContext, "checkpoints")
	assert.Contains(t, output.ReplayContext, "decisions")
	assert.Contains(t, output.ReplayContext, "errors")
}

func TestCommandsSession_CmdSessionReplay_Bad_MissingSessionID(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionReplay(core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "session_id is required")
}
