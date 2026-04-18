// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsSprint_RegisterCommands_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerSprintCommands()

	assert.Contains(t, c.Commands(), "sprint")
	assert.Contains(t, c.Commands(), "agentic:sprint")
	assert.Contains(t, c.Commands(), "sprint/create")
	assert.Contains(t, c.Commands(), "agentic:sprint/create")
	assert.Contains(t, c.Commands(), "sprint/get")
	assert.Contains(t, c.Commands(), "agentic:sprint/get")
	assert.Contains(t, c.Commands(), "sprint/list")
	assert.Contains(t, c.Commands(), "agentic:sprint/list")
	assert.Contains(t, c.Commands(), "sprint/update")
	assert.Contains(t, c.Commands(), "agentic:sprint/update")
	assert.Contains(t, c.Commands(), "sprint/archive")
	assert.Contains(t, c.Commands(), "agentic:sprint/archive")
}

func TestCommandsSprint_CmdSprintCreate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sprints", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		require.True(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
		require.Equal(t, "AX Follow-up", payload["title"])
		require.Equal(t, "Finish RFC parity", payload["goal"])
		require.Equal(t, "active", payload["status"])

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
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "slug:  ax-follow-up")
	assert.Contains(t, output, "title: AX Follow-up")
	assert.Contains(t, output, "status: active")
	assert.Contains(t, output, "goal:  Finish RFC parity")
}

func TestCommandsSprint_CmdSprintList_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/sprints", r.URL.Path)
		require.Equal(t, "active", r.URL.Query().Get("status"))
		require.Equal(t, "5", r.URL.Query().Get("limit"))

		_, _ = w.Write([]byte(`{"data":[{"id":1,"slug":"ax-follow-up","title":"AX Follow-up","status":"active"},{"id":2,"slug":"rfc-parity","title":"RFC Parity","status":"active"}],"count":2}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdSprintList(core.NewOptions(
			core.Option{Key: "status", Value: "active"},
			core.Option{Key: "limit", Value: 5},
		))
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "ax-follow-up")
	assert.Contains(t, output, "rfc-parity")
	assert.Contains(t, output, "2 sprint(s)")
}

func TestCommandsSprint_CmdSprintArchive_Bad_MissingIdentifier(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	result := subsystem.cmdSprintArchive(core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "id or slug is required")
}

func TestCommandsSprint_CmdSprintGet_Ugly_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.cmdSprintGet(core.NewOptions(core.Option{Key: "_arg", Value: "ax-follow-up"}))
	assert.False(t, result.OK)
}
