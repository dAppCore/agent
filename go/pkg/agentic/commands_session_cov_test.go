// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// covSessionErrServer fails every request with 500 so the session command
// error-envelope branches are exercised once the local cache misses.
func covSessionErrServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"backend down"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestCommandsSessionCov_CmdSessionGet_Bad_MissingID — no session id prints usage
// and returns the required-field error.
func TestCommandsSessionCov_CmdSessionGet_Bad_MissingID(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	output := captureStdout(t, func() { r = s.cmdSessionGet(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "session_id is required")
	core.AssertContains(t, output, "usage: core-agent session get")
}

// TestCommandsSessionCov_CmdSessionGet_Good_EndedAndSummary drives a completed
// session whose payload carries an ended_at + summary so those optional print
// lines — distinct from the existing active-session test — are exercised.
func TestCommandsSessionCov_CmdSessionGet_Good_EndedAndSummary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
			"data": map[string]any{
				"session_id": "ses-full", "plan_slug": "ax", "agent_type": "codex", "status": "completed",
				"summary": "Done", "ended_at": "2026-03-31T13:00:00Z",
			},
		})))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSessionGet(core.NewOptions(core.Option{Key: "_arg", Value: "ses-full"}))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, "session: ses-full")
	core.AssertContains(t, output, "plan:    ax")
	core.AssertContains(t, output, "summary: Done")
	core.AssertContains(t, output, "ended:   2026-03-31T13:00:00Z")
}

// TestCommandsSessionCov_CmdSessionGet_Ugly_BackendError — a 500 backend with no
// local cache entry hits the error-envelope arm.
func TestCommandsSessionCov_CmdSessionGet_Ugly_BackendError(t *testing.T) {
	s := testPrepWithPlatformServer(t, covSessionErrServer(t), "secret-token")
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSessionGet(core.NewOptions(core.Option{Key: "_arg", Value: "ses-missing"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsSessionCov_CmdSessionList_Good_EmptyList — a zero-count list prints
// the "no sessions" line and returns OK.
func TestCommandsSessionCov_CmdSessionList_Good_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"count":0}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	output := captureStdout(t, func() { r = s.cmdSessionList(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, output, "no sessions")
}

// TestCommandsSessionCov_CmdSessionList_Ugly_BackendError — a 500 backend hits
// the list error arm.
func TestCommandsSessionCov_CmdSessionList_Ugly_BackendError(t *testing.T) {
	s := testPrepWithPlatformServer(t, covSessionErrServer(t), "secret-token")
	var r core.Result
	output := captureStdout(t, func() { r = s.cmdSessionList(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsSessionCov_CmdSessionArtifact_Bad_MissingFields — missing path and
// missing action each return their required-field errors.
func TestCommandsSessionCov_CmdSessionArtifact_Bad_MissingFields(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")

	// session id present, path missing.
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdSessionArtifact(core.NewOptions(core.Option{Key: "_arg", Value: "ses-1"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "path is required")

	// session id + path present, action missing.
	captureStdout(t, func() {
		r = s.cmdSessionArtifact(core.NewOptions(
			core.Option{Key: "_arg", Value: "ses-1"},
			core.Option{Key: "path", Value: "x.go"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "action is required")
}

// TestCommandsSessionCov_CmdSessionArtifact_Ugly_BackendError — a 500 backend
// hits the artifact error arm.
func TestCommandsSessionCov_CmdSessionArtifact_Ugly_BackendError(t *testing.T) {
	s := testPrepWithPlatformServer(t, covSessionErrServer(t), "secret-token")
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSessionArtifact(core.NewOptions(
			core.Option{Key: "_arg", Value: "ses-1"},
			core.Option{Key: "path", Value: "x.go"},
			core.Option{Key: "action", Value: "modified"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsSessionCov_CmdSessionReplay_Ugly_BackendError — a 500 backend hits
// the replay error arm.
func TestCommandsSessionCov_CmdSessionReplay_Ugly_BackendError(t *testing.T) {
	s := testPrepWithPlatformServer(t, covSessionErrServer(t), "secret-token")
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSessionReplay(core.NewOptions(core.Option{Key: "_arg", Value: "ses-1"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsSessionCov_CmdSessionHandoff_Bad_MissingSessionID — no session id
// prints usage and returns the required-field error (distinct from the existing
// missing-summary test).
func TestCommandsSessionCov_CmdSessionHandoff_Bad_MissingSessionID(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSessionHandoff(core.NewOptions(core.Option{Key: "summary", Value: "ready"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "session_id is required")
	core.AssertContains(t, output, "usage: core-agent session handoff")
}

// TestCommandsSessionCov_CmdSessionHandoff_Good_PrintsSummary — a handoff over a
// cached session prints the session + summary lines. (The top-level
// blockers/next-steps print arms are unreachable: sessionHandoffContext nests
// those under handoff_notes, so HandoffContext has no top-level keys for them.)
func TestCommandsSessionCov_CmdSessionHandoff_Good_PrintsSummary(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireNoError(t, writeSessionCache(&Session{
		SessionID: "ses-h",
		AgentType: "codex",
		Status:    "active",
	}))

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSessionHandoff(core.NewOptions(
			core.Option{Key: "_arg", Value: "ses-h"},
			core.Option{Key: "summary", Value: "Ready for review"},
			core.Option{Key: "next_steps", Value: []string{"Run the verifier"}},
			core.Option{Key: "blockers", Value: []string{"Needs input"}},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, "session: ses-h")
	core.AssertContains(t, output, "summary: Ready for review")
}

// TestCommandsSessionCov_RegisterSessionCommands_Ugly_DuplicateConflict — a
// second registration fails on the first duplicate command.
func TestCommandsSessionCov_RegisterSessionCommands_Ugly_DuplicateConflict(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireTrue(t, s.registerSessionCommands().OK)
	core.AssertFalse(t, s.registerSessionCommands().OK)
}
