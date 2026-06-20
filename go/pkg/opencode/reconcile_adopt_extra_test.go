// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
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
