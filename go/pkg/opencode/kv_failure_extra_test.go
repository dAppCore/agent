// SPDX-Licence-Identifier: EUPL-1.2

// Fail-closed coverage for the KV-backed surface. Every profile / auth /
// enable accessor opens with kv() and documents a fall-back when the store
// is unreachable ("better to start cold than spawn on a transient KV blip",
// "a transient KV failure doesn't bork the proxy"). Those arms are the
// contract; this file exercises them by forcing kv() to fail through the
// SAME package vars newTestService already manipulates (kvOnce / kvInst /
// kvErr) — using the existing seam, not adding one. The test's Cleanup
// (registered by newTestService) restores the vars, and tests are
// sequential, so the forced failure never leaks.
//
// Plus two real-corruption cases: a malformed value stored under a profile
// key drives the GetProfile JSON-unmarshal-fail arm and the ListProfiles
// skip-bad-row arm.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// failKV forces the process-global kv() to return Fail for the rest of the
// test. MUST be called AFTER newTestService has finished seeding (kv() is
// memoised via kvOnce; we mark the cached error so the kvErr branch fires).
// newTestService's Cleanup resets kvOnce/kvInst/kvErr.
func failKV(t *testing.T) {
	t.Helper()
	kvErr = core.E("opencode.kv", "injected store failure", nil)
	kvInst = nil
}

// TestKVFail_GetProfile_FailsClosed — GetProfile surfaces the kv() failure
// rather than returning a phantom profile.
func TestKVFail_GetProfile_FailsClosed(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	r := svc.GetProfile("default")
	core.AssertFalse(t, r.OK)
}

// TestKVFail_ListProfiles_FailsClosed — ListProfiles surfaces the kv()
// failure (the !r.OK arm before GetAll).
func TestKVFail_ListProfiles_FailsClosed(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	r := svc.ListProfiles()
	core.AssertFalse(t, r.OK)
}

// TestKVFail_SaveProfile_FailsClosed — a schema-valid profile still fails
// to persist when kv() is down (the post-validation kv() arm).
func TestKVFail_SaveProfile_FailsClosed(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	r := svc.SaveProfile(Profile{Name: "ok-but-no-store"})
	core.AssertFalse(t, r.OK)
}

// TestKVFail_DeleteProfile_FailsClosed — DeleteProfile of a non-default
// name reaches the kv() arm (the default-name guard fires earlier, so we
// use a normal name) and fails closed.
func TestKVFail_DeleteProfile_FailsClosed(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	r := svc.DeleteProfile("some-profile")
	core.AssertFalse(t, r.OK)
}

// TestKVFail_setEnabled_FailsClosed — setEnabled surfaces the kv() failure
// so Enable/Disable abort before any container action.
func TestKVFail_setEnabled_FailsClosed(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	core.AssertFalse(t, svc.setEnabled(true).OK)
}

// TestKVFail_IsEnabled_DefaultsFalse — IsEnabled defaults to false on a
// kv() failure (no auto-spawn on a transient blip).
func TestKVFail_IsEnabled_DefaultsFalse(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	core.AssertFalse(t, svc.IsEnabled())
}

// TestKVFail_readEnabledFlag_NotPresent — readEnabledFlag returns
// ("", false) on a kv() failure (the kv-error arm, distinct from the
// NotFound arm covered elsewhere).
func TestKVFail_readEnabledFlag_NotPresent(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	raw, present := svc.readEnabledFlag()
	core.AssertFalse(t, present)
	core.AssertEqual(t, "", raw)
}

// TestKVFail_ServerPassword_FailsClosed — ServerPassword surfaces the kv()
// failure (it cannot mint + persist without the store).
func TestKVFail_ServerPassword_FailsClosed(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	core.AssertFalse(t, svc.ServerPassword().OK)
}

// TestKVFail_InstallID_FailsClosed — InstallID surfaces the kv() failure.
func TestKVFail_InstallID_FailsClosed(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	core.AssertFalse(t, svc.InstallID().OK)
}

// TestKVFail_authHeader_EmptyOnFailure — authHeader returns "" when
// ServerPassword fails, so applyAuth skips injection rather than sending a
// broken header (the proxy keeps working through a transient KV blip).
func TestKVFail_authHeader_EmptyOnFailure(t *testing.T) {
	svc := newTestService(t)
	failKV(t)
	core.AssertEqual(t, "", svc.authHeader())

	// applyAuth must then leave the request header unset (no-op arm).
	reqR := core.NewHTTPRequest(core.MethodGet, "http://127.0.0.1:1/x", nil)
	core.AssertTrue(t, reqR.OK)
	req := reqR.Value.(*core.Request)
	svc.applyAuth(req)
	core.AssertEqual(t, "", req.Header.Get("Authorization"))
}

// TestKVFail_Control_disable_500_HTTP — the disable control handler returns
// 500 when Disable fails (setEnabled kv() arm). Covers the disable error
// leg the success sweep can't reach.
func TestKVFail_Control_disable_500_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	failKV(t) // after route registration; the handler call hits the failure.

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/disable", nil))
	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
	core.AssertContains(t, w.Body.String(), "error")
}

// TestKVFail_Control_profileList_500_HTTP — the profileList control handler
// returns 500 when ListProfiles fails (kv() arm).
func TestKVFail_Control_profileList_500_HTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := newTestService(t)

	g := NewControlGroup(svc)
	e := gin.New()
	g.RegisterRoutes(e.Group(""))

	failKV(t)

	w := httptest.NewRecorder()
	e.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/profile", nil))
	core.AssertEqual(t, http.StatusInternalServerError, w.Code)
	core.AssertContains(t, w.Body.String(), "error")
}

// TestKVCorrupt_GetProfile_UnmarshalFails — a malformed value stored under
// a profile key makes GetProfile reach its JSON-unmarshal-fail arm (the
// stored bytes are not a valid Profile), surfacing a failure rather than a
// half-decoded struct.
func TestKVCorrupt_GetProfile_UnmarshalFails(t *testing.T) {
	svc := newTestService(t)

	st, r := kv()
	core.AssertTrue(t, r.OK)
	core.AssertNoError(t, st.Set(profileStoreGroup, "corrupt", "{not valid json"))

	got := svc.GetProfile("corrupt")
	core.AssertFalse(t, got.OK)
}

// TestKVCorrupt_ListProfiles_SkipsBadRow — ListProfiles skips a corrupt
// stored value (the unmarshal-fail-continue arm) and still returns the
// well-formed rows. The seeded default decodes; the corrupt row is dropped.
func TestKVCorrupt_ListProfiles_SkipsBadRow(t *testing.T) {
	svc := newTestService(t)

	st, r := kv()
	core.AssertTrue(t, r.OK)
	core.AssertNoError(t, st.Set(profileStoreGroup, "corrupt", "}{ not json"))

	lr := svc.ListProfiles()
	core.AssertTrue(t, lr.OK)
	rows, _ := lr.Value.([]Profile)
	// The default profile (seeded by newTestService) decodes; the corrupt
	// row is skipped, so at least one good row survives and the bad one is
	// not among them.
	core.AssertTrue(t, len(rows) >= 1)
	for _, p := range rows {
		core.AssertNotEqual(t, "corrupt", p.Name)
	}
}
