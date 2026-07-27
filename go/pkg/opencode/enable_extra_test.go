// SPDX-Licence-Identifier: EUPL-1.2

// Enable / Disable lifecycle coverage. Both drive the container surface
// (Enable spawns via Start when nothing is running; Disable stops via
// Stop), so they need the same process seam as opencode.go — a real
// process.Service registered into the test core with a harmless Runtime.
// See opencode_lifecycle_extra_test.go for procBackedService /
// startHealthServer.

package opencode

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
)

// TestEnable_Enable_Good_AlreadyRunning — Enable with a sandbox already
// Running sets the flag and short-circuits, returning the existing id
// WITHOUT spawning (no health server needed because Start is never
// reached). Covers the already-running branch of Enable.
func TestEnable_Enable_Good_AlreadyRunning(t *testing.T) {
	svc := procBackedService(t, "true")

	sb := Sandbox{ID: "oc-already", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	r := svc.Enable("")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "oc-already", r.Value.(string))
	core.AssertTrue(t, svc.IsEnabled())
}

// TestEnable_Enable_Good_SpawnsWhenNoneRunning — Enable with nothing
// running sets the flag and falls through to Start, spawning a sandbox
// (harmless runtime + pinned health server). Returns the new id and the
// flag persists.
func TestEnable_Enable_Good_SpawnsWhenNoneRunning(t *testing.T) {
	svc := procBackedService(t, "true")
	_, _ = startHealthServer(t)

	r := svc.Enable("")
	core.AssertTrue(t, r.OK)
	core.AssertNotEmpty(t, r.Value.(string))
	core.AssertTrue(t, svc.IsEnabled())

	// A sandbox is now Running.
	statusR := svc.Status()
	core.AssertTrue(t, statusR.OK)
	running, _ := statusR.Value.([]Sandbox)
	core.AssertEqual(t, 1, len(running))
}

// TestEnable_Disable_Good_StopsRunning — Disable clears the flag and runs
// the stop-sweep over every Running sandbox. With a "true" runtime the
// docker-rm Run succeeds and each record flips to Stopped. Covers the
// Disable stop-sweep loop body.
func TestEnable_Disable_Good_StopsRunning(t *testing.T) {
	svc := procBackedService(t, "true")
	core.AssertTrue(t, svc.setEnabled(true).OK)

	for _, id := range []string{"oc-a", "oc-b"} {
		sb := Sandbox{ID: id, Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
		core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
		svc.proxy.Set(id, "http://127.0.0.1:51823", "")
	}

	r := svc.Disable()
	core.AssertTrue(t, r.OK)
	core.AssertFalse(t, svc.IsEnabled())

	// Both flipped to Stopped; Status (Running-only) is now empty.
	statusR := svc.Status()
	core.AssertTrue(t, statusR.OK)
	running, _ := statusR.Value.([]Sandbox)
	core.AssertEqual(t, 0, len(running))
}

// TestEnable_Disable_Good_NothingRunning — Disable with no Running
// sandboxes clears the flag and returns Ok without entering the loop.
func TestEnable_Disable_Good_NothingRunning(t *testing.T) {
	svc := procBackedService(t, "true")
	core.AssertTrue(t, svc.setEnabled(true).OK)

	r := svc.Disable()
	core.AssertTrue(t, r.OK)
	core.AssertFalse(t, svc.IsEnabled())
}
