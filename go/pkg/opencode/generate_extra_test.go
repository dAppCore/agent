// SPDX-Licence-Identifier: EUPL-1.2

// ensureSandbox coverage. generate_test.go stubs ensureSandboxFn to
// exercise the session/message flow, which leaves the REAL
// Service.ensureSandbox at 0%. These tests drive it directly through the
// backed store + process seam (see opencode_lifecycle_extra_test.go for
// procBackedService / startHealthServer).
//
// ensureSandbox has three legs:
//   1. explicit sandboxID  → targetFor (Inspect; must be Running)
//   2. no id, one running  → reuse running[0].ID
//   3. no id, none running → Start (spawn)

package opencode

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
)

// TestGenerate_ensureSandbox_ExplicitRunningID — an explicit sandboxID
// that resolves to a Running record returns that id via the targetFor
// leg (no spawn).
func TestGenerate_ensureSandbox_ExplicitRunningID(t *testing.T) {
	svc := procBackedService(t, "true")

	sb := Sandbox{ID: "oc-explicit", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	r := svc.ensureSandbox("oc-explicit", "")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "oc-explicit", r.Value.(string))
}

// TestGenerate_ensureSandbox_ExplicitID_NotRunning — an explicit id whose
// record is Stopped fails the targetFor leg (status guard), returning the
// error rather than spawning.
func TestGenerate_ensureSandbox_ExplicitID_NotRunning(t *testing.T) {
	svc := procBackedService(t, "true")

	sb := Sandbox{ID: "oc-stopped", Image: "img", HostPort: 51823, Status: StatusStopped, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	r := svc.ensureSandbox("oc-stopped", "")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not running")
}

// TestGenerate_ensureSandbox_ReuseRunning — no explicit id but a Running
// sandbox exists, so ensureSandbox reuses the most-recent one (the Status
// reuse leg) without spawning.
func TestGenerate_ensureSandbox_ReuseRunning(t *testing.T) {
	svc := procBackedService(t, "true")

	sb := Sandbox{ID: "oc-reuse", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	r := svc.ensureSandbox("", "")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "oc-reuse", r.Value.(string))
}

// TestGenerate_ensureSandbox_SpawnsWhenNoneRunning — no id and nothing
// running, so ensureSandbox falls through to Start (harmless runtime +
// pinned health server) and returns the freshly-spawned id.
func TestGenerate_ensureSandbox_SpawnsWhenNoneRunning(t *testing.T) {
	svc := procBackedService(t, "true")
	_, _ = startHealthServer(t)

	r := svc.ensureSandbox("", "")
	core.AssertTrue(t, r.OK)
	core.AssertNotEmpty(t, r.Value.(string))
}
