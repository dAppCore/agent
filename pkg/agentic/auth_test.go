// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuth_HandleAuthProvision_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/auth/provision", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "user-42", payload["oauth_user_id"])
		require.Equal(t, "codex local", payload["name"])
		require.Equal(t, float64(60), payload["rate_limit"])
		require.Equal(t, "2026-04-01T00:00:00Z", payload["expires_at"])

		permissions, ok := payload["permissions"].([]any)
		require.True(t, ok)
		require.Equal(t, []any{"plans:read", "plans:write"}, permissions)

		ipRestrictions, ok := payload["ip_restrictions"].([]any)
		require.True(t, ok)
		require.Equal(t, []any{"10.0.0.0/8", "192.168.0.0/16"}, ipRestrictions)

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
	require.True(t, result.OK)

	output, ok := result.Value.(AuthProvisionOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, 7, output.Key.ID)
	assert.Equal(t, 3, output.Key.WorkspaceID)
	assert.Equal(t, "codex local", output.Key.Name)
	assert.Equal(t, "ak_live_secret", output.Key.Key)
	assert.Equal(t, "ak_live", output.Key.Prefix)
	assert.Equal(t, []string{"plans:read", "plans:write"}, output.Key.Permissions)
	assert.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, output.Key.IPRestrictions)
	assert.Equal(t, 60, output.Key.RateLimit)
	assert.Equal(t, 2, output.Key.CallCount)
}

func TestAuth_HandleAuthProvision_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleAuthProvision(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
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
	assert.False(t, result.OK)
}

func TestAuth_HandleAuthRevoke_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/auth/revoke/7", r.URL.Path)
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":{"key_id":"7","revoked":true}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleAuthRevoke(context.Background(), core.NewOptions(
		core.Option{Key: "key_id", Value: "7"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(AuthRevokeOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "7", output.KeyID)
	assert.True(t, output.Revoked)
}

func TestAuth_HandleAuthRevoke_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.handleAuthRevoke(context.Background(), core.NewOptions())
	assert.False(t, result.OK)
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
	require.True(t, result.OK)

	output, ok := result.Value.(AuthRevokeOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "7", output.KeyID)
	assert.True(t, output.Revoked)
}
