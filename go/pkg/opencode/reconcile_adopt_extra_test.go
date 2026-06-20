// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
	"dappco.re/go/process"
)

// TestReconcile_adoptFromOutput_NoAdoptVerdicts — output containing only
// non-adopt rows (alien container, label mismatch, missing label) returns
// 0 and never reaches Save/proxy/Subscribe. Exercises the adoptFromOutput
// entry + the verdict filter loop with no side effects.
func TestReconcile_adoptFromOutput_NoAdoptVerdicts(t *testing.T) {
	svc := newTestService(t)
	out := "" +
		"redis\t0.0.0.0:6379->6379/tcp\twhatever\n" + // alien — skip
		"lthn-opencode-evil\t127.0.0.1:51823->4096/tcp\tattacker\n" + // label mismatch
		"lthn-opencode-legacy\t127.0.0.1:51824->4096/tcp\t\n" // missing label
	got := svc.adoptFromOutput(out, "our-install", "Basic xxx")
	core.AssertEqual(t, 0, got)
}

// TestReconcile_adoptFromOutput_AdoptRow_SaveFailsBranch — a row that
// classifies as adopt drives the Save path; on the unbacked unit store
// Save fails, so the documented "Warn + continue" branch runs and the
// adopted count stays 0. No docker is touched (Save/proxy/Subscribe are
// in-process). This pins the adoption loop body without a migrated store.
func TestReconcile_adoptFromOutput_AdoptRow_SaveFailsBranch(t *testing.T) {
	svc := newTestService(t)
	// Mirrors the classifyReconcile adopt fixture: lthn-opencode prefix +
	// matching install id + a valid host port.
	out := "lthn-opencode-oc-7f3a2b1c\t127.0.0.1:51823->4096/tcp\tinstall-a\n"
	got := svc.adoptFromOutput(out, "install-a", "Basic xxx")
	// Save fails on the unbacked store → continue → 0 recovered. The body
	// (Sandbox build, orm.Save attempt, the failed-save Warn branch) ran.
	core.AssertEqual(t, 0, got)
}

// TestReconcile_NoopEmitHooks — emitDenials / emitSignatureVerified /
// emitSignatureRejected are retained no-op verify-outcome hooks (opencode
// runs inside a sandbox and does not audit itself). Calling them must be
// inert — no panic, no side effect. Covers the stub bodies so the
// control-flow-parity hooks stay green.
func TestReconcile_NoopEmitHooks(t *testing.T) {
	svc := newTestService(t)
	// emitDenials walks unfiltered output in the desktop original; here a
	// no-op. Pass representative output to exercise the call shape.
	svc.emitDenials("lthn-opencode-x\t127.0.0.1:1->2/tcp\t\n", "our-install")

	// The signature hooks are package functions, not methods.
	emitSignatureVerified("sha256:abc", "keyid-1")
	emitSignatureRejected("sha256:def", "keyid-2", "untrusted signer", core.Fail(core.E("t", "x", nil)))

	// Reaching here without panic is the assertion.
	core.AssertTrue(t, true)
}

// TestReconcile_Good_AdoptsMatchingContainer — full Reconcile through the
// process seam. The fixture-script runtime emits one docker-ps line whose
// install_id column EQUALS svc.InstallID() (resolved first and baked in),
// a backed store lets the adopt Save succeed, and the proxy target is
// registered. Polarity trap (per review): an install_id that doesn't match
// lands on verdictLabelMismatch with recovered=0 — a green test that never
// touches the adopt path. We assert recovered==1 AND the proxy/orm landed.
func TestReconcile_Good_AdoptsMatchingContainer(t *testing.T) {
	// Build a service WITHOUT a runtime yet — resolve the install id first.
	c := core.New(core.WithOption("name", "opencode-test"))
	resetKV(t)
	r := NewService(Options{})(c)
	core.AssertTrue(t, r.OK)
	svc, _ := r.Value.(*Service)
	mountSandboxStore(t, svc)

	idR := svc.InstallID()
	core.AssertTrue(t, idR.OK)
	installID := idR.Value.(string)
	core.AssertNotEmpty(t, installID)

	// One adopt-eligible row: matching prefix, our install id, valid port.
	fixture := "lthn-opencode-oc-recon\t127.0.0.1:51999->4096/tcp\t" + installID
	scriptPath := writeRuntimeScript(t, fixture)

	// Now register a process service pointed at the script. Re-create the
	// opencode Service with the script Runtime over the SAME core so the
	// resolved install id (already persisted in the KV) is unchanged.
	r2 := NewService(Options{Runtime: scriptPath})(c)
	core.AssertTrue(t, r2.OK)
	svc2, _ := r2.Value.(*Service)
	pr := process.NewService(process.Options{})(c)
	core.AssertTrue(t, pr.OK)
	core.AssertTrue(t, c.RegisterService("process", pr.Value).OK)

	recR := svc2.Reconcile()
	core.AssertTrue(t, recR.OK)
	recovered, _ := recR.Value.(int)
	core.AssertEqual(t, 1, recovered)

	// orm row landed.
	findR := orm.Of[Sandbox](svc2.Core()).Find("oc-recon")
	core.AssertTrue(t, findR.OK)
	sb, _ := findR.Value.(Sandbox)
	core.AssertEqual(t, 51999, sb.HostPort)
	core.AssertEqual(t, StatusRunning, sb.Status)
}

// TestReconcile_NoMatchingContainers_ZeroRecovered — the script emits only
// non-adopt rows (mismatched install id), so Reconcile runs both ps passes,
// adopts nothing, and returns 0. Exercises Reconcile's body end-to-end
// (InstallID resolve, both Run calls, adoptFromOutput + emitDenials) on the
// zero-adopt branch (fireSandboxChange NOT called).
func TestReconcile_NoMatchingContainers_ZeroRecovered(t *testing.T) {
	scriptPath := writeRuntimeScript(t, "lthn-opencode-evil\t127.0.0.1:51823->4096/tcp\tattacker")
	svc := procBackedService(t, scriptPath)

	r := svc.Reconcile()
	core.AssertTrue(t, r.OK)
	recovered, _ := r.Value.(int)
	core.AssertEqual(t, 0, recovered)
}

// TestReconcile_Bad_RunFails — a "false" runtime fails the first (adopt)
// ps.Run, so Reconcile returns that failure before parsing anything.
func TestReconcile_Bad_RunFails(t *testing.T) {
	svc := procBackedService(t, "false")
	r := svc.Reconcile()
	core.AssertFalse(t, r.OK)
}

// TestReconcile_NoProc_Unavailable — runtime-less Service surfaces the
// proc()-nil leg.
func TestReconcile_NoProc_Unavailable(t *testing.T) {
	r := (&Service{}).Reconcile()
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "process service unavailable")
}
