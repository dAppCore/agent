// SPDX-Licence-Identifier: EUPL-1.2

// hostConfigMerge control-handler arm coverage. control_http_extra_test.go
// hits the success leg loosely (< 500); this file pins the two failure
// arms the happy path can't reach:
//
//   - 409 Conflict (OutcomeDenied): an existing ~/.config/opencode/
//     opencode.json already binds provider.lthn to a DIFFERENT baseURL and
//     force=false → the substrate returns HostConfigConflict, the handler
//     returns 409 + the conflict code.
//   - 500 Error (OutcomeError): the existing opencode.json is malformed
//     JSON → MergeHostConfig fails with a non-conflict error → 500.
//
// Both write a fixture under a temp HOME (newTestService) — pure
// filesystem, no docker.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// seedHostConfig writes raw bytes to ~/.config/opencode/opencode.json under
// the test's (already temp) HOME so the next merge reads it.
func seedHostConfig(t *testing.T, raw string) {
	t.Helper()
	homeR := core.UserHomeDir()
	core.AssertTrue(t, homeR.OK)
	path := core.PathJoin(homeR.Value.(string), hostConfigSubpath)
	core.AssertTrue(t, core.MkdirAll(core.PathDir(path), 0o700).OK)
	core.AssertTrue(t, core.WriteFile(path, []byte(raw), 0o600).OK)
}

// TestControl_hostConfigMerge_Conflict_409_HTTP — an existing opencode.json
// binding provider.lthn to a baseURL that differs from the default
// profile's, merged with force=false, yields 409 + the conflict code.
func TestControl_hostConfigMerge_Conflict_409_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t) // seeds the default profile (lthn → host.docker.internal:8000)

	// Pre-existing host config points lthn at a DIFFERENT baseURL.
	seedHostConfig(t, `{"provider":{"lthn":{"options":{"baseURL":"http://localhost:9999/v1"}}}}`)

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	w := httptest.NewRecorder()
	// force omitted (false) → the differing baseURL must conflict.
	e.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/host-config",
		strings.NewReader(`{"profile":"default"}`)))

	core.AssertEqual(t, http.StatusConflict, w.Code)
	core.AssertContains(t, w.Body.String(), HostConfigConflict)
	core.AssertNotEmpty(t, w.Header().Get("X-Request-Id"))
}

// TestControl_hostConfigMerge_Force_Overwrites_200_HTTP — the same
// conflicting state but with force=true resolves: the handler returns 200
// and the merged result (created=false, since the file pre-existed). Pins
// the success leg's created=false branch alongside the force path.
func TestControl_hostConfigMerge_Force_Overwrites_200_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	seedHostConfig(t, `{"provider":{"lthn":{"options":{"baseURL":"http://localhost:9999/v1"}}}}`)

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/host-config",
		strings.NewReader(`{"profile":"default","force":true}`)))

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertContains(t, w.Body.String(), `"created":false`)
}

// TestControl_hostConfigMerge_MalformedExisting_500_HTTP — an existing
// opencode.json that is not valid JSON makes MergeHostConfig fail with a
// non-conflict error, so the handler takes the 500 OutcomeError arm.
func TestControl_hostConfigMerge_MalformedExisting_500_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)

	// Garbage where JSON is expected → not a conflict, a substrate error.
	seedHostConfig(t, `this is not json at all {{{`)

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/host-config",
		strings.NewReader(`{"profile":"default"}`)))

	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
	core.AssertContains(t, w.Body.String(), "error")
}
