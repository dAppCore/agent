// SPDX-Licence-Identifier: EUPL-1.2

// Tests for Mantis #1617 HIGH (Cerberus #22 HIGH-2) — the wire response
// returned by hostConfigMerge (POST /v1/api/opencode/host-config) MUST
// NOT include the merged opencode.json bytes, because those bytes
// preserve any pre-existing user provider apiKey blocks (OpenAI,
// Anthropic, etc.) verbatim across the merge.
//
// The fix is structural: MergeHostConfigResult.Bytes carries `json:"-"`,
// so every caller that JSON-encodes the struct silently drops the field.
// These tests pin that boundary at the type-system level — if a future
// refactor swaps the tag back to `json:"bytes"`, the leak test fails
// before the change ships.

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// TestHostConfigMerge_WireResponse_NoApiKeyLeak_Bad — the wire response
// returned by JSON-encoding a MergeHostConfigResult must NOT contain the
// merged file bytes, even when those bytes carry a pre-existing user
// apiKey from a provider block the merge left untouched.
//
// Construction mirrors the real success path: the user already had an
// OpenAI apiKey configured; the merge added our lthn provider; the
// pretty-printed result preserves the OpenAI block verbatim. The wire
// response MUST sanitise that bytes field out — Option B (`json:"-"`)
// catches it at the marshaller, no per-handler view-struct required.
func TestHostConfigMerge_WireResponse_NoApiKeyLeak_Bad(t *testing.T) {
	const userApiKey = "sk-proj-VICTIM-OPENAI-KEY-DO-NOT-LEAK"
	res := MergeHostConfigResult{
		Path:    "/home/user/.config/opencode/opencode.json",
		Profile: "default",
		Bytes: `{
  "provider": {
    "openai": {
      "options": {
        "apiKey": "` + userApiKey + `"
      }
    },
    "lthn": {
      "options": {
        "baseURL": "http://localhost:8000/v1"
      }
    }
  }
}`,
		Created: false,
	}
	r := core.JSONMarshal(res)
	core.AssertTrue(t, r.OK, "JSONMarshal failed: must encode MergeHostConfigResult cleanly")
	wire, _ := r.Value.([]byte)
	if core.Contains(string(wire), userApiKey) {
		t.Fatalf("hostConfigMerge wire response leaked user apiKey "+
			"(Mantis #1617). Got: %s", string(wire))
	}
	if core.Contains(string(wire), `"bytes"`) {
		t.Fatalf("hostConfigMerge wire response carries `bytes` field "+
			"— Option B requires `json:\"-\"` to suppress at marshal. "+
			"Got: %s", string(wire))
	}
}

// TestHostConfigMerge_WireResponse_RetainsPathProfileCreated_Good — the
// frontend / CLI still need the file path, applied profile name, and
// created flag in the wire response. Suppressing Bytes must not collapse
// the rest of the success-shape.
func TestHostConfigMerge_WireResponse_RetainsPathProfileCreated_Good(t *testing.T) {
	res := MergeHostConfigResult{
		Path:    "/home/user/.config/opencode/opencode.json",
		Profile: "default",
		Bytes:   `{"provider":{}}`,
		Created: true,
	}
	r := core.JSONMarshal(res)
	core.AssertTrue(t, r.OK, "JSONMarshal failed: must encode MergeHostConfigResult cleanly")
	wire := string(r.Value.([]byte))
	core.AssertTrue(t, core.Contains(wire, `"path":"/home/user/.config/opencode/opencode.json"`),
		"wire response must retain path field")
	core.AssertTrue(t, core.Contains(wire, `"profile":"default"`),
		"wire response must retain profile field")
	core.AssertTrue(t, core.Contains(wire, `"created":true`),
		"wire response must retain created field")
}

// TestMergeHostConfigResult_BytesAvailableInProcess_Good — Option B
// drops Bytes from the WIRE shape only; in-process Go callers (audit
// suppression decisions, reconciliation, internal diff display) still
// see the field by direct struct access. The point of `json:"-"` is to
// stop accidental leakage at the marshaller boundary, not to hide the
// field from the language. Regression guard: if someone "cleans up" by
// deleting the Bytes field entirely, this test fails.
func TestMergeHostConfigResult_BytesAvailableInProcess_Good(t *testing.T) {
	const merged = `{"provider":{"lthn":{}}}`
	res := MergeHostConfigResult{
		Path:    "/tmp/oc.json",
		Profile: "default",
		Bytes:   merged,
		Created: true,
	}
	if res.Bytes != merged {
		t.Fatalf("MergeHostConfigResult.Bytes lost its value through "+
			"struct construction: got %q want %q", res.Bytes, merged)
	}
}

// TestHostConfig_nestedString_Good — a nested map path resolves to the
// terminal string value.
func TestHostConfig_nestedString_Good(t *testing.T) {
	m := map[string]any{"options": map[string]any{"baseURL": "http://localhost:8000/v1"}}
	core.AssertEqual(t, "http://localhost:8000/v1", nestedString(m, "options", "baseURL"))
}

// TestHostConfig_nestedString_Bad_MissingKey — a missing key at any step yields "".
func TestHostConfig_nestedString_Bad_MissingKey(t *testing.T) {
	m := map[string]any{"options": map[string]any{"baseURL": "x"}}
	core.AssertEqual(t, "", nestedString(m, "options", "model"))
	core.AssertEqual(t, "", nestedString(m, "provider", "baseURL"))
}

// TestHostConfig_nestedString_Ugly_NonMapOrNonString — a non-map intermediate
// or non-string terminal yields "".
func TestHostConfig_nestedString_Ugly_NonMapOrNonString(t *testing.T) {
	core.AssertEqual(t, "", nestedString(map[string]any{"options": "not-a-map"}, "options", "baseURL"))
	core.AssertEqual(t, "", nestedString(map[string]any{"port": 8000}, "port"))
}

// TestHostConfig_MergeHostConfig_Good_CreatesThenIdempotent — a first merge on
// a host with no opencode.json creates the file under ~/.config/opencode/ with
// the merged provider block; a second merge with the same profile is
// idempotent (no conflict, Created=false).
func TestHostConfig_MergeHostConfig_Good_CreatesThenIdempotent(t *testing.T) {
	svc := newTestService(t)

	r := svc.MergeHostConfig(MergeHostConfigOptions{})
	core.AssertTrue(t, r.OK)
	res := r.Value.(MergeHostConfigResult)
	core.AssertTrue(t, res.Created)
	core.AssertContains(t, res.Path, hostConfigSubpath)
	core.AssertContains(t, res.Bytes, "provider")

	r2 := svc.MergeHostConfig(MergeHostConfigOptions{})
	core.AssertTrue(t, r2.OK)
	core.AssertFalse(t, r2.Value.(MergeHostConfigResult).Created)
}
