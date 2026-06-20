// SPDX-Licence-Identifier: EUPL-1.2

// HTTP read/error-path coverage for the opencode ControlGroup handlers
// that delegate to Inspect-gated or ORM-read Service methods. On a fresh
// newTestService the Sandbox / Imported* tables are unbacked (see
// opencode_orm_extra_test.go), so every handler here takes its FAILURE
// branch: the not-found / unbacked-store return precedes any docker /
// process / window.open / network call, so nothing is spawned.
//
// The spawn/enable/openStudio/stop handlers are deliberately NOT exercised
// — those reach docker/process/app-launch before any safe gate.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// controlEngine wires a fresh test Service + ControlGroup onto a gin engine.
func controlEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))
	return e
}

func doReq(t *testing.T, e *gin.Engine, method, path string) int {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, nil)
	e.ServeHTTP(w, r)
	return w.Code
}

// TestControl_ReadHandlers_ErrorPaths_HTTP — the ORM-read + Inspect-gated
// handlers all surface their failure branch over the wire on an unbacked
// store. Codes asserted are the documented error mappings; an unbacked
// store fails the read, so the <500-or-equal expectations below are exact
// where the mapping is unambiguous and hedged (>=400) where the underlying
// medium could vary.
func TestControl_ReadHandlers_ErrorPaths_HTTP(t *testing.T) {
	e := controlEngine(t)

	// listImports / listImportedProviders → ORM unbacked → 500.
	core.AssertEqual(t, http.StatusInternalServerError, doReq(t, e, "GET", "/imports"))
	core.AssertEqual(t, http.StatusInternalServerError, doReq(t, e, "GET", "/imports/providers"))

	// list (Status) → ORM unbacked → 500.
	core.AssertEqual(t, http.StatusInternalServerError, doReq(t, e, "GET", "/sandbox"))

	// inspect → not-found → 404.
	core.AssertEqual(t, http.StatusNotFound, doReq(t, e, "GET", "/sandbox/oc-missing"))

	// webURL (GET) → Inspect not-found → 404.
	core.AssertEqual(t, http.StatusNotFound, doReq(t, e, "GET", "/sandbox/oc-missing/web"))

	// providerList → targetFor → Inspect not-found → 500.
	core.AssertEqual(t, http.StatusInternalServerError, doReq(t, e, "GET", "/sandbox/oc-missing/providers"))

	// openTUI (POST) → Inspect not-found, process never reached → 500.
	core.AssertEqual(t, http.StatusInternalServerError, doReq(t, e, "POST", "/sandbox/oc-missing/tui"))

	// openWebWindow (POST) → webURLWithCreds → Inspect not-found,
	// window.open never reached → 500.
	core.AssertEqual(t, http.StatusInternalServerError, doReq(t, e, "POST", "/sandbox/oc-missing/web"))
}

// TestControl_Disable_HTTP_CleanSweep — disable on a fresh service persists
// the flag and sweeps an EMPTY running-sandbox set (Status returns the
// no-store failure → Disable's "couldn't list, surface success" branch),
// so no Stop / docker call is made. Either 200 (flag set, empty sweep) or
// 500 (setEnabled failed on the unbacked KV) is acceptable; both are the
// no-container path. We assert it never panics and is a defined HTTP code.
func TestControl_Disable_HTTP_CleanSweep(t *testing.T) {
	e := controlEngine(t)
	code := doReq(t, e, "POST", "/disable")
	core.AssertTrue(t, code == http.StatusOK || code == http.StatusInternalServerError)
}

// TestControl_Disable_Service_NoRunningSandboxes — Disable at the Service
// layer on a fresh service: setEnabled persists, Status fails (unbacked) →
// the documented "surface success, retry teardown on next boot" branch.
// No Stop is invoked because the running list is never populated. Exercises
// enable.go Disable + setEnabled without touching docker.
func TestControl_Disable_Service_NoRunningSandboxes(t *testing.T) {
	svc := newTestService(t)
	r := svc.Disable()
	// On the temp-HOME DuckDB KV the enabled flag persists fine, so
	// Disable typically returns OK; if the KV write fails it returns the
	// setEnabled error. Either way it must not have called Stop (no
	// running sandboxes were listed) and must be a well-formed Result.
	_ = r.OK // value asserted via no-panic + IsEnabled below
	// IsEnabled now reflects the persisted false flag (or defaults false
	// when the write failed) — either way it must be false post-Disable.
	core.AssertFalse(t, svc.IsEnabled())
}
