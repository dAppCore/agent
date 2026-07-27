// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
)

// newTestService builds a fully-initialised opencode Service against an
// isolated, temp-$HOME DuckDB KV store. The process-global kv() store
// (profile.go kvOnce) is reset before AND after via t.Cleanup so the binding
// never leaks to another test — the cross-test hazard host_config_mode_test.go
// documents but sidesteps. Tests are sequential (no t.Parallel here), so the
// global mutation is safe.
func newTestService(t *testing.T) *Service {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	reset := func() {
		kvOnce = core.Once{}
		kvInst = nil
		kvErr = nil
	}
	reset()
	t.Cleanup(reset)

	c := core.New(core.WithOption("name", "opencode-test"))
	r := NewService(Options{})(c)
	core.AssertTrue(t, r.OK)
	svc, _ := r.Value.(*Service)
	if svc == nil {
		t.Fatal("NewService registrar returned a nil *Service")
	}
	return svc
}

// TestAuth_ServerPassword_Good_GeneratesAndPersists — first call mints a
// 48-char hex (24-byte) password and persists it; subsequent calls are
// idempotent and return the same value.
func TestAuth_ServerPassword_Good_GeneratesAndPersists(t *testing.T) {
	svc := newTestService(t)

	r1 := svc.ServerPassword()
	core.AssertTrue(t, r1.OK)
	pw1 := r1.Value.(string)
	core.AssertEqual(t, 48, len(pw1))

	r2 := svc.ServerPassword()
	core.AssertTrue(t, r2.OK)
	core.AssertEqual(t, pw1, r2.Value.(string))
}

// TestAuth_InstallID_Good_GeneratesAndPersists — first call mints a 32-char
// hex (16-byte) id and persists it; subsequent calls return the same value.
func TestAuth_InstallID_Good_GeneratesAndPersists(t *testing.T) {
	svc := newTestService(t)

	r1 := svc.InstallID()
	core.AssertTrue(t, r1.OK)
	id1 := r1.Value.(string)
	core.AssertEqual(t, 32, len(id1))

	r2 := svc.InstallID()
	core.AssertTrue(t, r2.OK)
	core.AssertEqual(t, id1, r2.Value.(string))
}

// TestAuth_authHeader_Good_BasicFormat — the header is HTTP Basic with the
// canonical "opencode:<password>" credential built from ServerPassword.
func TestAuth_authHeader_Good_BasicFormat(t *testing.T) {
	svc := newTestService(t)

	pw := svc.ServerPassword().Value.(string)
	want := "Basic " + core.Base64Encode([]byte("opencode:"+pw))
	core.AssertEqual(t, want, svc.authHeader())
}
