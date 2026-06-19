// SPDX-License-Identifier: EUPL-1.2

// Tests for the `core-agent hub` wiring (RFC.serve.md Unit B): the
// loopback coreapi.Engine builds with the three route groups, strict
// bind rejects a non-loopback address without --public, the bearer
// token is generate-or-load at 0600, and the brain→Laravel hop enforces
// loopback-or-wss://.

package main

import (
	"testing"

	core "dappco.re/go"
)

// --- buildHubEngine -----------------------------------------------

// TestHub_buildHubEngine_Good — the engine binds loopback and registers
// the opencode control, sandbox proxy, and brain route groups.
func TestHub_buildHubEngine_Good(t *testing.T) {
	c := newCoreAgent()
	cmds := applicationCommandSet{coreApp: c}

	engine, r := cmds.buildHubEngine(defaultHubHTTPAddr, "test-token", false)
	if !r.OK {
		t.Fatalf("buildHubEngine failed: %v", r.Value)
	}
	if engine.Addr() != defaultHubHTTPAddr {
		t.Fatalf("engine addr = %q, want %q", engine.Addr(), defaultHubHTTPAddr)
	}

	want := map[string]bool{
		"/v1/api/opencode": false,
		"/v1/api/sandbox":  false,
		"/api/brain":       false,
	}
	for _, g := range engine.Groups() {
		if _, ok := want[g.BasePath()]; ok {
			want[g.BasePath()] = true
		}
	}
	for base, seen := range want {
		if !seen {
			t.Fatalf("route group %q not registered on hub engine", base)
		}
	}
}

// TestHub_buildHubEngine_Bad_NonLoopbackRejected — strict bind rejects a
// non-loopback address at Serve time without --public.
func TestHub_buildHubEngine_Bad_NonLoopbackRejected(t *testing.T) {
	c := newCoreAgent()
	cmds := applicationCommandSet{coreApp: c}

	engine, r := cmds.buildHubEngine("0.0.0.0:9201", "test-token", false)
	if !r.OK {
		t.Fatalf("buildHubEngine failed: %v", r.Value)
	}
	ctx, cancel := core.WithCancel(c.Context())
	cancel() // ensure Serve does not block if validation passes unexpectedly
	if err := engine.Serve(ctx); err == nil {
		t.Fatal("expected non-loopback bind to be rejected without --public")
	}
}

// TestHub_buildHubEngine_Ugly_MissingOpencodeService — a Core without
// the opencode service cannot build the hub engine.
func TestHub_buildHubEngine_Ugly_MissingOpencodeService(t *testing.T) {
	c := core.New(core.WithOption("name", "core-agent"))
	cmds := applicationCommandSet{coreApp: c}

	if _, r := cmds.buildHubEngine(defaultHubHTTPAddr, "test-token", false); r.OK {
		t.Fatal("expected build to fail without the opencode service registered")
	}
}

// --- hubLoadOrGenerateToken ---------------------------------------

// TestHub_hubLoadOrGenerateToken_Good — a fresh path mints a token and
// writes it 0600; a second call loads the same token.
func TestHub_hubLoadOrGenerateToken_Good(t *testing.T) {
	fs := (&core.Fs{}).New("/")
	dir := fs.TempDir("core-hub-token")
	defer fs.DeleteAll(dir)
	path := core.JoinPath(dir, "hub.token")

	tok1, r := hubLoadOrGenerateToken(fs, path)
	if !r.OK {
		t.Fatalf("generate failed: %v", r.Value)
	}
	if len(tok1) != 64 { // 32 bytes hex-encoded
		t.Fatalf("token length = %d, want 64 hex chars", len(tok1))
	}

	tok2, r := hubLoadOrGenerateToken(fs, path)
	if !r.OK {
		t.Fatalf("load failed: %v", r.Value)
	}
	if tok1 != tok2 {
		t.Fatalf("reload produced a different token: %q vs %q", tok1, tok2)
	}
}

// TestHub_hubLoadOrGenerateToken_Bad — nil fs / empty path fails loud.
func TestHub_hubLoadOrGenerateToken_Bad(t *testing.T) {
	if _, r := hubLoadOrGenerateToken(nil, "/tmp/x"); r.OK {
		t.Fatal("nil fs must fail")
	}
	fs := (&core.Fs{}).New("/")
	if _, r := hubLoadOrGenerateToken(fs, ""); r.OK {
		t.Fatal("empty path must fail")
	}
}

// --- laravelURLReject ---------------------------------------------

// TestHub_laravelURLReject_Good — loopback ws:// and any wss:// pass.
func TestHub_laravelURLReject_Good(t *testing.T) {
	for _, u := range []string{
		"ws://localhost:9876/ws",
		"ws://127.0.0.1:9876/ws",
		"ws://[::1]:9876/ws",
		"wss://api.lthn.ai/ws",
		"wss://localhost/ws",
	} {
		if reason := laravelURLReject(u); reason != "" {
			t.Fatalf("permitted URL %q rejected: %q", u, reason)
		}
	}
}

// TestHub_laravelURLReject_Bad — a non-loopback ws:// (cleartext bearer)
// is rejected.
func TestHub_laravelURLReject_Bad(t *testing.T) {
	for _, u := range []string{
		"ws://api.lthn.ai/ws",
		"ws://10.0.0.5:9876/ws",
		"ws://example.com:8080/ws",
		"ws://127.evil.com:9876/ws", // SSRF: a substring "127." check wrongly admits this hostname
	} {
		if laravelURLReject(u) == "" {
			t.Fatalf("non-loopback ws:// %q must be rejected", u)
		}
	}
}

// --- pure helpers (defaultHubTokenFile / defaultHubAuditPath / optStringOr /
//     publicSuffix / auditMetaString / toBytes / hostIsLoopback) -------------

// TestHub_defaultHubTokenFile_Good — the token file sits under the core
// workspace root at hub/hub.token.
func TestHub_defaultHubTokenFile_Good(t *testing.T) {
	core.AssertContains(t, defaultHubTokenFile(), core.JoinPath("hub", "hub.token"))
}

// TestHub_defaultHubAuditPath_Good — the audit log sits under the core
// workspace root at hub/audit.jsonl.
func TestHub_defaultHubAuditPath_Good(t *testing.T) {
	core.AssertContains(t, defaultHubAuditPath(), core.JoinPath("hub", "audit.jsonl"))
}

// TestHub_optStringOr_Good — a present, non-empty option wins over the fallback.
func TestHub_optStringOr_Good(t *testing.T) {
	opts := core.NewOptions(core.Option{Key: "addr", Value: "127.0.0.1:9201"})
	core.AssertEqual(t, "127.0.0.1:9201", optStringOr(opts, "addr", "fallback"))
}

// TestHub_optStringOr_Bad_MissingFallsBack — a missing key yields the fallback.
func TestHub_optStringOr_Bad_MissingFallsBack(t *testing.T) {
	core.AssertEqual(t, "fallback", optStringOr(core.NewOptions(), "addr", "fallback"))
}

// TestHub_optStringOr_Ugly_WhitespaceFallsBack — a whitespace-only value trims
// to empty and yields the fallback.
func TestHub_optStringOr_Ugly_WhitespaceFallsBack(t *testing.T) {
	opts := core.NewOptions(core.Option{Key: "addr", Value: "   "})
	core.AssertEqual(t, "fallback", optStringOr(opts, "addr", "fallback"))
}

// TestHub_publicSuffix_Good — --public annotates the bind log line.
func TestHub_publicSuffix_Good(t *testing.T) {
	core.AssertEqual(t, ", PUBLIC opt-in", publicSuffix(true))
}

// TestHub_publicSuffix_Bad_PrivateEmpty — loopback bind adds no annotation.
func TestHub_publicSuffix_Bad_PrivateEmpty(t *testing.T) {
	core.AssertEqual(t, "", publicSuffix(false))
}

// TestHub_auditMetaString_Good — a present string field is returned.
func TestHub_auditMetaString_Good(t *testing.T) {
	core.AssertEqual(t, "go-io", auditMetaString(map[string]any{"repo": "go-io"}, "repo"))
}

// TestHub_auditMetaString_Bad_NilOrMissing — nil map or absent key yields "".
func TestHub_auditMetaString_Bad_NilOrMissing(t *testing.T) {
	core.AssertEqual(t, "", auditMetaString(nil, "repo"))
	core.AssertEqual(t, "", auditMetaString(map[string]any{"repo": "go-io"}, "agent"))
}

// TestHub_auditMetaString_Ugly_NonString — a non-string value yields "".
func TestHub_auditMetaString_Ugly_NonString(t *testing.T) {
	core.AssertEqual(t, "", auditMetaString(map[string]any{"count": 7}, "count"))
}

// TestHub_toBytes_Good_String — a string passes through unchanged.
func TestHub_toBytes_Good_String(t *testing.T) {
	core.AssertEqual(t, "abc", toBytes("abc"))
}

// TestHub_toBytes_Bad_ByteSlice — a []byte is coerced to its string form.
func TestHub_toBytes_Bad_ByteSlice(t *testing.T) {
	core.AssertEqual(t, "abc", toBytes([]byte("abc")))
}

// TestHub_toBytes_Ugly_OtherType — any other type yields "".
func TestHub_toBytes_Ugly_OtherType(t *testing.T) {
	core.AssertEqual(t, "", toBytes(42))
}

// TestHub_hostIsLoopback_Good — localhost and loopback IPs (incl. an
// unterminated "[::1") count as loopback.
func TestHub_hostIsLoopback_Good(t *testing.T) {
	for _, h := range []string{"localhost", "127.0.0.1:9876", "[::1]:9876", "[::1"} {
		core.AssertTrue(t, hostIsLoopback(h))
	}
}

// TestHub_hostIsLoopback_Bad_OffBox — DNS names (incl. the "127."-prefixed
// SSRF bait) and non-loopback IPs are rejected.
func TestHub_hostIsLoopback_Bad_OffBox(t *testing.T) {
	for _, h := range []string{"api.lthn.ai", "10.0.0.5:9876", "127.evil.com:9876"} {
		core.AssertFalse(t, hostIsLoopback(h))
	}
}

// TestHub_hostIsLoopback_Ugly_Empty — an empty host is not loopback.
func TestHub_hostIsLoopback_Ugly_Empty(t *testing.T) {
	core.AssertFalse(t, hostIsLoopback(""))
}
