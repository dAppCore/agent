// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestAuth_HandleAuthProvision_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/auth/provision", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "user-42", payload["oauth_user_id"])
		core.AssertEqual(t, "codex local", payload["name"])
		core.AssertEqual(t, float64(60), payload["rate_limit"])
		core.AssertEqual(t, "2026-04-01T00:00:00Z", payload["expires_at"])

		permissions, ok := payload["permissions"].([]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, []any{"plans:read", "plans:write"}, permissions)

		ipRestrictions, ok := payload["ip_restrictions"].([]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, []any{"10.0.0.0/8", "192.168.0.0/16"}, ipRestrictions)

		_, _ = w.Write([]byte(`{"data":{"id":7,"workspace_id":3,"name":"codex local","key":"ak_live_secret","prefix":"ak_live","permissions":["plans:read","plans:write"],"ip_restrictions":["10.0.0.0/8","192.168.0.0/16"],"rate_limit":60,"call_count":2,"expires_at":"2026-04-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleAuthProvision(context.Background(), core.NewOptions(
		core.Option{Key: "oauth_user_id", Value: "user-42"},
		core.Option{Key: "name", Value: "codex local"},
		core.Option{Key: "permissions", Value: "plans:read,plans:write"},
		core.Option{Key: "ip_restrictions", Value: "10.0.0.0/8,192.168.0.0/16"},
		core.Option{Key: "rate_limit", Value: 60},
		core.Option{Key: "expires_at", Value: "2026-04-01T00:00:00Z"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(AuthProvisionOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 7, output.Key.ID)
	core.AssertEqual(t, 3, output.Key.WorkspaceID)
	core.AssertEqual(t, "codex local", output.Key.Name)
	core.AssertEqual(t, "ak_live_secret", output.Key.Key)
	core.AssertEqual(t, "ak_live", output.Key.Prefix)
	core.AssertEqual(t, []string{"plans:read", "plans:write"}, output.Key.Permissions)
	core.AssertEqual(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, output.Key.IPRestrictions)
	core.AssertEqual(t, 60, output.Key.RateLimit)
	core.AssertEqual(t, 2, output.Key.CallCount)
}

func TestAuth_HandleAuthProvision_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleAuthProvision(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestAuth_HandleAuthProvision_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{broken json`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleAuthProvision(context.Background(), core.NewOptions(
		core.Option{Key: "oauth_user_id", Value: "user-42"},
	))
	core.AssertFalse(t, result.OK)
}

func TestAuth_HandleAuthRevoke_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/auth/revoke/7", r.URL.Path)
		core.AssertEqual(t, http.MethodDelete, r.Method)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":{"key_id":"7","revoked":true}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleAuthRevoke(context.Background(), core.NewOptions(
		core.Option{Key: "key_id", Value: "7"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(AuthRevokeOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "7", output.KeyID)
	core.AssertTrue(t, output.Revoked)
}

func TestAuth_HandleAuthRevoke_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleAuthRevoke(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestAuth_HandleAuthRevoke_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":true}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleAuthRevoke(context.Background(), core.NewOptions(
		core.Option{Key: "key_id", Value: "7"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(AuthRevokeOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "7", output.KeyID)
	core.AssertTrue(t, output.Revoked)
}

func TestAuth_HandleAuthLogin_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/auth/login", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		// Login is unauthenticated — pairing code is the proof.
		core.AssertEqual(t, "", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "123456", payload["code"])

		_, _ = w.Write([]byte(`{"data":{"key":{"id":11,"name":"charon","key":"ak_live_abcdef","prefix":"ak_live","permissions":["fleet:run"],"expires_at":"2027-01-01T00:00:00Z"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "")
	subsystem.brainURL = server.URL
	subsystem.brainKey = ""

	result := subsystem.handleAuthLogin(context.Background(), core.NewOptions(
		core.Option{Key: "code", Value: "123456"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(AuthLoginOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 11, output.Key.ID)
	core.AssertEqual(t, "ak_live_abcdef", output.Key.Key)
	core.AssertEqual(t, "ak_live", output.Key.Prefix)
	core.AssertEqual(t, []string{"fleet:run"}, output.Key.Permissions)
}

func TestAuth_HandleAuthLogin_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.handleAuthLogin(context.Background(), core.NewOptions())
	core.AssertFalse(t, result.OK)
}

func TestAuth_HandleAuthLogin_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server returns a malformed payload: missing key field entirely.
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "")
	subsystem.brainURL = server.URL
	subsystem.brainKey = ""

	result := subsystem.handleAuthLogin(context.Background(), core.NewOptions(
		core.Option{Key: "code", Value: "999999"},
	))
	core.AssertFalse(t, result.OK)
}
