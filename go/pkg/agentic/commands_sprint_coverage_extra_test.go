// SPDX-License-Identifier: EUPL-1.2

// Extra coverage for the sprint command surface: the cmdSprint action
// dispatcher (every case + usage + unknown), the cmd success-print and
// error branches for get / update / archive, and the underlying
// sprintUpdate / sprintArchive request builders driven against a stub
// platform backend.

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// --- cmdSprint: action dispatcher ------------------------------------

// TestCommandsSprint_CmdSprint_Usage_NoAction — no --action prints usage and
// returns OK without touching the backend.
func TestCommandsSprint_CmdSprint_Usage_NoAction(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdSprint(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent sprint")
}

// TestCommandsSprint_CmdSprint_Unknown_Action — an unrecognised --action
// prints usage and returns a non-OK result carrying the unknown command.
func TestCommandsSprint_CmdSprint_Unknown_Action(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdSprint(core.NewOptions(core.Option{Key: "action", Value: "frobnicate"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent sprint")
	core.AssertContains(t, r.Value.(error).Error(), "unknown sprint command: frobnicate")
}

// TestCommandsSprint_CmdSprint_DispatchesByAction — each action routes to the
// matching sub-handler. A stub backend answers every sprint route so the
// dispatch arms all reach their handler. The create + list + get + update +
// archive cases (and their aliases show/delete) are exercised through the
// single dispatcher entry point.
func TestCommandsSprint_CmdSprint_DispatchesByAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Uniform sprint envelope works for every verb/route.
		_, _ = w.Write([]byte(`{"data":{"sprint":{"id":7,"slug":"ax","title":"AX","status":"active"},"sprints":[],"total":0}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	for _, action := range []string{"create", "get", "show", "list", "update", "archive", "delete"} {
		action := action
		t.Run(action, func(t *testing.T) {
			opts := core.NewOptions(
				core.Option{Key: "action", Value: action},
				core.Option{Key: "title", Value: "AX"},      // create/update need a field
				core.Option{Key: "_arg", Value: "ax"},       // get/update/archive need an id
				core.Option{Key: "status", Value: "active"}, // update field
			)
			out := captureStdout(t, func() {
				r := s.cmdSprint(opts)
				core.AssertTrue(t, r.OK)
			})
			_ = out
		})
	}
}

// --- cmdSprintGet: success + error -----------------------------------

// TestCommandsSprint_CmdSprintGet_Good_PrintsSprint — a populated get renders
// the slug / title / status / goal lines.
func TestCommandsSprint_CmdSprintGet_Good_PrintsSprint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints/ax", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"sprint":{"slug":"ax","title":"AX Follow-up","status":"active","goal":"ship it"}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdSprintGet(core.NewOptions(core.Option{Key: "_arg", Value: "ax"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "slug:  ax")
	core.AssertContains(t, out, "goal:  ship it")
}

// TestCommandsSprint_CmdSprintGet_Bad_MissingIdentifier — no slug/id prints
// usage and returns non-OK.
func TestCommandsSprint_CmdSprintGet_Bad_MissingIdentifier(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdSprintGet(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent sprint get")
}

// TestCommandsSprint_CmdSprintGet_Bad_BackendError — a 500 from the backend
// surfaces as an error result.
func TestCommandsSprint_CmdSprintGet_Bad_BackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	captureStdout(t, func() {
		r = s.cmdSprintGet(core.NewOptions(core.Option{Key: "_arg", Value: "ax"}))
	})
	core.AssertFalse(t, r.OK)
}

// --- cmdSprintUpdate: success + error --------------------------------

// TestCommandsSprint_CmdSprintUpdate_Good_PrintsSprint — an update with at
// least one field renders the updated sprint.
func TestCommandsSprint_CmdSprintUpdate_Good_PrintsSprint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "/v1/sprints/ax", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"sprint":{"slug":"ax","title":"Renamed","status":"completed","goal":"g"}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdSprintUpdate(core.NewOptions(
			core.Option{Key: "_arg", Value: "ax"},
			core.Option{Key: "title", Value: "Renamed"},
			core.Option{Key: "status", Value: "completed"},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "title: Renamed")
	core.AssertContains(t, out, "status: completed")
}

// TestCommandsSprint_CmdSprintUpdate_Bad_MissingIdentifier — no slug/id prints
// usage and returns non-OK.
func TestCommandsSprint_CmdSprintUpdate_Bad_MissingIdentifier(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdSprintUpdate(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "usage: core-agent sprint update")
}

// --- cmdSprintArchive: success-print ---------------------------------

// TestCommandsSprint_CmdSprintArchive_Good_PrintsArchived — a successful
// archive prints the archived identifier.
func TestCommandsSprint_CmdSprintArchive_Good_PrintsArchived(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodDelete, r.Method)
		core.AssertEqual(t, "/v1/sprints/ax", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"sprint":{"slug":"ax","success":true}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdSprintArchive(core.NewOptions(core.Option{Key: "_arg", Value: "ax"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "archived: ax")
}

// --- sprintUpdate (underlying request builder) -----------------------

// TestSprint_SprintUpdate_Bad_NoFields — an update with no fields set fails
// the "at least one field" guard before any request.
func TestSprint_SprintUpdate_Bad_NoFields(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	r := s.sprintUpdate(context.Background(), SprintUpdateInput{Slug: "ax"})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "at least one field is required")
}

// TestSprint_SprintUpdate_Bad_MissingIdentifier — no slug/id fails up front.
func TestSprint_SprintUpdate_Bad_MissingIdentifier(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	r := s.sprintUpdate(context.Background(), SprintUpdateInput{Title: "x"})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "id or slug is required")
}

// TestSprint_SprintUpdate_Good_AllFields — every optional field lands in the
// PATCH body and the response parses into a SprintOutput.
func TestSprint_SprintUpdate_Good_AllFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := core.ReadAll(r.Body)
		core.RequireTrue(t, body.OK)
		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(body.Value.(string), &payload).OK)
		core.AssertEqual(t, "T", payload["title"])
		core.AssertEqual(t, "G", payload["goal"])
		core.AssertEqual(t, "active", payload["status"])
		core.AssertEqual(t, "2026-01-01", payload["started_at"])
		core.AssertEqual(t, "2026-02-01", payload["ended_at"])
		_, _ = w.Write([]byte(`{"data":{"sprint":{"id":9,"slug":"ax","title":"T","status":"active"}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	r := s.sprintUpdate(context.Background(), SprintUpdateInput{
		ID:        "9",
		Title:     "T",
		Goal:      "G",
		Status:    "active",
		Metadata:  map[string]any{"k": "v"},
		StartedAt: "2026-01-01",
		EndedAt:   "2026-02-01",
	})
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(SprintOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 9, out.Sprint.ID)
}

// --- sprintArchive (underlying request builder) ----------------------

// TestSprint_SprintArchive_Good_ResourceOverride — when the archive response
// carries a sprint resource, its slug + success flag override the defaults.
func TestSprint_SprintArchive_Good_ResourceOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodDelete, r.Method)
		_, _ = w.Write([]byte(`{"data":{"sprint":{"slug":"renamed-ax","success":true}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	r := s.sprintArchive(context.Background(), SprintArchiveInput{ID: "9"})
	core.RequireTrue(t, r.OK)
	out, ok := r.Value.(SprintArchiveOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "renamed-ax", out.Archived)
	core.AssertTrue(t, out.Success)
}

// TestSprint_SprintArchive_Bad_MissingIdentifier — no slug/id fails up front.
func TestSprint_SprintArchive_Bad_MissingIdentifier(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	r := s.sprintArchive(context.Background(), SprintArchiveInput{})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "id or slug is required")
}
