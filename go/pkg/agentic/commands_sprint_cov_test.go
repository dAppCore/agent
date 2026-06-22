// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// covSprintErrServer fails every request with 500 so the sprint command error
// arms are exercised.
func covSprintErrServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"sprint backend down"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestCommandsSprintCov_CmdSprintCreate_Ugly_BackendError — a failing backend
// hits the create error arm.
func TestCommandsSprintCov_CmdSprintCreate_Ugly_BackendError(t *testing.T) {
	s := testPrepWithPlatformServer(t, covSprintErrServer(t), "secret-token")
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSprintCreate(core.NewOptions(core.Option{Key: "title", Value: "AX Follow-up"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsSprintCov_CmdSprintUpdate_Ugly_BackendError — a failing backend
// hits the update error arm.
func TestCommandsSprintCov_CmdSprintUpdate_Ugly_BackendError(t *testing.T) {
	s := testPrepWithPlatformServer(t, covSprintErrServer(t), "secret-token")
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSprintUpdate(core.NewOptions(
			core.Option{Key: "_arg", Value: "ax-follow-up"},
			core.Option{Key: "status", Value: "completed"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsSprintCov_CmdSprintArchive_Ugly_BackendError — a failing backend
// hits the archive error arm.
func TestCommandsSprintCov_CmdSprintArchive_Ugly_BackendError(t *testing.T) {
	s := testPrepWithPlatformServer(t, covSprintErrServer(t), "secret-token")
	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSprintArchive(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, output, "error:")
}

// TestCommandsSprintCov_CmdSprintCreate_Good_PrintsGoal — a created sprint with a
// goal exercises the optional goal print line.
func TestCommandsSprintCov_CmdSprintCreate_Good_PrintsGoal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"sprint":{"id":3,"slug":"ax","title":"AX","goal":"Finish parity","status":"active"}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdSprintCreate(core.NewOptions(
			core.Option{Key: "title", Value: "AX"},
			core.Option{Key: "goal", Value: "Finish parity"},
		))
	})
	core.RequireTrue(t, r.OK)
	core.AssertContains(t, output, "slug:  ax")
	core.AssertContains(t, output, "goal:  Finish parity")
}

// TestCommandsSprintCov_CmdSprintCreate_Bad_MissingTitle — no title prints usage
// and returns the required-field error.
func TestCommandsSprintCov_CmdSprintCreate_Bad_MissingTitle(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	output := captureStdout(t, func() { r = s.cmdSprintCreate(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "title is required")
	core.AssertContains(t, output, "usage: core-agent sprint create")
}

// TestCommandsSprintCov_RegisterSprintCommands_Ugly_DuplicateConflict — a second
// registration fails on the first duplicate command.
func TestCommandsSprintCov_RegisterSprintCommands_Ugly_DuplicateConflict(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireTrue(t, s.registerSprintCommands().OK)
	core.AssertFalse(t, s.registerSprintCommands().OK)
}
