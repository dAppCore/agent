// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// --- resolveHost (extended — base cases are in paths_test.go) ---

func TestRemote_ResolveHost_Good_CaseInsensitive(t *testing.T) {
	core.AssertEqual(t, "10.69.69.165:9101", resolveHost("Charon"))
	core.AssertEqual(t, "10.69.69.165:9101", resolveHost("CHARON"))
	core.AssertEqual(t, "127.0.0.1:9101", resolveHost("Cladius"))
	core.AssertEqual(t, "127.0.0.1:9101", resolveHost("LOCAL"))
}

func TestRemote_ResolveHost_Good_CustomHost(t *testing.T) {
	host := resolveHost("my-server")
	withPort := resolveHost("192.168.1.100:8080")
	core.AssertEqual(t, "my-server:9101", host)
	core.AssertEqual(t, "192.168.1.100:8080", withPort)
}

func TestRemote_ResolveHost_Good_TrimmedInput(t *testing.T) {
	alias := resolveHost("  charon  ")
	custom := resolveHost("  my-server  ")
	core.AssertEqual(t, "10.69.69.165:9101", alias)
	core.AssertEqual(t, "my-server:9101", custom)
}

// --- remoteToken ---

func TestRemote_RemoteToken_Good_FromEnv(t *testing.T) {
	t.Setenv("AGENT_TOKEN_CHARON", "env-token-123")
	token := remoteToken("CHARON")
	core.AssertEqual(t, "env-token-123", token)
}

func TestRemote_RemoteToken_Good_FallbackMCPAuth(t *testing.T) {
	t.Setenv("AGENT_TOKEN_TOKENTEST", "")
	t.Setenv("MCP_AUTH_TOKEN", "mcp-fallback")
	token := remoteToken("tokentest")
	core.AssertEqual(t, "mcp-fallback", token)
}

func TestRemote_RemoteToken_Good_EnvPrecedence(t *testing.T) {
	t.Setenv("AGENT_TOKEN_PRIO", "specific-token")
	t.Setenv("MCP_AUTH_TOKEN", "generic-token")
	token := remoteToken("PRIO")
	core.AssertEqual(t, "specific-token", token, "host-specific env should take precedence")
}

func TestRemote_RemoteToken_Good_TrimmedInput(t *testing.T) {
	t.Setenv("AGENT_TOKEN_CHARON", "trimmed-token")
	token := remoteToken("  charon  ")
	core.AssertEqual(t, "trimmed-token", token)
}

// --- resolveHost Bad/Ugly ---

func TestRemote_ResolveHost_Bad_Case(t *testing.T) {
	// Empty string — returns empty host with default port appended
	result := resolveHost("")
	core.AssertEqual(t, ":9101", result)
	core.AssertContains(t, result, ":9101")
}

func TestRemote_ResolveHost_Ugly_Case(t *testing.T) {
	// Unicode host name — not an alias, no colon, so default port appended
	result := resolveHost("\u00e9nchantr\u00efx")
	core.AssertEqual(t, "\u00e9nchantr\u00efx:9101", result)
	core.AssertContains(t, result, ":9101")
}

// --- remoteToken Bad/Ugly ---

func TestRemote_RemoteToken_Bad_Case(t *testing.T) {
	// Host with no matching env var and no file — returns empty
	t.Setenv("AGENT_TOKEN_NOHOST", "")
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("DIR_HOME", t.TempDir()) // no token files
	token := remoteToken("nohost")
	core.AssertEqual(t, "", token)
}

func TestRemote_RemoteToken_Ugly_Case(t *testing.T) {
	// Host name with dashes and dots — creates odd env key like AGENT_TOKEN_MY-HOST.LOCAL
	// Env lookup will use the exact uppercased key
	t.Setenv("AGENT_TOKEN_MY-HOST.LOCAL", "")
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("DIR_HOME", t.TempDir())
	token := remoteToken("my-host.local")
	core.AssertEqual(t, "", token)
}
