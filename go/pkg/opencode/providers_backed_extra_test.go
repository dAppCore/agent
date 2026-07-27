// SPDX-Licence-Identifier: EUPL-1.2

// Success-path coverage for the provider-enumeration surface. The
// existing tests reach ProviderList only on its empty-id / not-running
// guards; this file seeds a *running* Sandbox row whose HostPort points
// at an httptest server impersonating opencode-serve's GET /provider, so
// ProviderList → targetFor → callOpenCode all execute their happy path,
// and the upstream-error (code >= 400) branch is exercised too.
//
// No docker, no real container: the "opencode-serve" is an in-process
// httptest server bound to 127.0.0.1, exactly the address callOpenCode
// dials. The control providerList HTTP handler is driven through gin on
// the same backed Service. seedRunningSandbox is reused from
// web_backed_extra_test.go.

package opencode

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// listenerPort extracts the dynamic port an httptest server bound to.
// callOpenCode dials http://127.0.0.1:<HostPort>/provider, so a seeded
// Sandbox.HostPort must equal this for the request to reach the server.
func listenerPort(t *testing.T, srv *httptest.Server) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener addr %q: %v", srv.Listener.Addr(), err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse port %q: %v", portStr, err)
	}
	return port
}

// TestProviderList_Good_ReturnsUpstreamBody — a running sandbox + a live
// /provider endpoint: ProviderList returns the upstream JSON verbatim, the
// request carries the injected Basic auth header, and the path is /provider.
func TestProviderList_Good_ReturnsUpstreamBody(t *testing.T) {
	const providerJSON = `{"providers":[{"id":"lthn","name":"Lethean"}]}`

	var gotAuth, gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(providerJSON))
	}))
	defer srv.Close()

	svc := newBackedService(t)
	seedRunningSandbox(t, svc, "oc-running", listenerPort(t, srv))

	r := svc.ProviderList("oc-running")
	core.AssertTrue(t, r.OK)
	body, _ := r.Value.(string)
	core.AssertEqual(t, providerJSON, body)

	// The proxy-bypassing direct call must hit GET /provider with the
	// Basic auth header applyAuth installs.
	core.AssertEqual(t, "/provider", gotPath)
	core.AssertEqual(t, http.MethodGet, gotMethod)
	want := "Basic " + core.Base64Encode([]byte("opencode:"+svc.ServerPassword().Value.(string)))
	core.AssertEqual(t, want, gotAuth)
}

// TestProviderList_Bad_UpstreamErrorStatus — the sandbox is running but
// opencode-serve returns 503: ProviderList Fails and the error text carries
// the upstream status + body (the code >= 400 branch).
func TestProviderList_Bad_UpstreamErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("provider backend down"))
	}))
	defer srv.Close()

	svc := newBackedService(t)
	seedRunningSandbox(t, svc, "oc-running", listenerPort(t, srv))

	r := svc.ProviderList("oc-running")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "503")
	core.AssertContains(t, r.Error(), "provider backend down")
}

// TestProviderList_Bad_EmptyID — the empty-id guard fires before any ORM
// or HTTP work (kept distinct: this is the validation arm, not the
// not-running arm covered in web_backed_extra_test.go).
func TestProviderList_Bad_EmptyID(t *testing.T) {
	svc := newBackedService(t)
	r := svc.ProviderList("  ")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "id is required")
}

// TestControl_ProviderList_Good_HTTP — the /sandbox/:id/providers control
// route streams the upstream /provider body through with 200 + the
// application/json content type. Drives the providerList gin handler's
// success path against the same backed Service + httptest opencode-serve.
func TestControl_ProviderList_Good_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const providerJSON = `{"providers":[{"id":"anthropic"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(providerJSON))
	}))
	defer srv.Close()

	svc := newBackedService(t)
	seedRunningSandbox(t, svc, "oc-running", listenerPort(t, srv))

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sandbox/oc-running/providers", nil))

	core.AssertEqual(t, http.StatusOK, w.Code)
	core.AssertEqual(t, providerJSON, w.Body.String())
	core.AssertContains(t, w.Header().Get("Content-Type"), "application/json")
}

// TestControl_ProviderList_Bad_HTTP — a missing sandbox makes the
// providerList handler surface 500 + the error envelope (the !r.OK arm).
func TestControl_ProviderList_Bad_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newBackedService(t)
	// No sandbox seeded → Inspect fails → ProviderList fails.

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sandbox/nope/providers", nil))

	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
	core.AssertContains(t, w.Body.String(), "error")
}
