// SPDX-Licence-Identifier: EUPL-1.2

// Tests for the opencode HTTP control surface request-ID primitive.
//
// In the desktop original this file also verified that every
// privilege-bearing endpoint emitted exactly one audit row per call
// (Mantis #1602 / Cerberus #22). opencode runs inside a sandbox and
// does NOT audit itself — the desktop (a SASE) audits at its access
// edge — so the audit-emit verification tests + their in-memory
// recorder scaffolding moved out with the audit dependency. What
// remains here is the server-authoritative request-ID generator, which
// is still load-bearing: the handlers server-generate a UUIDv4 (NOT
// the caller's X-Request-Id, per Cerberus #18 / Mantis #1511) and echo
// it in the response header so a caller can correlate.

package opencode

import (
	"testing"
)

// --- newRequestID -------------------------------------------------

// TestNewRequestID_ShapeIsUUIDv4_Good — the helper must produce a
// canonical RFC-4122 §4.4 UUIDv4 string. The version-4 + variant-1
// bit-pattern distinguishes handler-generated IDs from caller-supplied
// junk that survived a regression.
func TestNewRequestID_ShapeIsUUIDv4_Good(t *testing.T) {
	id := newRequestID()
	if id == "" {
		t.Fatalf("newRequestID returned empty string — core.RandomBytes likely failing")
	}
	// 8-4-4-4-12 hex layout = 36 chars total.
	if len(id) != 36 {
		t.Fatalf("newRequestID length = %d; want 36 (RFC 4122 §3 canonical form): %q", len(id), id)
	}
	for i, pos := range []int{8, 13, 18, 23} {
		if id[pos] != '-' {
			t.Fatalf("newRequestID separator %d at position %d = %q; want '-': %q",
				i, pos, id[pos], id)
		}
	}
	// Version nibble — position 14 (index 14 == first hex of group 3)
	// must be '4' per §4.4.
	if id[14] != '4' {
		t.Fatalf("newRequestID version nibble = %q; want '4' (UUIDv4): %q", id[14], id)
	}
	// Variant nibble — position 19 (index 19 == first hex of group 4)
	// must be one of 8, 9, a, b (top two bits == 10).
	switch id[19] {
	case '8', '9', 'a', 'b':
	default:
		t.Fatalf("newRequestID variant nibble = %q; want 8/9/a/b (variant 1): %q", id[19], id)
	}
}

// TestNewRequestID_PerCallUnique_Good — two consecutive calls must
// return different IDs. The request-ID's correlation property depends
// on collision-free generation; if this regresses, multiple concurrent
// requests would smear into one correlation key.
func TestNewRequestID_PerCallUnique_Good(t *testing.T) {
	a := newRequestID()
	b := newRequestID()
	if a == "" || b == "" {
		t.Fatalf("newRequestID returned empty — RandomBytes broken? a=%q b=%q", a, b)
	}
	if a == b {
		t.Fatalf("newRequestID returned same value twice — broken randomness: %q", a)
	}
}
