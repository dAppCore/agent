// SPDX-Licence-Identifier: EUPL-1.2

// Control-handler lifecycle coverage. The spawn / stop / enable / import
// gin handlers delegate to g.svc.Start / Stop / Enable / ImportFromHost,
// which the process seam now drives end-to-end (see
// opencode_lifecycle_extra_test.go for procBackedService /
// startHealthServer). These tests cross the HTTP boundary so the handler
// bodies — request bind, audit emit, status code, JSON envelope — run.

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

// procControlEngine wires a ControlGroup over the given (process-seamed)
// service onto a fresh gin engine and returns a do() that serves one
// request with an optional JSON body. Distinct from control_read_handlers_
// extra_test.go's controlEngine (which builds its own bare newTestService);
// the lifecycle handlers need a process-backed service, so the caller
// supplies it.
func procControlEngine(t *testing.T, svc *Service) func(method, path, body string) (int, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))
	return func(method, path, body string) (int, string) {
		w := httptest.NewRecorder()
		var req *http.Request
		if body != "" {
			req = httptest.NewRequest(method, path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(method, path, nil)
		}
		e.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}
}

// TestControl_spawn_Good_HTTP — POST /sandbox spawns via Start (harmless
// runtime + pinned health server) and returns 200 with the id + url.
func TestControl_spawn_Good_HTTP(t *testing.T) {
	svc := procBackedService(t, "true")
	_, _ = startHealthServer(t)
	do := procControlEngine(t, svc)

	code, body := do("POST", "/sandbox", `{"profile":""}`)
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, `"id"`)
	core.AssertContains(t, body, `/v1/api/sandbox/`)
}

// TestControl_spawn_Bad_HTTP — a "false" runtime fails Start, so the
// handler emits the error audit + returns 500 with the error body.
func TestControl_spawn_Bad_HTTP(t *testing.T) {
	svc := procBackedService(t, "false")
	_, _ = startHealthServer(t)
	do := procControlEngine(t, svc)

	code, body := do("POST", "/sandbox", "")
	core.AssertEqual(t, http.StatusInternalServerError, code)
	core.AssertContains(t, body, `"error"`)
}

// TestControl_stop_Good_HTTP — DELETE /sandbox/:id stops a seeded Running
// sandbox (true runtime) and returns 200 {"stopped": id}.
func TestControl_stop_Good_HTTP(t *testing.T) {
	svc := procBackedService(t, "true")
	sb := Sandbox{ID: "oc-h", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
	do := procControlEngine(t, svc)

	code, body := do("DELETE", "/sandbox/oc-h", "")
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, "oc-h")
}

// TestControl_stop_Bad_EmptyID_HTTP — a whitespace :id trims to empty, so
// Stop returns the "id is required" failure and the handler emits the
// error audit + returns 500.
func TestControl_stop_Bad_EmptyID_HTTP(t *testing.T) {
	svc := procBackedService(t, "true")
	do := procControlEngine(t, svc)

	code, body := do("DELETE", "/sandbox/%20", "")
	core.AssertEqual(t, http.StatusInternalServerError, code)
	core.AssertContains(t, body, `"error"`)
}

// TestControl_enable_Good_HTTP — POST /enable with a Running sandbox
// already present sets the flag + short-circuits, returning 200 with the
// existing id (no health server needed — Start is never reached).
func TestControl_enable_Good_HTTP(t *testing.T) {
	svc := procBackedService(t, "true")
	sb := Sandbox{ID: "oc-en", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
	do := procControlEngine(t, svc)

	code, body := do("POST", "/enable", `{"profile":"default"}`)
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, "oc-en")
	core.AssertContains(t, body, `"enabled":true`)
}

// TestControl_importFromHost_Bad_HTTP — POST /import drives ImportFromHost,
// which spawns the host `opencode serve` binary. That binary is absent on
// the test host, so StartWithOptions fails and the handler returns 500.
// Exercises the importFromHost handler error leg + ImportFromHost's spawn
// path through the process seam (proc() non-nil, allocatePort pinned).
func TestControl_importFromHost_Bad_HTTP(t *testing.T) {
	svc := procBackedService(t, "true")
	_, _ = startHealthServer(t) // pins allocatePort so we reach the spawn
	do := procControlEngine(t, svc)

	code, body := do("POST", "/import", "")
	core.AssertEqual(t, http.StatusInternalServerError, code)
	core.AssertContains(t, body, `"error"`)
}
