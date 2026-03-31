// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestCommandsplatform_CmdFleetRegister_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.cmdFleetRegister(core.NewOptions())
	assert.False(t, result.OK)
}

func TestCommandsplatform_CmdAuthProvision_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.cmdAuthProvision(core.NewOptions())
	assert.False(t, result.OK)
}

func TestCommandsplatform_CmdAuthRevoke_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"key_id":"7","revoked":true}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdAuthRevoke(core.NewOptions(core.Option{Key: "_arg", Value: "7"}))
		assert.True(t, result.OK)
	})

	assert.Contains(t, output, "revoked: 7")
}

func TestCommandsplatform_CmdFleetNodes_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":1,"agent_id":"charon","platform":"linux","models":["codex"],"status":"online"}],"total":1}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetNodes(core.NewOptions())
		assert.True(t, result.OK)
	})

	assert.Contains(t, output, "charon")
	assert.Contains(t, output, "total: 1")
}

func TestCommandsplatform_CmdSyncStatus_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"agent_id":"charon","status":"online","last_push_at":"2026-03-31T08:00:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdSyncStatus(core.NewOptions(core.Option{Key: "_arg", Value: "charon"}))
		assert.True(t, result.OK)
	})

	assert.Contains(t, output, "agent:         charon")
	assert.Contains(t, output, "status:        online")
}

func TestCommandsplatform_CmdSubscriptionDetect_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"providers":{"claude":true},"available":["claude"]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdSubscriptionDetect(core.NewOptions())
		assert.True(t, result.OK)
	})

	assert.Contains(t, output, "available: claude")
}
