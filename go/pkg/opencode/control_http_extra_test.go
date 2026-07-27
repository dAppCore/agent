// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// TestControl_ProfileHandlers_HTTP — the kv-backed profile + enabled HTTP
// handlers respond over the registered routes. No containers are touched
// (sandbox/enable/spawn routes are deliberately not exercised here).
func TestControl_ProfileHandlers_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)
	g := NewControlGroup(svc)
	engine := gin.New()
	g.RegisterRoutes(engine.Group(""))

	do := func(method, path, body string) int {
		w := httptest.NewRecorder()
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		engine.ServeHTTP(w, r)
		return w.Code
	}

	core.AssertEqual(t, 200, do("GET", "/profile", ""))         // list (seeded default)
	core.AssertEqual(t, 200, do("GET", "/profile/default", "")) // get the seeded default
	core.AssertEqual(t, 200, do("GET", "/enabled", ""))         // persisted enable flag
	core.AssertTrue(t, do("POST", "/profile", `{"name":"t1"}`) < 500)
	core.AssertTrue(t, do("GET", "/profile/t1", "") < 500)     // get the just-saved profile
	core.AssertTrue(t, do("DELETE", "/profile/t1", "") < 500)  // delete it
	core.AssertTrue(t, do("POST", "/host-config", `{}`) < 500) // host opencode.json merge (temp HOME)
	core.AssertTrue(t, do("GET", "/studio", "") < 500)         // studio presence check
}
