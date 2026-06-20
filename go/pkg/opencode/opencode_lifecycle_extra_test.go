// SPDX-Licence-Identifier: EUPL-1.2

// Lifecycle coverage for opencode.go's container-driving surface —
// Start, Stop, waitHealthy, applyProfile. The previously-"untestable"
// verdict was wrong: s.proc() resolves the process runtime via
// core.ServiceFor[*process.Service](c, "process"), so registering a
// REAL process.Service into the test core and pointing the opencode
// Runtime at a harmless stand-in binary drives Run/Start/Stop through
// the real process plumbing with no docker daemon. This is exactly how
// the process package's own tests exercise Run (echo/true/false/sh) —
// "fake" here means "real service, harmless runtime", because
// s.proc()'s return type is concrete and can't be interface-swapped.
//
// allocatePort's package vars (pickPortInRange, portProbe) and the
// Start path's health server are pinned/overridden with restore-via-
// Cleanup, mirroring newTestService's kvOnce reset-before-and-after.

package opencode

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
	"dappco.re/go/process"
)

// procBackedService is newBackedService + a real process.Service
// registered under "process" so s.proc() resolves non-nil. runtime is
// the binary opencode's ps.Run will exec in place of docker — pass
// "true" for a success-empty-output runtime, "false" for a failing
// one, or an absolute path to a fixture script for structured stdout.
func procBackedService(t *testing.T, runtime string) *Service {
	t.Helper()
	c := core.New(core.WithOption("name", "opencode-test"))
	resetKV(t)

	r := NewService(Options{Runtime: runtime})(c)
	core.AssertTrue(t, r.OK)
	svc, _ := r.Value.(*Service)
	if svc == nil {
		t.Fatal("NewService registrar returned a nil *Service")
	}

	// Real process service into the same core — this is the seam.
	pr := process.NewService(process.Options{})(c)
	core.AssertTrue(t, pr.OK)
	core.AssertTrue(t, c.RegisterService("process", pr.Value).OK)
	core.AssertTrue(t, svc.proc() != nil)

	mountSandboxStore(t, svc)
	return svc
}

// resetKV mirrors newTestService's kvOnce reset-before-and-after so the
// process-global kv() store binding never leaks across tests. Also pins
// HOME to a temp dir so the DuckDB KV is isolated.
func resetKV(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	reset := func() {
		kvOnce = core.Once{}
		kvInst = nil
		kvErr = nil
	}
	reset()
	t.Cleanup(reset)
}

// mountSandboxStore mounts an in-memory DuckDB with the Sandbox table so
// orm.Of[Sandbox](c).Save/Find/Get round-trip in Start/Stop/Reconcile.
func mountSandboxStore(t *testing.T, svc *Service) {
	t.Helper()
	mr := orm.NewDuckDB(":memory:")
	core.AssertTrue(t, mr.OK)
	m, _ := mr.Value.(*orm.DuckDBMedium)
	t.Cleanup(func() { _ = m.Close() })
	m.RegisterTable("opencode_sandboxes", Sandbox{}.Schema())
	core.AssertTrue(t, orm.Mount(svc.Core(), "default", m).OK)
}

// --- Stop ---

// TestOpencode_Stop_Good_RuntimeTrue — Stop with a "true" runtime: the
// docker-rm Run succeeds (exit 0), the proxy entry is dropped, and the
// seeded Running record flips to Stopped. Drives the whole Stop body.
func TestOpencode_Stop_Good_RuntimeTrue(t *testing.T) {
	svc := procBackedService(t, "true")

	sb := Sandbox{ID: "oc-stop", Image: "img", HostPort: 51823, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
	svc.proxy.Set("oc-stop", "http://127.0.0.1:51823", "")

	r := svc.Stop("oc-stop")
	core.AssertTrue(t, r.OK)

	// Record flipped to Stopped.
	findR := orm.Of[Sandbox](svc.Core()).Find("oc-stop")
	core.AssertTrue(t, findR.OK)
	got, _ := findR.Value.(Sandbox)
	core.AssertEqual(t, StatusStopped, got.Status)
}

// TestOpencode_Stop_Bad_EmptyID — empty id short-circuits before proc().
func TestOpencode_Stop_Bad_EmptyID(t *testing.T) {
	svc := procBackedService(t, "true")
	r := svc.Stop("   ")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "id is required")
}

// TestOpencode_Stop_NoProc_Unavailable — a runtime-less Service surfaces
// the "process service unavailable" leg (proc() nil).
func TestOpencode_Stop_NoProc_Unavailable(t *testing.T) {
	r := (&Service{}).Stop("oc-x")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "process service unavailable")
}

// TestOpencode_Stop_Good_RuntimeFalse_MissingRecord — a "false" runtime
// makes the docker-rm Run fail (exit 1), but Stop ignores rm failure by
// design; with no orm record the Find leg is skipped and Stop still
// returns Ok (the record-find branch is the only orm touch).
func TestOpencode_Stop_Good_RuntimeFalse_MissingRecord(t *testing.T) {
	svc := procBackedService(t, "false")
	svc.proxy.Set("oc-gone", "http://127.0.0.1:1", "")

	r := svc.Stop("oc-gone")
	core.AssertTrue(t, r.OK)
}

// --- waitHealthy (free function, tested directly) ---

// TestOpencode_waitHealthy_Good — a 200 on /global/health returns Ok
// promptly.
func TestOpencode_waitHealthy_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/global/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := waitHealthy(srv.URL, "", 2*core.Second)
	core.AssertTrue(t, r.OK)
}

// TestOpencode_waitHealthy_Good_WithAuth — the Authorization header is
// forwarded when authHeader is non-empty.
func TestOpencode_waitHealthy_Good_WithAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "Basic test-cred", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := waitHealthy(srv.URL, "Basic test-cred", 2*core.Second)
	core.AssertTrue(t, r.OK)
}

// TestOpencode_waitHealthy_Bad_NeverHealthy — a server that always 503s
// fails after the (short) timeout with the documented message.
func TestOpencode_waitHealthy_Bad_NeverHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	r := waitHealthy(srv.URL, "", 300*core.Millisecond)
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "did not become healthy")
}

// --- applyProfile (free function, tested directly) ---

// TestOpencode_applyProfile_Good — a 200 on PATCH /global/config with a
// JSON content-type returns Ok.
func TestOpencode_applyProfile_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "/global/config", r.URL.Path)
		core.AssertEqual(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := applyProfile(srv.URL, "Basic test-cred", Profile{Name: "default"})
	core.AssertTrue(t, r.OK)
}

// TestOpencode_applyProfile_Bad_4xx — a 4xx response surfaces the status
// code + body in the error (the >=400 leg).
func TestOpencode_applyProfile_Bad_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()

	r := applyProfile(srv.URL, "", Profile{Name: "default"})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "patch returned 400")
	core.AssertContains(t, r.Error(), "nope")
}

// TestOpencode_applyProfile_Bad_RequestBuild — a malformed target URL
// fails the request-build leg before any network call.
func TestOpencode_applyProfile_Bad_RequestBuild(t *testing.T) {
	r := applyProfile("://bad-url", "", Profile{Name: "default"})
	core.AssertFalse(t, r.OK)
}

// TestOpencode_applyProfile_Bad_Unreachable — a target nothing is
// listening on fails the client.Do leg.
func TestOpencode_applyProfile_Bad_Unreachable(t *testing.T) {
	r := applyProfile("http://127.0.0.1:1", "", Profile{Name: "default"})
	core.AssertFalse(t, r.OK)
}

// --- Start (full path: port-pinned health server) ---

// startHealthServer stands up an httptest server answering /global/health
// (200) + PATCH /global/config (200), pins pickPortInRange to its port,
// and overrides portProbe to accept it (the server already holds the
// port, so the real probe would reject it). All three package globals are
// restored via Cleanup so allocatePort/proxy tests don't inherit stubs.
func startHealthServer(t *testing.T) (*httptest.Server, int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	core.AssertNoError(t, err)
	pr := core.Atoi(u.Port())
	core.AssertTrue(t, pr.OK)
	port := pr.Value.(int)

	origPick, origProbe := pickPortInRange, portProbe
	t.Cleanup(func() { pickPortInRange, portProbe = origPick, origProbe })
	pickPortInRange = func() int { return port }
	portProbe = func(int) error { return nil }

	return srv, port
}

// TestOpencode_Start_Good_FullPath — Start drives the whole happy path:
// allocate (pinned) port, ServerPassword + InstallID mint, ps.Run the
// docker-run (harmless "true"), persist the Sandbox, register the proxy
// target, waitHealthy (200), applyProfile (200). Returns the new id and
// the record reads back Running.
func TestOpencode_Start_Good_FullPath(t *testing.T) {
	svc := procBackedService(t, "true")
	_, port := startHealthServer(t)

	r := svc.Start("")
	core.AssertTrue(t, r.OK)
	id, _ := r.Value.(string)
	core.AssertNotEmpty(t, id)

	findR := orm.Of[Sandbox](svc.Core()).Find(id)
	core.AssertTrue(t, findR.OK)
	sb, _ := findR.Value.(Sandbox)
	core.AssertEqual(t, StatusRunning, sb.Status)
	core.AssertEqual(t, port, sb.HostPort)
}

// TestOpencode_Start_Bad_RunFails — a "false" runtime makes the docker-run
// Run fail; Start returns that failure before persisting anything.
func TestOpencode_Start_Bad_RunFails(t *testing.T) {
	svc := procBackedService(t, "false")
	_, _ = startHealthServer(t)

	r := svc.Start("")
	core.AssertFalse(t, r.OK)

	// Nothing persisted — Status stays empty.
	statusR := svc.Status()
	core.AssertTrue(t, statusR.OK)
	running, _ := statusR.Value.([]Sandbox)
	core.AssertEqual(t, 0, len(running))
}

// TestOpencode_Start_NoProc_Unavailable — runtime-less Service surfaces
// the proc()-nil leg.
func TestOpencode_Start_NoProc_Unavailable(t *testing.T) {
	r := (&Service{}).Start("")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "process service unavailable")
}

// TestOpencode_Start_Bad_UnknownProfile — a non-existent profile name
// fails at GetProfile, before the container is spawned.
func TestOpencode_Start_Bad_UnknownProfile(t *testing.T) {
	svc := procBackedService(t, "true")
	_, _ = startHealthServer(t)

	r := svc.Start("does-not-exist")
	core.AssertFalse(t, r.OK)
}
