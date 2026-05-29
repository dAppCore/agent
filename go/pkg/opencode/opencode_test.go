// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"errors"
	"testing"
)

// --- allocatePort -------------------------------------------------

// TestAllocatePort_HappyPath_Good — a single probe that returns nil
// must succeed on attempt 1. Pins the Mantis #1604 fix's first-try
// shape so a regression to the old port-0 idiom (which would
// always-succeed without probing) is caught.
func TestAllocatePort_HappyPath_Good(t *testing.T) {
	origProbe := portProbe
	origPick := pickPortInRange
	t.Cleanup(func() {
		portProbe = origProbe
		pickPortInRange = origPick
	})
	pickPortInRange = func() int { return 50000 }
	portProbe = func(port int) error { return nil }

	r := allocatePort()
	if !r.OK {
		t.Fatalf("allocatePort failed on free port: %v", r.Error())
	}
	port, ok := r.Value.(int)
	if !ok || port != 50000 {
		t.Fatalf("allocatePort returned %v (%T); want int 50000", r.Value, r.Value)
	}
}

// TestAllocatePort_PortInRange_Good — the returned port MUST sit
// inside the IANA dynamic/private range so docker bind targets the
// ephemeral pool the OS itself uses (Cerberus #22 forward-arc note).
// Drives the real portProbe / pickPortInRange against a fresh
// allocation so the live range-math is exercised, not the mock.
func TestAllocatePort_PortInRange_Good(t *testing.T) {
	r := allocatePort()
	if !r.OK {
		t.Fatalf("allocatePort failed on real probe: %v", r.Error())
	}
	port, ok := r.Value.(int)
	if !ok {
		t.Fatalf("allocatePort returned %T; want int", r.Value)
	}
	if port < OpencodeHostPortRangeStart || port > OpencodeHostPortRangeEnd {
		t.Fatalf("port %d outside [%d, %d]",
			port, OpencodeHostPortRangeStart, OpencodeHostPortRangeEnd)
	}
}

// TestAllocatePort_RetryOnEADDRINUSE_Good — the first N probes return
// EADDRINUSE, the (N+1)th returns nil. Allocation must succeed on the
// (N+1)th port. Pins the bounded-tolerance shape that distinguishes
// this fix from a fail-fast or unbounded-loop alternative.
func TestAllocatePort_RetryOnEADDRINUSE_Good(t *testing.T) {
	origProbe := portProbe
	origPick := pickPortInRange
	t.Cleanup(func() {
		portProbe = origProbe
		pickPortInRange = origPick
	})

	calls := 0
	pickPortInRange = func() int {
		calls++
		return 50000 + calls
	}
	portProbe = func(port int) error {
		if calls <= 3 {
			return errors.New("listen tcp 127.0.0.1:X: bind: address already in use")
		}
		return nil
	}

	r := allocatePort()
	if !r.OK {
		t.Fatalf("allocatePort failed after retries: %v", r.Error())
	}
	port, _ := r.Value.(int)
	if port != 50004 {
		t.Fatalf("returned port = %d; want 50004 (succeeded on 4th attempt)", port)
	}
}

// TestAllocatePort_ExhaustedAfterMax_Bad — every probe returns
// EADDRINUSE; allocation must Fail with the typed
// "opencode.allocatePort" / "port range exhausted" shape. Pins the
// bounded-loop guarantee — without it, a hostile adversary could trap
// the allocator forever.
func TestAllocatePort_ExhaustedAfterMax_Bad(t *testing.T) {
	origProbe := portProbe
	origPick := pickPortInRange
	t.Cleanup(func() {
		portProbe = origProbe
		pickPortInRange = origPick
	})
	pickPortInRange = func() int { return 49999 }
	portProbe = func(port int) error {
		return errors.New("listen tcp 127.0.0.1:X: bind: address already in use")
	}

	r := allocatePort()
	if r.OK {
		t.Fatalf("allocatePort returned OK on all-busy; want Fail")
	}
	if msg := r.Error(); !contains(msg, "port range exhausted") {
		t.Fatalf("error %q missing 'port range exhausted' marker", msg)
	}
}
