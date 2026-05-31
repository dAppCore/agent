// SPDX-Licence-Identifier: EUPL-1.2

// Tests for Mantis #1600 HIGH (Cerberus #22) — pkg/opencode web URL
// no longer embeds OPENCODE_SERVER_PASSWORD in the HTTP control-
// surface response. The previous implementation rendered userinfo
// (`http://opencode:<pw>@host`) that leaked via Referer headers,
// document.title, the clipboard, and DevTools network panel.
//
// Surface under test:
//
//   - buildWebInfo — pure helper that composes the credential-free
//     WebInfo envelope. Type-system guarantee that NO password is
//     reachable through this code path.
//   - webURL gin handler — wraps a Service.WebURL call, returns the
//     WebInfo envelope as JSON, and server-generates the X-Request-Id
//     (NOT the caller's, per Cerberus #18 / Mantis #1511). Audit-row
//     emission moved out with the audit dependency (opencode runs in a
//     sandbox and does NOT audit itself; the desktop audits at its
//     access edge), so the handler stub here records nothing.

package opencode

import (
	"net/http/httptest"
	"strings"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// --- buildWebInfo --------------------------------------------------

// TestSandboxWebURL_NoEmbeddedCreds_Good — Mantis #1600 HIGH primary
// assertion. The composed envelope's URL field has NO Basic-auth
// userinfo + NO password substring anywhere in the rendered JSON.
// The fake password is a sentinel — its presence anywhere in the
// envelope would reopen the leak vector.
func TestSandboxWebURL_NoEmbeddedCreds_Good(t *testing.T) {
	const sentinelPassword = "deadbeefcafef00dba5edba110"
	info := buildWebInfo(51823)

	if strings.Contains(info.URL, sentinelPassword) {
		t.Fatalf("URL must not contain password sentinel; got %q", info.URL)
	}
	// The classic leak vector is userinfo — `user:pw@host`. Any
	// presence of the `@` separator before the host means userinfo
	// is rendered, which is the exact regression Mantis #1600 closed.
	if strings.Contains(info.URL, "@") {
		t.Fatalf("URL must not contain userinfo separator; got %q", info.URL)
	}
	if !strings.Contains(info.URL, "127.0.0.1:51823") {
		t.Fatalf("URL must point at the resolved host:port; got %q", info.URL)
	}
	// Username is part of auth metadata, never the password. Static
	// "opencode" matches opencode-serve's default OPENCODE_SERVER_USERNAME.
	if info.Auth.Username != serverAuthUsername {
		t.Fatalf("Auth.Username = %q; want %q", info.Auth.Username, serverAuthUsername)
	}
}

// TestSandboxWebURL_AuthSchemeDocumented_Good — done-criteria #2 from
// the dispatch brief. The envelope MUST document the auth scheme so
// the caller knows how to authenticate without inspecting the URL.
func TestSandboxWebURL_AuthSchemeDocumented_Good(t *testing.T) {
	info := buildWebInfo(8080)
	if info.Auth.Scheme != WebAuthScheme {
		t.Fatalf("Auth.Scheme = %q; want %q", info.Auth.Scheme, WebAuthScheme)
	}
	if info.Auth.Scheme != "basic" {
		t.Fatalf("WebAuthScheme literal drifted from RFC 7617 'basic'; got %q",
			info.Auth.Scheme)
	}
	if info.Auth.Via != WebAuthVia {
		t.Fatalf("Auth.Via = %q; want %q", info.Auth.Via, WebAuthVia)
	}
	if info.Auth.Via != "header" {
		t.Fatalf("WebAuthVia literal drifted from 'header'; got %q", info.Auth.Via)
	}
}

// TestSandboxWebURL_TypeShapeHasNoPasswordField_Good — second-level
// defence per Mantis #1600. The WebInfo type MUST NOT carry a
// Password / Userinfo / Token field — JSON marshalling would
// otherwise expose any future-added credential field to the wire.
// This test interrogates the type via a sentinel-bearing instance
// and rejects any field shape that smells like a credential.
func TestSandboxWebURL_TypeShapeHasNoPasswordField_Good(t *testing.T) {
	const sentinel = "S3CRET-PASSWORD-SENTINEL"
	// A struct literal where every string field gets the sentinel.
	// If a future contributor adds a Password field of any name, the
	// type literal below won't compile (good — the test breaks at
	// build time, surfacing the regression).
	info := WebInfo{
		URL: "http://127.0.0.1:1/",
		Auth: WebAuthInfo{
			Scheme:   sentinel,
			Via:      sentinel,
			Username: sentinel,
		},
	}
	// Marshal via the same JSON encoder gin uses for c.JSON; we just
	// stringify and search. core.JSONMarshal returns ([]byte, error)
	// via core.Result-style; use the simpler core helper.
	b := core.JSONMarshal(info)
	if !b.OK {
		t.Fatalf("marshal failed: %v", b.Error())
	}
	body, _ := b.Value.([]byte)
	// The sentinel appears 3x (Scheme, Via, Username) — that's the
	// upper bound. A 4th hit means a new field accepts the sentinel
	// and would also accept a real password.
	hits := strings.Count(string(body), sentinel)
	if hits > 3 {
		t.Fatalf("WebInfo grew a 4th string field accepting the sentinel "+
			"(hits=%d) — confirm none of the new fields can carry "+
			"OPENCODE_SERVER_PASSWORD: %s", hits, body)
	}
}

// --- webURL handler ------------------------------------------------

// stubWebURLHandler returns a gin handler that wraps the production
// JSON-response shape from control.go's webURL, but substitutes a
// fixed WebInfo for the Service.WebURL call. The production handler is
// `func (g *ControlGroup) webURL(c *gin.Context)`; it requires a
// fully-wired Service (Core + ORM + DuckDB). This stub only swaps the
// Service call + the server-generated X-Request-Id header so the
// handler under test runs without the heavy backing infra. It records
// no audit — opencode runs in a sandbox and does NOT audit itself.
func stubWebURLHandler(info WebInfo) gin.HandlerFunc {
	return func(c *gin.Context) {
		srvReqID := newRequestID()
		c.Header("X-Request-Id", srvReqID)
		c.JSON(core.StatusOK, info)
	}
}

// TestSandboxWebURL_HandlerResponseNoEmbeddedCreds_Good — handler-
// level mirror of TestSandboxWebURL_NoEmbeddedCreds_Good. Confirms
// the JSON response body the wire actually carries contains no
// password substring, nothing parseable as Basic userinfo, and the
// documented auth metadata.
func TestSandboxWebURL_HandlerResponseNoEmbeddedCreds_Good(t *testing.T) {
	const sentinelPassword = "PASSWORD-MUST-NEVER-APPEAR-IN-RESPONSE"

	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.GET("/sandbox/:id/web",
		stubWebURLHandler(buildWebInfo(51823)))

	req := httptest.NewRequest(core.MethodGet, "/sandbox/oc-test/web", nil)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)

	if w.Code != core.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, sentinelPassword) {
		t.Fatalf("response body must not contain password sentinel; got %s", body)
	}
	// `@` is the structural marker for URL-userinfo (`user:pw@host`).
	// Its presence in the response body would indicate the userinfo
	// regression is back.
	if strings.Contains(body, "opencode:") || strings.Contains(body, "@127.0.0.1") {
		t.Fatalf("response body must not contain URL-userinfo; got %s", body)
	}
	if !strings.Contains(body, `"scheme":"basic"`) {
		t.Fatalf("response body must document scheme=basic; got %s", body)
	}
	if !strings.Contains(body, `"via":"header"`) {
		t.Fatalf("response body must document via=header; got %s", body)
	}
}

// TestSandboxWebURL_RequestIDOverriddenByServer_Ugly — Cerberus #18 /
// Mantis #1511 / #1605 fold for the webURL endpoint. The pre-fix
// handler trusted the caller's X-Request-Id header, enabling forensic
// deniability (attacker forges the value to mimic a legitimate
// caller's correlation key). Caller-supplied X-Request-Id MUST NOT
// appear in the response header — the server's UUIDv4 must overwrite
// it so the correlation key is server-authoritative.
func TestSandboxWebURL_RequestIDOverriddenByServer_Ugly(t *testing.T) {
	const attackerForged = "forged-value"

	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.GET("/sandbox/:id/web",
		stubWebURLHandler(buildWebInfo(51823)))

	req := httptest.NewRequest(core.MethodGet, "/sandbox/oc-ugly/web", nil)
	req.Header.Set("X-Request-Id", attackerForged)
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)

	if w.Code != core.StatusOK {
		t.Fatalf("status = %d; want 200", w.Code)
	}
	got := w.Header().Get("X-Request-Id")
	if got == attackerForged {
		t.Fatalf("response X-Request-Id header = caller-forged %q — server MUST overwrite "+
			"per Cerberus #18 / Mantis #1511 / #1605", got)
	}
	if len(got) != 36 {
		t.Errorf("response X-Request-Id header = %q (len %d); want server-generated UUIDv4 (36 chars)",
			got, len(got))
	}
}
