// SPDX-Licence-Identifier: EUPL-1.2

// Success-delegation coverage for the Wails lifecycle wrappers (WStart /
// WStop / WEnable) and the control enable error arm. These delegate to
// Service.Start / Stop / Enable, which the process seam drives end-to-end
// with a HARMLESS runtime ("true" / "false") + a pinned health server —
// never a real docker container. See opencode_lifecycle_extra_test.go for
// procBackedService / startHealthServer.
//
// Each test asserts the real lifecycle side-effect (a Running row appears /
// flips to Stopped / the enabled flag persists), not a bare OK.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
	"github.com/gin-gonic/gin"
)

// TestWStart_Good_SpawnsRunningSandbox — WStart on a bound, process-seamed
// service reaches Service.Start (harmless runtime + pinned health server),
// returns the new id, and a Running Sandbox row materialises. Drives the
// WStart success-delegation line the nil-guard test cannot.
func TestWStart_Good_SpawnsRunningSandbox(t *testing.T) {
	svc := procBackedService(t, "true")
	_, port := startHealthServer(t)
	w := NewWailsService(svc)

	r := w.WStart("")
	core.AssertTrue(t, r.OK)
	id, _ := r.Value.(string)
	core.AssertNotEmpty(t, id)

	// The lifecycle side-effect: a Running row at the pinned port.
	findR := orm.Of[Sandbox](svc.Core()).Find(id)
	core.AssertTrue(t, findR.OK)
	sb, _ := findR.Value.(Sandbox)
	core.AssertEqual(t, StatusRunning, sb.Status)
	core.AssertEqual(t, port, sb.HostPort)
}

// TestWStop_Good_FlipsToStopped — WStop on a seeded Running sandbox reaches
// Service.Stop (true runtime → docker-rm succeeds), and the row flips to
// Stopped. Drives the WStop success-delegation line.
func TestWStop_Good_FlipsToStopped(t *testing.T) {
	svc := procBackedService(t, "true")
	sb := Sandbox{ID: "oc-wstop", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
	w := NewWailsService(svc)

	core.AssertTrue(t, w.WStop("oc-wstop").OK)

	findR := orm.Of[Sandbox](svc.Core()).Find("oc-wstop")
	core.AssertTrue(t, findR.OK)
	got, _ := findR.Value.(Sandbox)
	core.AssertEqual(t, StatusStopped, got.Status)
}

// TestWEnable_Good_AlreadyRunning_PersistsFlag — WEnable with a sandbox
// already Running sets the enabled flag and short-circuits to the existing
// id (Start is never reached, so no health server needed). Drives the
// WEnable success-delegation line + asserts the persisted flag side-effect.
func TestWEnable_Good_AlreadyRunning_PersistsFlag(t *testing.T) {
	svc := procBackedService(t, "true")
	sb := Sandbox{ID: "oc-wen", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
	w := NewWailsService(svc)

	r := w.WEnable("")
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "oc-wen", r.Value.(string))

	// The flag persisted as a real side-effect of the wrapper's delegate.
	core.AssertTrue(t, svc.IsEnabled())
}

// TestControl_enable_StartFailure_Error_HTTP — POST /enable with an empty
// body (profile defaults to DefaultProfile) and NO running sandbox: the
// flag persists, then Enable → Start fails on the "false" runtime, so the
// handler takes its 500 error arm. Covers both the profile-default arm and
// the enable error arm. Safe: "false" is /bin/false, not docker.
func TestControl_enable_StartFailure_Error_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := procBackedService(t, "false")
	_, _ = startHealthServer(t) // pins allocatePort so Start reaches ps.Run

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	w := httptest.NewRecorder()
	// Empty JSON object → req.Profile == "" → handler defaults to
	// DefaultProfile (the 451-453 arm) before Enable fails.
	e.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/enable", strings.NewReader(`{}`)))

	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
	core.AssertContains(t, w.Body.String(), "error")
	// setEnabled ran before Start failed — flag persisted true.
	core.AssertTrue(t, svc.IsEnabled())
}

// TestInspect_Bad_EmptyID — Inspect with a whitespace id trims to empty and
// fails on the id-required guard before any ORM read. Covers the Inspect
// empty-id arm distinctly from the not-found / success arms.
func TestInspect_Bad_EmptyID(t *testing.T) {
	svc := newTestService(t)
	r := svc.Inspect("   ")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "id is required")
}
