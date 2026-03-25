// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- resolveHost (extended — base cases are in paths_test.go) ---

func TestResolveHost_Good_CaseInsensitive(t *testing.T) {
	assert.Equal(t, "10.69.69.165:9101", resolveHost("Charon"))
	assert.Equal(t, "10.69.69.165:9101", resolveHost("CHARON"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("Cladius"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("LOCAL"))
}

func TestResolveHost_Good_CustomHost(t *testing.T) {
	assert.Equal(t, "my-server:9101", resolveHost("my-server"))
	assert.Equal(t, "192.168.1.100:8080", resolveHost("192.168.1.100:8080"))
}

// --- remoteToken ---

func TestRemoteToken_Good_FromEnv(t *testing.T) {
	t.Setenv("AGENT_TOKEN_CHARON", "env-token-123")
	token := remoteToken("CHARON")
	assert.Equal(t, "env-token-123", token)
}

func TestRemoteToken_Good_FallbackMCPAuth(t *testing.T) {
	t.Setenv("AGENT_TOKEN_TOKENTEST", "")
	t.Setenv("MCP_AUTH_TOKEN", "mcp-fallback")
	token := remoteToken("tokentest")
	assert.Equal(t, "mcp-fallback", token)
}

func TestRemoteToken_Good_EnvPrecedence(t *testing.T) {
	t.Setenv("AGENT_TOKEN_PRIO", "specific-token")
	t.Setenv("MCP_AUTH_TOKEN", "generic-token")
	token := remoteToken("PRIO")
	assert.Equal(t, "specific-token", token, "host-specific env should take precedence")
}
