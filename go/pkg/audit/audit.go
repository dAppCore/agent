// SPDX-License-Identifier: EUPL-1.2

// Package audit is the hub's audit edge. RFC.serve.md §7.3.1 makes the
// core-agent hub the new audit edge for opencode lifecycle + brain
// mutations: opencode's own emit hooks are deliberate no-ops because
// "the desktop (a SASE) audits at its access edge, not inside the
// sandbox". The hub deletes that desktop edge, so unless the hub
// becomes the new edge, audit vanishes. This package is that edge — a
// JSONL append sink that records the privilege-bearing decision flow
// (event + outcome + sandbox_id + path-prefix) and NEVER the request
// bytes or any credential material.
//
// Usage example:
//
//	sink := audit.NewFileSink(c.Fs(), "/var/lib/core-agent/audit.jsonl")
//	sink.Emit(audit.Event{
//	    Event:     "opencode.sandbox.spawn",
//	    Outcome:   "ok",
//	    RequestID: "8f3a-...",
//	    SandboxID: "oc-7f3a2b1c",
//	    Meta:      map[string]any{"profile": "default"},
//	})
package audit

import (
	core "dappco.re/go"
)

// Event is one audited decision on the hub's privilege-bearing surface.
// The shape is deliberately narrow: the fields below are the only data
// that may be recorded. Request bodies, opencode-serve credentials,
// provider apiKeys, and host-config bytes are structurally absent — the
// emit-sites cannot reach them and Sanitise drops credential-shaped Meta
// keys defensively.
//
// Usage example:
//
//	ev := audit.Event{Event: "opencode.sandbox.stop", Outcome: "ok", SandboxID: "oc-1"}
type Event struct {
	// Event is the reserved event-name literal (e.g.
	// "opencode.sandbox.spawn"). Defined by the emitting surface.
	Event string `json:"event"`

	// Outcome is one of "ok", "denied", "error".
	Outcome string `json:"outcome"`

	// RequestID is the server-authoritative correlation id (never the
	// caller-supplied X-Request-Id — that is dropped upstream per
	// Cerberus #18 / Mantis #1511).
	RequestID string `json:"request_id,omitempty"`

	// SandboxID is the opencode sandbox the decision concerns, when the
	// event is sandbox-scoped.
	SandboxID string `json:"sandbox_id,omitempty"`

	// PathPrefix is the forwarded path's leading segment for proxy
	// events — never the full path (which can carry session ids /
	// query material), only the prefix that identifies the upstream
	// surface (e.g. "/global", "/session").
	PathPrefix string `json:"path_prefix,omitempty"`

	// Meta carries event-specific scalar context (profile name, error
	// code, counts). Sanitise drops any credential-shaped key before
	// the event is written.
	Meta map[string]any `json:"meta,omitempty"`

	// At is the RFC3339Nano timestamp; filled by the sink when zero.
	At string `json:"at"`
}

// Sink receives audited events. Implementations must be safe for
// concurrent Emit calls — the hub's HTTP handlers run on many
// goroutines.
//
// Usage example:
//
//	var s audit.Sink = audit.NewFileSink(fs, path)
//	s.Emit(audit.Event{Event: "opencode.upgrade", Outcome: "ok"})
type Sink interface {
	Emit(ev Event)
}

// credentialKeySubstrings are Meta key fragments that must never reach
// the audit log. A key containing any of these (case-insensitive) is
// dropped by Sanitise, defence-in-depth behind the structural guarantee
// that the emit-sites cannot reach credential bytes.
var credentialKeySubstrings = []string{
	"password", "secret", "token", "apikey", "api_key",
	"bearer", "authorization", "credential", "privatekey", "private_key",
	"bytes", "payload",
}

// Sanitise returns a copy of meta with credential-shaped keys removed.
// Defensive: the opencode emit-sites already structurally cannot carry
// credential bytes, but Sanitise guarantees the property regardless of
// who calls Emit.
//
// Usage example:
//
//	clean := audit.Sanitise(map[string]any{"profile": "x", "token": "sk-..."})
//	// clean == map[string]any{"profile": "x"}
func Sanitise(meta map[string]any) map[string]any {
	if len(meta) == 0 {
		return nil
	}
	out := make(map[string]any, len(meta))
	for k, v := range meta {
		if isCredentialKey(k) {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// isCredentialKey reports whether a Meta key looks credential-bearing.
//
//	isCredentialKey("profile")   // false
//	isCredentialKey("API_TOKEN") // true
func isCredentialKey(k string) bool {
	lower := core.Lower(k)
	for _, frag := range credentialKeySubstrings {
		if core.Contains(lower, frag) {
			return true
		}
	}
	return false
}
