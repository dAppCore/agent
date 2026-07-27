// SPDX-Licence-Identifier: EUPL-1.2

// Backed-store success paths for the running-sandbox read surface:
// Inspect / targetFor / WebURL / webURLWithCreds all resolve through the
// orm Sandbox record and (for the URL builders) pure URL construction —
// no docker, no container network. Reuses newBackedService from
// import_store_extra_test.go.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
	"github.com/gin-gonic/gin"
)

// seedRunningSandbox saves a Running sandbox row so the Inspect-gated read
// surface resolves it. No container is created — the row is metadata only.
func seedRunningSandbox(t *testing.T, svc *Service, id string, port int) {
	t.Helper()
	sb := Sandbox{ID: id, Image: "img", HostPort: port, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
}

// TestWeb_Backed_RunningSandbox_ReadSurface — Inspect/targetFor/WebURL/
// webURLWithCreds all succeed against a Running sandbox row. These build
// URLs / read the record only; no docker or container HTTP is touched.
func TestWeb_Backed_RunningSandbox_ReadSurface(t *testing.T) {
	svc := newBackedService(t)
	seedRunningSandbox(t, svc, "oc-run", 51823)

	// Inspect success — returns the Sandbox.
	ir := svc.Inspect("oc-run")
	core.AssertTrue(t, ir.OK)
	sb, _ := ir.Value.(Sandbox)
	core.AssertEqual(t, "oc-run", sb.ID)

	// targetFor success — resolves the in-process reverse-proxy target.
	target, tr := svc.targetFor("oc-run")
	core.AssertTrue(t, tr.OK)
	core.AssertContains(t, target, "51823")

	// WebURL success — credential-free WebInfo.
	wr := svc.WebURL("oc-run")
	core.AssertTrue(t, wr.OK)
	info, _ := wr.Value.(WebInfo)
	core.AssertContains(t, info.URL, "51823")
	core.AssertNotEmpty(t, info.Auth.Scheme)

	// webURLWithCreds success — the in-process userinfo URL (never on the
	// wire; this is the GUI-only path). Builds a URL with the mint
	// password from ServerPassword.
	cr := svc.webURLWithCreds("oc-run")
	core.AssertTrue(t, cr.OK)
	withCreds, _ := cr.Value.(string)
	core.AssertContains(t, withCreds, "51823")
	core.AssertContains(t, withCreds, serverAuthUsername)
}

// TestWeb_Backed_NotRunningSandbox_Guards — a Stopped sandbox row makes
// WebURL/webURLWithCreds/targetFor fail on the "not running" guard
// (distinct from the not-found branch the unbacked test covers).
func TestWeb_Backed_NotRunningSandbox_Guards(t *testing.T) {
	svc := newBackedService(t)
	sb := Sandbox{ID: "oc-stopped", Image: "img", HostPort: 51999, Status: StatusStopped, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	core.AssertFalse(t, svc.WebURL("oc-stopped").OK)
	core.AssertFalse(t, svc.webURLWithCreds("oc-stopped").OK)
	if _, tr := svc.targetFor("oc-stopped"); tr.OK {
		t.Fatal("targetFor should fail for a stopped sandbox")
	}
}

// TestControl_WebURL_HTTP_Success — the webURL handler returns 200 with the
// WebInfo body (and the X-Request-Id header) for a Running sandbox.
func TestControl_WebURL_HTTP_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newBackedService(t)
	seedRunningSandbox(t, svc, "oc-web", 51824)

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest("GET", "/sandbox/oc-web/web", nil))

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertContains(t, w.Body.String(), "51824")
	core.AssertNotEmpty(t, w.Header().Get("X-Request-Id"))
}
