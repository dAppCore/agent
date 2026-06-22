// SPDX-Licence-Identifier: EUPL-1.2

// Error/denied-path coverage for the profile + enable control handlers.
// The existing control_http_extra_test.go drives the happy path with
// loose `< 500` assertions; this file pins the *specific* status + body
// of each rejection arm the happy path can't reach:
//
//   - profileSave: malformed JSON → 400 denied; schema-invalid → 500 error
//   - profileGet:  unknown name → 404 not-found
//   - profileDelete: "default" (protected) → 400
//
// All run on the kv-backed newTestService — no docker, no container. The
// enable/disable handlers' Start-failure arm is deliberately NOT exercised
// here: Enable → Start would attempt a real `<runtime> run -d` against the
// host, a container side-effect a unit test must not cause. That arm stays
// on the untestable docker tail.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// newControlEngine wires a ControlGroup onto a fresh gin engine for the
// supplied Service and returns a do() that returns (status, body).
func newControlEngine(t *testing.T, svc *Service) func(method, path, body string) (int, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))
	return func(method, path, body string) (int, string) {
		w := httptest.NewRecorder()
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		e.ServeHTTP(w, r)
		return w.Code, w.Body.String()
	}
}

// TestControl_profileSave_MalformedJSON_Denied_HTTP — a body that is not
// valid JSON hits the ShouldBindJSON denied arm: 400 + an error naming the
// invalid profile, and the server still echoes an X-Request-Id.
func TestControl_profileSave_MalformedJSON_Denied_HTTP(t *testing.T) {
	svc := newTestService(t)
	do := newControlEngine(t, svc)

	code, body := do("POST", "/profile", `{"name": "broken"`) // truncated JSON
	core.AssertEqual(t, http.StatusBadRequest, code)
	core.AssertContains(t, body, "invalid profile JSON")
}

// TestControl_profileSave_SchemaInvalid_Error_HTTP — a syntactically valid
// body carrying a schema-invalid provider hits the SaveProfile-error arm:
// 500 + the schema error message.
func TestControl_profileSave_SchemaInvalid_Error_HTTP(t *testing.T) {
	svc := newTestService(t)
	do := newControlEngine(t, svc)

	// "evil" is not in profileAllowedProviderKeys → validateProfileSchema
	// fails inside SaveProfile → handler 500 arm.
	code, body := do("POST", "/profile",
		`{"name":"x","provider":{"evil":{"npm":"@attacker/sdk"}}}`)
	core.AssertEqual(t, http.StatusInternalServerError, code)
	core.AssertContains(t, body, "evil")
}

// TestControl_profileGet_NotFound_HTTP — GET for a name with no stored
// profile returns 404 + the not-found error (the !r.OK arm of profileGet).
func TestControl_profileGet_NotFound_HTTP(t *testing.T) {
	svc := newTestService(t)
	do := newControlEngine(t, svc)

	code, body := do("GET", "/profile/does-not-exist", "")
	core.AssertEqual(t, http.StatusNotFound, code)
	core.AssertContains(t, body, "not found")
}

// TestControl_profileDelete_DefaultProtected_HTTP — DELETE of "default"
// is refused: 400 + the "cannot delete the default profile" message (the
// !r.OK arm of profileDelete, reached via the DeleteProfile safety floor).
func TestControl_profileDelete_DefaultProtected_HTTP(t *testing.T) {
	svc := newTestService(t)
	do := newControlEngine(t, svc)

	code, body := do("DELETE", "/profile/default", "")
	core.AssertEqual(t, http.StatusBadRequest, code)
	core.AssertContains(t, body, "default profile")
}
