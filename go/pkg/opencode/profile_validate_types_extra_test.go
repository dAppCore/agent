// SPDX-Licence-Identifier: EUPL-1.2

// Type-shape rejection coverage for the closed-schema profile validator
// (Mantis #1603 HIGH / Cerberus #22). The existing profile_test.go suite
// covers the unknown-key and shell-metachar arms; this file targets the
// *type-assertion* reject branches — a caller who sends a string where a
// map is expected, a number where a string is expected, an over-long
// identifier, etc. Each must surface ProfileInvalidSchema with the
// offending key path, not a panic.
//
// Pure validator — no DuckDB, no process. validateProfileSchema is the
// single boundary all callers (HTTP, CLI, import) inherit.

package opencode

import (
	"strings"
	"testing"

	core "dappco.re/go"
)

// assertSchemaReject is the shared expectation: err is non-nil, coded
// ProfileInvalidSchema, and names the supplied path fragment.
func assertSchemaReject(t *testing.T, err error, wantPathFragment string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected ProfileInvalidSchema reject naming %q, got nil", wantPathFragment)
	}
	if got := core.Fail(err).Code(); got != ProfileInvalidSchema {
		t.Fatalf("error code = %q; want %q (err: %v)", got, ProfileInvalidSchema, err)
	}
	if wantPathFragment != "" && !strings.Contains(err.Error(), wantPathFragment) {
		t.Errorf("error %q should name path fragment %q", err.Error(), wantPathFragment)
	}
}

// TestProfileValidate_ProviderNotMap_Bad — a provider entry whose value is
// a scalar (not a map) rejects at the "must be a map" arm. Defends against
// `provider: { openai: "anything" }`.
func TestProfileValidate_ProviderNotMap_Bad(t *testing.T) {
	p := Profile{
		Name:     "tight",
		Provider: map[string]any{"openai": "not-a-map"},
	}
	assertSchemaReject(t, validateProfileSchema(p), "provider.openai must be a map")
}

// TestProfileValidate_ProviderOptionsNotMap_Bad — `options` present but a
// scalar rejects at the options "must be a map" arm.
func TestProfileValidate_ProviderOptionsNotMap_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		Provider: map[string]any{
			"openai": map[string]any{"options": "not-a-map"},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "provider.openai.options must be a map")
}

// TestProfileValidate_ProviderBaseURLNotString_Bad — baseURL present but a
// number rejects at the baseURL "must be a string" arm (before the URL
// shape check).
func TestProfileValidate_ProviderBaseURLNotString_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		Provider: map[string]any{
			"openai": map[string]any{
				"options": map[string]any{"baseURL": float64(8000)},
			},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "provider.openai.options.baseURL must be a string")
}

// TestProfileValidate_ProviderOptionsGenericKey_Good — a non-baseURL
// options key (apiKey) routes through validateProfileAnyValue and a clean
// string value validates. Pins the generic-value continue arm in
// validateProfileProviderOptions.
func TestProfileValidate_ProviderOptionsGenericKey_Good(t *testing.T) {
	p := Profile{
		Name: "tight",
		Provider: map[string]any{
			"openai": map[string]any{
				"options": map[string]any{"apiKey": "sk-clean-value"},
			},
		},
	}
	if err := validateProfileSchema(p); err != nil {
		t.Fatalf("clean apiKey options value should validate, got: %v", err)
	}
}

// TestProfileValidate_ProviderGenericKeyRejected_Bad — a non-options
// provider sub-key (models) carrying an over-long string propagates the
// reject out through validateProfileAnyValue. Pins the provider-level
// generic-value arm.
func TestProfileValidate_ProviderGenericKeyRejected_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		Provider: map[string]any{
			"openai": map[string]any{"name": strings.Repeat("x", profileMaxStringLen+1)},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "provider.openai.name")
}

// TestProfileValidate_MCPNotMap_Bad — an MCP record whose value is a scalar
// rejects at the mcp "must be a map" arm.
func TestProfileValidate_MCPNotMap_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP:  map[string]any{"server": "not-a-map"},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server must be a map")
}

// TestProfileValidate_MCPCommandNotString_Bad — command present but a
// number rejects at the command "must be a string" arm.
func TestProfileValidate_MCPCommandNotString_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{"command": float64(42)},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server.command must be a string")
}

// TestProfileValidate_MCPArgsNotArray_Bad — args present but a scalar
// rejects at the args "must be an array" arm.
func TestProfileValidate_MCPArgsNotArray_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{
				"command": "mcp-fs",
				"args":    "should-be-array",
			},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server.args must be an array")
}

// TestProfileValidate_MCPArgsItemNotString_Bad — an args array carrying a
// non-string element rejects at the per-item "must be a string" arm.
func TestProfileValidate_MCPArgsItemNotString_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{
				"command": "mcp-fs",
				"args":    []any{"--root", float64(7)},
			},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server.args[1] must be a string")
}

// TestProfileValidate_MCPURLNotString_Bad — url present but a number
// rejects at the url "must be a string" arm.
func TestProfileValidate_MCPURLNotString_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{"url": float64(9000)},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server.url must be a string")
}

// TestProfileValidate_MCPURLNonHTTP_Bad — url is a string but not http(s)
// rejects at the URL-shape arm.
func TestProfileValidate_MCPURLNonHTTP_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{"url": "file:///etc/passwd"},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server.url is not a valid")
}

// TestProfileValidate_MCPCleanURL_Good — a clean https url-shape MCP record
// validates (the url branch's happy path, with no command set so the
// both-declared guard does not fire).
func TestProfileValidate_MCPCleanURL_Good(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"http-server": map[string]any{
				"url":     "https://mcp.example.com/sse",
				"enabled": true,
			},
		},
	}
	if err := validateProfileSchema(p); err != nil {
		t.Fatalf("clean url MCP record should validate, got: %v", err)
	}
}

// TestProfileValidate_MCPGenericKeyRejected_Bad — an MCP `env` key (the
// default arm → validateProfileAnyValue) carrying a NUL byte propagates the
// reject. Pins the MCP default-key generic-value arm.
func TestProfileValidate_MCPGenericKeyRejected_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{
				"command": "mcp-fs",
				"env":     map[string]any{"TOKEN": "abc\x00def"},
			},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server.env.TOKEN")
}

// TestProfileValidate_MCPIdentifierEmpty_Bad — an empty MCP server id
// rejects at the identifier non-empty arm.
func TestProfileValidate_MCPIdentifierEmpty_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"": map[string]any{"command": "mcp-fs"},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp identifier must be non-empty")
}

// TestProfileValidate_IdentifierOverLong_Bad — an MCP id over 64 bytes
// rejects at the identifier length arm.
func TestProfileValidate_IdentifierOverLong_Bad(t *testing.T) {
	longID := strings.Repeat("a", 65)
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			longID: map[string]any{"command": "mcp-fs"},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "exceeds 64 bytes")
}

// TestProfileValidate_AgentNotMap_Bad — an agent entry whose value is a
// scalar rejects at the agent "must be a map" arm.
func TestProfileValidate_AgentNotMap_Bad(t *testing.T) {
	p := Profile{
		Name:  "tight",
		Agent: map[string]any{"build": "not-a-map"},
	}
	assertSchemaReject(t, validateProfileSchema(p), "agent.build must be a map")
}

// TestProfileValidate_PermissionNotString_Bad — a permission value that is
// not a string (e.g. a bool) rejects at the permission "must be ... (got
// non-string)" arm, distinct from the unknown-value arm.
func TestProfileValidate_PermissionNotString_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		Permission: map[string]any{
			"bash": true, // not a string
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "got non-string")
}

// TestProfileValidate_IdentifierValidPunctuation_Good — dot, dash, and
// underscore are all legal identifier characters; a 64-byte id of exactly
// the boundary length validates. Pins the identifier accept path.
func TestProfileValidate_IdentifierValidPunctuation_Good(t *testing.T) {
	id := "fs-server.v2_alpha"
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			id: map[string]any{"url": "https://example.com/mcp"},
		},
	}
	if err := validateProfileSchema(p); err != nil {
		t.Fatalf("valid-punctuation identifier %q should validate, got: %v", id, err)
	}
}

// TestProfileValidate_ProviderOptionsGenericValueRejected_Bad — a
// non-baseURL options value (apiKey) carrying a NUL byte propagates the
// reject out of validateProfileAnyValue. Pins the generic-value reject arm
// inside validateProfileProviderOptions (distinct from the clean-value
// accept covered above).
func TestProfileValidate_ProviderOptionsGenericValueRejected_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		Provider: map[string]any{
			"openai": map[string]any{
				"options": map[string]any{"apiKey": "sk-evil\x00key"},
			},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "provider.openai.options.apiKey")
}

// TestProfileValidate_MCPUnknownKeyRejected_Bad — an MCP record carrying a
// key outside profileAllowedMCPKeys rejects at the unknown-mcp-key arm.
func TestProfileValidate_MCPUnknownKeyRejected_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{
				"command": "mcp-fs",
				"hook":    "@attacker/inject", // unknown key
			},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "unknown mcp key: mcp.server.hook")
}

// TestProfileValidate_AnyValueSliceItemRejected_Bad — an agent value that
// is an array carrying a bad string element propagates the reject through
// the []any recursion's per-item check. Pins the slice-item error arm in
// validateProfileAnyValue (line 648) reached via SaveProfile-shaped input.
func TestProfileValidate_AnyValueSliceItemRejected_Bad(t *testing.T) {
	p := Profile{
		Name: "tight",
		Agent: map[string]any{
			"build": map[string]any{
				"tools": []any{"bash", "edit\x00evil"},
			},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "agent.build.tools[1]")
}

// TestProfileValidate_MCPCommandOverLong_Bad — an MCP command exceeding
// profileMaxStringLen rejects through validateProfileNoShellMetachars'
// pre-check delegation to validateProfileStringValue (line 684), the
// length arm rather than the metachar arm.
func TestProfileValidate_MCPCommandOverLong_Bad(t *testing.T) {
	// Over-long but metachar-free so the reject MUST come from the
	// length pre-check inside validateProfileNoShellMetachars.
	cmd := strings.Repeat("a", profileMaxStringLen+1)
	p := Profile{
		Name: "tight",
		MCP: map[string]any{
			"server": map[string]any{"command": cmd},
		},
	}
	assertSchemaReject(t, validateProfileSchema(p), "mcp.server.command exceeds max length")
}
