// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- resolveHost (extended — base cases are in paths_test.go) ---

func TestRemote_ResolveHost_Good_CaseInsensitive(t *testing.T) {
	assert.Equal(t, "10.69.69.165:9101", resolveHost("Charon"))
	assert.Equal(t, "10.69.69.165:9101", resolveHost("CHARON"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("Cladius"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("LOCAL"))
}

func TestRemote_ResolveHost_Good_CustomHost(t *testing.T) {
	assert.Equal(t, "my-server:9101", resolveHost("my-server"))
	assert.Equal(t, "192.168.1.100:8080", resolveHost("192.168.1.100:8080"))
}

// --- remoteToken ---

func TestRemote_RemoteToken_Good_FromEnv(t *testing.T) {
	t.Setenv("AGENT_TOKEN_CHARON", "env-token-123")
	token := remoteToken("CHARON")
	assert.Equal(t, "env-token-123", token)
}

func TestRemote_RemoteToken_Good_FallbackMCPAuth(t *testing.T) {
	t.Setenv("AGENT_TOKEN_TOKENTEST", "")
	t.Setenv("MCP_AUTH_TOKEN", "mcp-fallback")
	token := remoteToken("tokentest")
	assert.Equal(t, "mcp-fallback", token)
}

func TestRemote_RemoteToken_Good_EnvPrecedence(t *testing.T) {
	t.Setenv("AGENT_TOKEN_PRIO", "specific-token")
	t.Setenv("MCP_AUTH_TOKEN", "generic-token")
	token := remoteToken("PRIO")
	assert.Equal(t, "specific-token", token, "host-specific env should take precedence")
}

// --- resolveHost Bad/Ugly ---

func TestRemote_ResolveHost_Bad(t *testing.T) {
	// Empty string — returns empty host with default port appended
	result := resolveHost("")
	assert.Equal(t, ":9101", result)
}

func TestRemote_ResolveHost_Ugly(t *testing.T) {
	// Unicode host name — not an alias, no colon, so default port appended
	result := resolveHost("\u00e9nchantr\u00efx")
	assert.Equal(t, "\u00e9nchantr\u00efx:9101", result)
}

// --- remoteToken Bad/Ugly ---

func TestRemote_RemoteToken_Bad(t *testing.T) {
	// Host with no matching env var and no file — returns empty
	t.Setenv("AGENT_TOKEN_NOHOST", "")
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("DIR_HOME", t.TempDir()) // no token files
	token := remoteToken("nohost")
	assert.Equal(t, "", token)
}

func TestRemote_RemoteToken_Ugly(t *testing.T) {
	// Host name with dashes and dots — creates odd env key like AGENT_TOKEN_MY-HOST.LOCAL
	// Env lookup will use the exact uppercased key
	t.Setenv("AGENT_TOKEN_MY-HOST.LOCAL", "")
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("DIR_HOME", t.TempDir())
	token := remoteToken("my-host.local")
	assert.Equal(t, "", token)
}
