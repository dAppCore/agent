// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	iofs "io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/device/pair", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		assert.Equal(t, "123456", payload["code"])

		_, _ = w.Write([]byte(`{"agent_api_key":"ak_live_test","agent_id":"charon","expires_at":"2027-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	homeDir := t.TempDir()
	t.Setenv("CORE_HOME", homeDir)

	subsystem := testPrepWithPlatformServer(t, server, "")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetLogin(core.NewOptions(core.Option{Key: "_arg", Value: "123456"}))
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "logged in")
	assert.Contains(t, output, "agent:      charon")
	assert.Contains(t, output, "saved to:")

	keyPath := core.JoinPath(homeDir, ".core", "agent.key")
	readResult := fs.Read(keyPath)
	require.True(t, readResult.OK)
	assert.Equal(t, "ak_live_test", core.Trim(readResult.Value.(string)))

	statResult := fs.Stat(keyPath)
	require.True(t, statResult.OK)
	info := statResult.Value.(iofs.FileInfo)
	assert.Equal(t, iofs.FileMode(0600), info.Mode().Perm())
}

func TestLogin_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid pairing code"}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetLogin(core.NewOptions(core.Option{Key: "_arg", Value: "123456"}))
		require.False(t, result.OK)
	})

	assert.Contains(t, output, "error:")
	assert.Contains(t, output, "invalid pairing code")
}

func TestLogin_Ugly(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("CORE_HOME", homeDir)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetLogin(core.NewOptions(core.Option{Key: "_arg", Value: "12ab"}))
		require.False(t, result.OK)
	})

	assert.Contains(t, output, "usage: core-agent login <6-digit-code>")
	assert.False(t, fs.Exists(core.JoinPath(homeDir, ".core", "agent.key")))
}
