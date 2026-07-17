// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	iofs "io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestLogin_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/device/pair", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "123456", payload["code"])

		_, _ = w.Write([]byte(`{"agent_api_key":"ak_live_test","agent_id":"charon","expires_at":"2027-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	homeDir := t.TempDir()
	t.Setenv("CORE_HOME", homeDir)

	subsystem := testPrepWithPlatformServer(t, server, "")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetLogin(core.NewOptions(core.Option{Key: "_arg", Value: "123456"}))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "logged in")
	core.AssertContains(t, output, "agent:      charon")
	core.AssertContains(t, output, "saved to:")

	keyPath := core.JoinPath(homeDir, ".core", "agent.key")
	readResult := fs.Read(keyPath)
	core.RequireTrue(t, readResult.OK)
	core.AssertEqual(t, "ak_live_test", core.Trim(readResult.Value.(string)))

	statResult := fs.Stat(keyPath)
	core.RequireTrue(t, statResult.OK)
	info := statResult.Value.(iofs.FileInfo)
	core.AssertEqual(t, iofs.FileMode(0600), info.Mode().Perm())
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
		core.AssertFalse(t, result.OK)
	})

	core.AssertContains(t, output, "error:")
	core.AssertContains(t, output, "invalid pairing code")
}

func TestLogin_Ugly(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("CORE_HOME", homeDir)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetLogin(core.NewOptions(core.Option{Key: "_arg", Value: "12ab"}))
		core.AssertFalse(t, result.OK)
	})

	core.AssertContains(t, output, "usage: core-agent login <6-digit-code>")
	core.AssertFalse(t, fs.Exists(core.JoinPath(homeDir, ".core", "agent.key")).OK)
}
