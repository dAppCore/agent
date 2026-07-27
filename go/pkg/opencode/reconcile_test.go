// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	core "dappco.re/go"
)

// In the desktop original this file also verified the adoption /
// denial audit events Reconcile emitted (via an in-memory recorder).
// opencode runs inside a sandbox and does NOT audit itself — the
// desktop (a SASE) audits reconcile outcomes at its access edge — so
// emitDenials is now a no-op and its audit-emit verification tests
// moved out with the audit dependency. The adoption-gate DECISION
// logic (classifyReconcile) is unchanged and still covered below.

// TestReconcile_parseHostPort_Good covers the canonical docker
// `Ports` column shapes — ipv4-only, ipv4+ipv6 alias, and a v6 alone.
func TestReconcile_parseHostPort_Good(t *core.T) {
	cases := []struct {
		ports string
		want  int
	}{
		{"127.0.0.1:51823->4096/tcp", 51823},
		{"0.0.0.0:51823->4096/tcp, [::]:51823->4096/tcp", 51823},
		{"[::]:51823->4096/tcp", 51823},
	}
	for _, tc := range cases {
		got := parseHostPort(tc.ports)
		if got != tc.want {
			t.Errorf("parseHostPort(%q) = %d, want %d", tc.ports, got, tc.want)
		}
	}
}

// TestReconcile_parseHostPort_Bad — malformed inputs return 0 so
// the caller can skip the row.
func TestReconcile_parseHostPort_Bad(t *core.T) {
	cases := []string{
		"",
		"no-arrow",
		"127.0.0.1->4096/tcp", // no host port
		"127.0.0.1:nope->4096/tcp",
	}
	for _, tc := range cases {
		got := parseHostPort(tc)
		if got != 0 {
			t.Errorf("parseHostPort(%q) = %d, want 0", tc, got)
		}
	}
}

// TestReconcile_parseReconcileLines_Good — well-formed docker-ps
// output is split into three columns per row; blank lines + rows
// without all three columns are dropped.
func TestReconcile_parseReconcileLines_Good(t *core.T) {
	out := "" +
		"lthn-opencode-oc-1\t127.0.0.1:51823->4096/tcp\tinstall-a\n" +
		"\n" + // blank — dropped
		"alien-container\t127.0.0.1:51824->4096/tcp\t\n" +
		"badrow\t\n" + // only 2 columns — dropped
		"lthn-opencode-oc-2\t127.0.0.1:51825->4096/tcp\tinstall-b\n"
	got := parseReconcileLines(out)
	if len(got) != 3 {
		t.Fatalf("parseReconcileLines: want 3 rows, got %d (%+v)", len(got), got)
	}
	if got[0].Name != "lthn-opencode-oc-1" || got[0].InstallID != "install-a" {
		t.Errorf("row 0: %+v", got[0])
	}
	if got[1].Name != "alien-container" || got[1].InstallID != "" {
		t.Errorf("row 1: %+v", got[1])
	}
	if got[2].Name != "lthn-opencode-oc-2" || got[2].InstallID != "install-b" {
		t.Errorf("row 2: %+v", got[2])
	}
}

// TestReconcile_classifyReconcile_Good_Adopt covers the green-path
// gate: prefix match + label match + valid port → adopt verdict.
func TestReconcile_classifyReconcile_Good_Adopt(t *core.T) {
	v := classifyReconcile(reconcileLine{
		Name:      "lthn-opencode-oc-7f3a2b1c",
		Ports:     "127.0.0.1:51823->4096/tcp",
		InstallID: "install-a",
	}, "install-a")
	if v.Status != verdictAdopt {
		t.Fatalf("Status = %q, want %q", v.Status, verdictAdopt)
	}
	if v.SandboxID != "oc-7f3a2b1c" {
		t.Errorf("SandboxID = %q, want oc-7f3a2b1c", v.SandboxID)
	}
	if v.HostPort != 51823 {
		t.Errorf("HostPort = %d, want 51823", v.HostPort)
	}
}

// TestReconcile_classifyReconcile_Bad_LabelMismatch covers the
// attack we are gating on: prefix matches + label is present but
// belongs to a DIFFERENT install. Must verdictLabelMismatch (NOT
// adopt) even if port is valid.
func TestReconcile_classifyReconcile_Bad_LabelMismatch(t *core.T) {
	v := classifyReconcile(reconcileLine{
		Name:      "lthn-opencode-evil",
		Ports:     "127.0.0.1:51823->4096/tcp",
		InstallID: "attacker-install",
	}, "our-install")
	if v.Status != verdictLabelMismatch {
		t.Fatalf("Status = %q, want %q", v.Status, verdictLabelMismatch)
	}
}

// TestReconcile_classifyReconcile_Bad_MissingLabel covers the
// pre-#1599 migration case: prefix matches but no label exists.
// Must verdictMissingLabel (NOT adopt) — user has to repair or
// docker-rm the orphan.
func TestReconcile_classifyReconcile_Bad_MissingLabel(t *core.T) {
	v := classifyReconcile(reconcileLine{
		Name:      "lthn-opencode-legacy",
		Ports:     "127.0.0.1:51823->4096/tcp",
		InstallID: "",
	}, "our-install")
	if v.Status != verdictMissingLabel {
		t.Fatalf("Status = %q, want %q", v.Status, verdictMissingLabel)
	}
}

// TestReconcile_classifyReconcile_Ugly_PrefixMissAlienContainer —
// alien container (no prefix) is skipped silently even if it carries
// some other install_id. Reconcile is not a global container audit;
// the prefix is the outer scope.
func TestReconcile_classifyReconcile_Ugly_PrefixMissAlienContainer(t *core.T) {
	v := classifyReconcile(reconcileLine{
		Name:      "redis",
		Ports:     "0.0.0.0:6379->6379/tcp",
		InstallID: "anything",
	}, "our-install")
	if v.Status != verdictSkip {
		t.Fatalf("Status = %q, want %q", v.Status, verdictSkip)
	}
}

// TestReconcile_classifyReconcile_Ugly_BadPort — label matches but
// Ports column is unparseable. Gate passes but bad_port verdict
// stops adoption (caller has no host:port to register with the
// reverse proxy).
func TestReconcile_classifyReconcile_Ugly_BadPort(t *core.T) {
	v := classifyReconcile(reconcileLine{
		Name:      "lthn-opencode-oc-x",
		Ports:     "no-arrow-here",
		InstallID: "our-install",
	}, "our-install")
	if v.Status != verdictBadPort {
		t.Fatalf("Status = %q, want %q", v.Status, verdictBadPort)
	}
}

// TestReconcile_InstallIDLabel_Constant — the docker label key is
// a wire-contract value (must match what spawn writes and what
// `docker ps --filter label=...` consumes). A bare rename of the
// constant without updating the spawn site would break the gate;
// this test pins the canonical string so the change shows up as a
// failing diff rather than a silent regression.
func TestReconcile_InstallIDLabel_Constant(t *core.T) {
	if InstallIDLabel != "lthn.opencode.install_id" {
		t.Fatalf("InstallIDLabel = %q, want lthn.opencode.install_id", InstallIDLabel)
	}
}
