// SPDX-License-Identifier: EUPL-1.2

// Tests for the sandbox-proxy path-traversal reject (RFC.serve.md
// §7.3.3) and the hub audit-edge dispatch (§7.3.1) wired through the
// installable AuditFunc.

package opencode

import (
	"maps"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// --- proxyPathReject ----------------------------------------------

// TestProxy_proxyPathReject_Good — clean paths pass.
func TestProxy_proxyPathReject_Good(t *testing.T) {
	for _, p := range []string{"/global/health", "/session/abc", "/", "/provider"} {
		if reason := proxyPathReject(p); reason != "" {
			t.Fatalf("clean path %q rejected: %q", p, reason)
		}
	}
}

// TestProxy_proxyPathReject_Bad — traversal segments are rejected.
func TestProxy_proxyPathReject_Bad(t *testing.T) {
	for _, p := range []string{"/../secret", "/a/../../b", "/..", "..", "/global/../etc"} {
		if proxyPathReject(p) != "path_traversal" {
			t.Fatalf("traversal path %q not rejected as path_traversal", p)
		}
	}
}

// TestProxy_proxyPathReject_Ugly — non-printable / control bytes are
// rejected.
func TestProxy_proxyPathReject_Ugly(t *testing.T) {
	for _, p := range []string{"/a\x00b", "/a\nb", "/a\x7fb"} {
		if proxyPathReject(p) != "non_printable" {
			t.Fatalf("non-printable path %q not rejected", p)
		}
	}
}

// --- proxyPathPrefix ----------------------------------------------

// TestProxy_proxyPathPrefix_Good — the leading segment only is surfaced.
func TestProxy_proxyPathPrefix_Good(t *testing.T) {
	cases := map[string]string{
		"/global/health": "/global",
		"/session/abc":   "/session",
		"/provider":      "/provider",
		"/":              "/",
	}
	for in, want := range cases {
		if got := proxyPathPrefix(in); got != want {
			t.Fatalf("proxyPathPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

// --- dispatch audit edge ------------------------------------------

// auditCapture is a test recorder for the installed AuditFunc.
type auditCapture struct {
	events []map[string]any
}

func (a *auditCapture) fn(event, scope, outcome, requestID string, meta map[string]any) {
	rec := map[string]any{"event": event, "scope": scope, "outcome": outcome}
	maps.Copy(rec, meta)
	a.events = append(a.events, rec)
}

// TestProxy_dispatch_Bad_TraversalEmitsDeniedAudit — a traversal path is
// rejected with 400 and emits a denied audit event through the installed
// sink (the hub edge).
func TestProxy_dispatch_Bad_TraversalEmitsDeniedAudit(t *testing.T) {
	cap := &auditCapture{}
	SetAuditSink(cap.fn)
	defer SetAuditSink(nil)

	g := NewSandboxProxyGroup()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	g.RegisterRoutes(engine.Group(g.BasePath()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/api/sandbox/oc-1/../secret", nil)
	engine.ServeHTTP(w, req)

	if w.Code != core.StatusBadRequest {
		t.Fatalf("traversal must be 400, got %d", w.Code)
	}
	if len(cap.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(cap.events))
	}
	ev := cap.events[0]
	if ev["event"] != EventOpencodeSandboxProxy || ev["outcome"] != outcomeDenied {
		t.Fatalf("expected denied proxy audit, got %#v", ev)
	}
	if ev["error_code"] != "path_traversal" {
		t.Fatalf("expected error_code path_traversal, got %#v", ev["error_code"])
	}
}

// TestProxy_dispatch_Ugly_UnknownSandboxNoForward — a clean path to an
// unmounted sandbox is 404 (no upstream to forward to), and still emits
// no spawn — the audit edge only records on the forward decision.
func TestProxy_dispatch_Ugly_UnknownSandboxNoForward(t *testing.T) {
	cap := &auditCapture{}
	SetAuditSink(cap.fn)
	defer SetAuditSink(nil)

	g := NewSandboxProxyGroup()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	g.RegisterRoutes(engine.Group(g.BasePath()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/api/sandbox/oc-missing/global/health", nil)
	engine.ServeHTTP(w, req)

	if w.Code != core.StatusNotFound {
		t.Fatalf("unmounted sandbox must be 404, got %d", w.Code)
	}
	// Clean path passed the reject gate, but the sandbox is absent — no
	// forward happened, so no ok-forward audit row is emitted.
	for _, ev := range cap.events {
		if ev["outcome"] == outcomeOK {
			t.Fatalf("unexpected ok audit for an unmounted sandbox: %#v", ev)
		}
	}
}
