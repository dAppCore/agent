// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestCommandsSprint_RegisterCommands_Good_Case(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerSprintCommands()

	core.AssertContains(t, c.Commands(), "sprint")
	core.AssertContains(t, c.Commands(), "agentic:sprint")
	core.AssertContains(t, c.Commands(), "sprint/create")
	core.AssertContains(t, c.Commands(), "agentic:sprint/create")
	core.AssertContains(t, c.Commands(), "sprint/get")
	core.AssertContains(t, c.Commands(), "agentic:sprint/get")
	core.AssertContains(t, c.Commands(), "sprint/list")
	core.AssertContains(t, c.Commands(), "agentic:sprint/list")
	core.AssertContains(t, c.Commands(), "sprint/update")
	core.AssertContains(t, c.Commands(), "agentic:sprint/update")
	core.AssertContains(t, c.Commands(), "sprint/archive")
	core.AssertContains(t, c.Commands(), "agentic:sprint/archive")
}

func TestCommandsSprint_CmdSprintCreate_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		core.AssertEqual(t, "AX Follow-up", payload["title"])
		core.AssertEqual(t, "Finish RFC parity", payload["goal"])
		core.AssertEqual(t, "active", payload["status"])

		_, _ = w.Write([]byte(`{"data":{"sprint":{"id":7,"slug":"ax-follow-up","title":"AX Follow-up","goal":"Finish RFC parity","status":"active"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdSprintCreate(core.NewOptions(
			core.Option{Key: "title", Value: "AX Follow-up"},
			core.Option{Key: "goal", Value: "Finish RFC parity"},
			core.Option{Key: "status", Value: "active"},
		))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "slug:  ax-follow-up")
	core.AssertContains(t, output, "title: AX Follow-up")
	core.AssertContains(t, output, "status: active")
	core.AssertContains(t, output, "goal:  Finish RFC parity")
}

func TestCommandsSprint_CmdSprintList_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/sprints", r.URL.Path)
		core.AssertEqual(t, "active", r.URL.Query().Get("status"))
		core.AssertEqual(t, "5", r.URL.Query().Get("limit"))

		_, _ = w.Write([]byte(`{"data":[{"id":1,"slug":"ax-follow-up","title":"AX Follow-up","status":"active"},{"id":2,"slug":"rfc-parity","title":"RFC Parity","status":"active"}],"count":2}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdSprintList(core.NewOptions(
			core.Option{Key: "status", Value: "active"},
			core.Option{Key: "limit", Value: 5},
		))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "ax-follow-up")
	core.AssertContains(t, output, "rfc-parity")
	core.AssertContains(t, output, "2 sprint(s)")
}

func TestCommandsSprint_CmdSprintArchive_Bad_MissingIdentifier(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSprintArchive(core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "id or slug is required")
}

func TestCommandsSprint_CmdSprintGet_Ugly_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSprintGet(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))
	core.AssertFalse(t, result.OK)
}
