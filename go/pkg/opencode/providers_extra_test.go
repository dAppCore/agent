// SPDX-Licence-Identifier: EUPL-1.2

// callOpenCode coverage — the shared internal HTTP client opencode-serve
// calls route through (ProviderList, Generate's session flow). Tested
// directly against an httptest server; the Basic-auth header is injected
// from the service's ServerPassword (KV-backed via newTestService).

package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestProviders_callOpenCode_Good — a 200 returns (body, 200, nil) and the
// auto-injected Authorization header is present on the wire.
func TestProviders_callOpenCode_Good(t *testing.T) {
	svc := newTestService(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertContains(t, r.Header.Get("Authorization"), "Basic ")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	body, code, err := svc.callOpenCode(core.MethodGet, srv.URL+"/provider", nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, http.StatusOK, code)
	core.AssertContains(t, body, "ok")
}

// TestProviders_callOpenCode_Good_PassesStatus — a non-2xx status is
// returned to the caller verbatim (callOpenCode does not treat >=400 as a
// transport error; the caller decides).
func TestProviders_callOpenCode_Good_PassesStatus(t *testing.T) {
	svc := newTestService(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	body, code, err := svc.callOpenCode(core.MethodGet, srv.URL, nil)
	core.AssertNoError(t, err)
	core.AssertEqual(t, http.StatusInternalServerError, code)
	core.AssertContains(t, body, "boom")
}

// TestProviders_callOpenCode_Bad_RequestBuild — a malformed URL fails the
// request-build leg, returning a non-nil error and zero status.
func TestProviders_callOpenCode_Bad_RequestBuild(t *testing.T) {
	svc := newTestService(t)

	_, code, err := svc.callOpenCode(core.MethodGet, "://bad-url", nil)
	core.AssertError(t, err)
	core.AssertEqual(t, 0, code)
}

// TestProviders_callOpenCode_Bad_Unreachable — a dead target fails the
// client.Do leg.
func TestProviders_callOpenCode_Bad_Unreachable(t *testing.T) {
	svc := newTestService(t)

	_, _, err := svc.callOpenCode(core.MethodGet, "http://127.0.0.1:1", nil)
	core.AssertError(t, err)
}
