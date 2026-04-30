// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestCommandsplatform_CmdFleetRegister_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.cmdFleetRegister(core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestCommandsplatform_CmdAuthProvision_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.cmdAuthProvision(core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestCommandsplatform_CmdAuthProvision_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/auth/provision", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.AssertTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.AssertTrue(t, parseResult.OK)
		core.AssertEqual(t, []any{"10.0.0.0/8", "192.168.0.0/16"}, payload["ip_restrictions"])

		_, _ = w.Write([]byte(`{"data":{"id":7,"workspace_id":3,"name":"codex local","key":"ak_live_secret","prefix":"ak_live","permissions":["plans:read","plans:write"],"ip_restrictions":["10.0.0.0/8","192.168.0.0/16"],"rate_limit":60,"call_count":2,"expires_at":"2026-04-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdAuthProvision(core.NewOptions(
			core.Option{Key: "_arg", Value: "user-42"},
			core.Option{Key: "name", Value: "codex local"},
			core.Option{Key: "permissions", Value: "plans:read,plans:write"},
			core.Option{Key: "ip_restrictions", Value: "10.0.0.0/8,192.168.0.0/16"},
			core.Option{Key: "rate_limit", Value: 60},
			core.Option{Key: "expires_at", Value: "2026-04-01T00:00:00Z"},
		))
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "ip restrictions: 10.0.0.0/8,192.168.0.0/16")
	core.AssertContains(t, output, "prefix:      ak_live")
}

func TestCommandsplatform_CmdAuthRevoke_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"key_id":"7","revoked":true}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdAuthRevoke(core.NewOptions(core.Option{Key: "_arg", Value: "7"}))
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "revoked: 7")
}

func TestCommandsplatform_CmdFleetNodes_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":1,"agent_id":"charon","platform":"linux","models":["codex"],"status":"online"}],"total":1}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetNodes(core.NewOptions())
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "charon")
	core.AssertContains(t, output, "total: 1")
}

func TestCommandsplatform_CmdFleetEvents_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/fleet/events":
			_, _ = w.Write([]byte("data: {\"event\":\"task.assigned\",\"agent_id\":\"charon\",\"task_id\":9,\"repo\":\"core/go-io\",\"branch\":\"dev\"}\n\n"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetEvents(core.NewOptions(core.Option{Key: "_arg", Value: "charon"}))
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "event:       task.assigned")
	core.AssertContains(t, output, "agent:       charon")
	core.AssertContains(t, output, "repo:        core/go-io")
}

func TestCommandsplatform_CmdFleetEvents_Good_FallbackToTaskNext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/fleet/events":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"event stream unavailable"}`))
		case "/v1/fleet/task/next":
			_, _ = w.Write([]byte(`{"data":{"id":11,"repo":"core/go-io","branch":"dev","task":"Fix tests","template":"coding","agent_model":"codex","status":"assigned"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetEvents(core.NewOptions(core.Option{Key: "_arg", Value: "charon"}))
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "event:       task.assigned")
	core.AssertContains(t, output, "agent:       charon")
	core.AssertContains(t, output, "task id:     11")
	core.AssertContains(t, output, "repo:        core/go-io")
}

func TestCommandsplatform_CmdSyncStatus_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"agent_id":"charon","status":"online","last_push_at":"2026-03-31T08:00:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdSyncStatus(core.NewOptions(core.Option{Key: "_arg", Value: "charon"}))
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "agent:         charon")
	core.AssertContains(t, output, "status:        online")
}

func TestCommandsplatform_CmdAuthLogin_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.cmdAuthLogin(core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestCommandsplatform_CmdAuthLogin_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/auth/login", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.AssertTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.AssertTrue(t, parseResult.OK)
		core.AssertEqual(t, "654321", payload["code"])

		_, _ = w.Write([]byte(`{"data":{"key":{"id":42,"name":"charon","key":"ak_live_xyz","prefix":"ak_live","expires_at":"2027-01-01T00:00:00Z"}}}`))
	}))
	defer server.Close()

	// Pin HOME to a temp dir so we do not overwrite a real ~/.claude/brain.key.
	homeDir := t.TempDir()
	t.Setenv("CORE_HOME", homeDir)

	subsystem := testPrepWithPlatformServer(t, server, "")
	subsystem.brainURL = server.URL
	subsystem.brainKey = ""

	output := captureStdout(t, func() {
		result := subsystem.cmdAuthLogin(core.NewOptions(core.Option{Key: "_arg", Value: "654321"}))
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "logged in")
	core.AssertContains(t, output, "key prefix: ak_live")
	core.AssertContains(t, output, "saved to:")

	// Verify the key was persisted so the next dispatch authenticates.
	keyPath := core.JoinPath(homeDir, ".claude", "brain.key")
	readResult := fs.Read(keyPath)
	core.AssertTrue(t, readResult.OK)
	core.AssertEqual(t, "ak_live_xyz", core.Trim(readResult.Value.(string)))
}

func TestCommandsplatform_CmdSubscriptionDetect_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"providers":{"claude":true},"available":["claude"]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdSubscriptionDetect(core.NewOptions())
		core.AssertTrue(t, result.OK)
	})

	core.AssertContains(t, output, "available: claude")
}
