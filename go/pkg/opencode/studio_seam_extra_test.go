// SPDX-Licence-Identifier: EUPL-1.2

// openStudio control-handler coverage via the injectable studio presence +
// launch seams (studio.go: studioInstalled / studioOpen). Before these vars
// existed the handler was untestable: its present leg checked the real
// /Applications/OpenCode.app and would shell `open -a OpenCode`, launching a
// GUI on the test host. Overriding the two vars lets both legs run
// deterministically with no filesystem dependency and no real launch.

package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// studioEngine wires a fresh test Service + ControlGroup onto a gin engine
// and returns a POST helper that reports the status code.
func studioEngine(t *testing.T) func(method, path string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))
	return func(method, path string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, path, nil)
		e.ServeHTTP(w, r)
		return w.Code
	}
}

// TestControl_OpenStudio_Absent_404 — when the native app is not detected the
// POST /studio handler returns 404 and never reaches the launch seam.
func TestControl_OpenStudio_Absent_404(t *testing.T) {
	origInstalled, origOpen := studioInstalled, studioOpen
	defer func() { studioInstalled, studioOpen = origInstalled, origOpen }()

	studioInstalled = func() bool { return false }
	launched := false
	studioOpen = func(*Service, context.Context) core.Result {
		launched = true // must not be reached on the absent leg
		return core.Ok(true)
	}

	do := studioEngine(t)
	core.AssertEqual(t, http.StatusNotFound, do("POST", "/studio"))
	core.AssertFalse(t, launched)
}

// TestControl_OpenStudio_Present_200 — when the native app is detected and the
// launch seam succeeds, POST /studio returns 200. The launch is stubbed so no
// real `open -a OpenCode` GUI is spawned.
func TestControl_OpenStudio_Present_200(t *testing.T) {
	origInstalled, origOpen := studioInstalled, studioOpen
	defer func() { studioInstalled, studioOpen = origInstalled, origOpen }()

	studioInstalled = func() bool { return true }
	launched := false
	studioOpen = func(*Service, context.Context) core.Result {
		launched = true
		return core.Ok(true)
	}

	do := studioEngine(t)
	core.AssertEqual(t, http.StatusOK, do("POST", "/studio"))
	core.AssertTrue(t, launched)
}

// TestControl_OpenStudio_Present_LaunchFails_500 — detected but the launch seam
// fails: the handler maps the OpenStudio Fail to a 500.
func TestControl_OpenStudio_Present_LaunchFails_500(t *testing.T) {
	origInstalled, origOpen := studioInstalled, studioOpen
	defer func() { studioInstalled, studioOpen = origInstalled, origOpen }()

	studioInstalled = func() bool { return true }
	studioOpen = func(*Service, context.Context) core.Result {
		return core.Fail(core.E("opencode.OpenStudio", "launch failed", nil))
	}

	do := studioEngine(t)
	core.AssertEqual(t, http.StatusInternalServerError, do("POST", "/studio"))
}

// TestControl_Studio_Presence_GET — the GET /studio presence check reports the
// installed flag straight off the seam, both ways.
func TestControl_Studio_Presence_GET(t *testing.T) {
	origInstalled := studioInstalled
	defer func() { studioInstalled = origInstalled }()

	svc := newTestService(t)

	studioInstalled = func() bool { return true }
	core.AssertTrue(t, svc.IsStudioInstalled())

	studioInstalled = func() bool { return false }
	core.AssertFalse(t, svc.IsStudioInstalled())
}

// TestOpenStudio_Present_DelegatesToSeam — the Service.OpenStudio method (not
// just the HTTP handler) reaches studioOpen once installed, and surfaces its
// result. Covers the post-guard delegation line.
func TestOpenStudio_Present_DelegatesToSeam(t *testing.T) {
	origInstalled, origOpen := studioInstalled, studioOpen
	defer func() { studioInstalled, studioOpen = origInstalled, origOpen }()

	studioInstalled = func() bool { return true }
	studioOpen = func(*Service, context.Context) core.Result { return core.Ok("opened") }

	svc := &Service{}
	r := svc.OpenStudio()
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "opened", r.Value)
}
