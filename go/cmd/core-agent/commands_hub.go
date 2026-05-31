// SPDX-License-Identifier: EUPL-1.2

// The `core-agent hub` subcommand — RFC.serve.md Unit B. core/agent
// stops being only a CLI dispatcher and becomes a served hub: a loopback
// coreapi.Engine HTTP control plane (opencode lifecycle + sandbox proxy
// + brain memory) plus a fail-closed MCP HTTP+SSE tool plane for
// Cladius. The hub is the new audit edge (the opencode no-op hooks
// relied on the desktop SASE edge that Unit D deletes).
//
//	core-agent hub --http 127.0.0.1:9201 --token-file ~/.core/hub.token
//	core-agent hub --mcp-http 127.0.0.1:9202 --no-mcp

package main

import (
	"context"
	"net"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/audit"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/opencode"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
	coreapi "dappco.re/go/api"
	"dappco.re/go/ws"
)

const (
	// defaultHubHTTPAddr is the HTTP control-plane bind — loopback, on a
	// fixed hub port distinct from the desktop's :8000 (lthn serve) and
	// the lthn-ai LEM-runtime :9100. RFC.serve.md §3.2 illustrates :8787;
	// Mantis #1807 Unit B pins :9201 to keep clear of both desktop and
	// lthn-ai on a shared box.
	defaultHubHTTPAddr = "127.0.0.1:9201"

	// defaultHubMCPAddr is the served-MCP HTTP+SSE bind — loopback,
	// distinct from the HTTP control plane (:9201) and the legacy
	// :9100/:9101 MCP defaults (RFC.serve.md §10.2).
	defaultHubMCPAddr = "127.0.0.1:9202"

	// hubTokenFileMode is the 0600 mode for the bearer token file — the
	// hub bearer is container-exec-grade (RFC.serve.md §7.3.2), so the
	// file is owner-read-write only.
	hubTokenFileMode core.FileMode = 0o600

	// hubDesktopOrigin is the single CORS origin permitted on the
	// control plane — the desktop GUI (RFC.serve.md §4.1).
	hubDesktopOrigin = "http://localhost"
)

// hub stands up the served hub and blocks until the process context is
// cancelled. It is the long-running daemon mode of core-agent.
//
//	core-agent hub --http 127.0.0.1:9201
func (commands applicationCommandSet) hub(opts core.Options) core.Result {
	c := commands.coreApp
	ctx := c.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	httpAddr := optStringOr(opts, "http", defaultHubHTTPAddr)
	mcpAddr := optStringOr(opts, "mcp-http", defaultHubMCPAddr)
	noHTTP := opts.Bool("no-http")
	noMCP := opts.Bool("no-mcp")
	public := opts.Bool("public")

	// Bearer token: generate-or-load, 0600. The control-plane listener
	// refuses to start without it (RFC.serve.md §3.2).
	tokenFile := optStringOr(opts, "token-file", defaultHubTokenFile())
	token, r := hubLoadOrGenerateToken(c.Fs(), tokenFile)
	if !r.OK {
		return r
	}

	// Audit edge: a real pkg/audit JSONL sink installed into opencode so
	// the spawn/stop/upgrade/proxy/port hooks (no-ops by default) record
	// the privilege-bearing decision flow. NON-OPTIONAL — the no-op was
	// only safe because of the desktop edge Unit D deletes
	// (RFC.serve.md §7.3.1).
	sink := audit.NewFileSink(c.Fs(), defaultHubAuditPath())
	opencode.SetAuditSink(func(event, scope, outcome, requestID string, meta map[string]any) {
		sink.Emit(audit.Event{
			Event:      event,
			Outcome:    outcome,
			RequestID:  requestID,
			SandboxID:  auditMetaString(meta, "sandbox_id"),
			PathPrefix: auditMetaString(meta, "path_prefix"),
			Meta:       meta,
		})
	})

	if noHTTP && noMCP {
		return core.Fail(core.E("hub", "nothing to serve: both --no-http and --no-mcp set", nil))
	}

	errCh := make(chan error, 2)
	started := 0

	if !noHTTP {
		engine, r := commands.buildHubEngine(httpAddr, token, public)
		if !r.OK {
			return r
		}
		started++
		go func() { errCh <- engine.Serve(ctx) }()
		applicationPrint("hub: HTTP control plane on %s (loopback%s, bearer required)", httpAddr, publicSuffix(public))
		applicationPrint("hub: token-file %s", tokenFile)
	}

	if !noMCP {
		mcpSvc, ok := core.ServiceFor[*coremcp.Service](c, "mcp")
		if !ok || mcpSvc == nil {
			return core.Fail(core.E("hub", "mcp service not registered — cannot serve MCP plane", nil))
		}
		// The served MCP transport is fail-closed (RFC.serve.md §7.1):
		// it refuses to bind without a distinct MCP_JWT_SECRET. Surface
		// the requirement here rather than letting ServeHTTP error after
		// the control plane is already up.
		if core.Trim(core.Env("MCP_JWT_SECRET")) == "" {
			return core.Fail(core.E("hub", "MCP_JWT_SECRET is required for the served MCP plane (distinct from the API token, no fallback)", nil))
		}
		started++
		go func() { errCh <- mcpSvc.ServeHTTP(ctx, mcpAddr) }()
		applicationPrint("hub: MCP HTTP+SSE tool plane on %s (loopback, per-request bearer)", mcpAddr)
	}

	// Block until the first server returns (a bind error or ctx cancel).
	for i := 0; i < started; i++ {
		if err := <-errCh; err != nil {
			return core.Fail(err)
		}
	}
	return core.Ok(nil)
}

// buildHubEngine constructs the loopback coreapi.Engine with strict bind
// + mandatory bearer and registers the three route groups: opencode
// control (/v1/api/opencode), the opencode sandbox proxy
// (/v1/api/sandbox), and brain (/api/brain).
func (commands applicationCommandSet) buildHubEngine(
	addr, token string,
	public bool,
) (*coreapi.Engine, core.Result) {
	c := commands.coreApp

	opencodeSvc, ok := core.ServiceFor[*opencode.Service](c, "opencode")
	if !ok || opencodeSvc == nil {
		return nil, core.Fail(core.E("hub", "opencode service not registered", nil))
	}

	// brain provider: an ide.Bridge to the Laravel backend + a ws.Hub
	// for completion pushes. The brain→Laravel hop must be
	// loopback-or-wss:// (RFC.serve.md §7.3.4) — a non-loopback ws://
	// carries the bearer in cleartext and is rejected here.
	laravelURL := core.Trim(core.Env("LARAVEL_WS_URL"))
	if laravelURL == "" {
		laravelURL = ide.DefaultConfig().LaravelWSURL
	}
	if reason := laravelURLReject(laravelURL); reason != "" {
		return nil, core.Fail(core.E("hub", "brain→Laravel URL rejected: "+reason+" ("+laravelURL+")", nil))
	}
	hub := ws.NewHub()
	bridge := ide.NewBridge(hub, ide.Config{
		LaravelWSURL:  laravelURL,
		WorkspaceRoot: agentic.WorkspaceRoot(),
		Token:         core.Env("LARAVEL_WS_TOKEN"),
	})
	bridge.Start(c.Context())
	brainProvider := brain.NewProvider(bridge, hub)

	engineOpts := []coreapi.Option{
		coreapi.WithAddr(addr),
		coreapi.WithBearerAuth(token),
		coreapi.WithStrictBind(),
		coreapi.WithRequestID(),
		coreapi.WithCORS(hubDesktopOrigin),
	}
	if public {
		engineOpts = append(engineOpts, coreapi.WithPublicBind())
	}

	engine, err := coreapi.New(engineOpts...)
	if err != nil {
		return nil, core.Fail(err)
	}
	engine.Register(opencode.NewControlGroup(opencodeSvc))
	engine.Register(opencodeSvc.ProxyGroup())
	engine.Register(brainProvider)

	return engine, core.Ok(nil)
}

// hubLoadOrGenerateToken reads the bearer token at path, or mints a new
// 32-byte hex token and writes it 0600 when absent. Mirrors the
// desktop's apikey.GenerateOrLoad shape (RFC.serve.md §3.2).
func hubLoadOrGenerateToken(fs *core.Fs, path string) (string, core.Result) {
	if fs == nil || core.Trim(path) == "" {
		return "", core.Fail(core.E("hub.token", "fs and token-file path are required", nil))
	}
	if fs.IsFile(path) {
		r := fs.Read(path)
		if !r.OK {
			return "", r
		}
		token := core.Trim(toBytes(r.Value))
		if token == "" {
			return "", core.Fail(core.E("hub.token", "token-file is empty: "+path, nil))
		}
		return token, core.Ok(nil)
	}
	rb := core.RandomBytes(32)
	if !rb.OK {
		return "", rb
	}
	b, ok := rb.Value.([]byte)
	if !ok {
		return "", core.Fail(core.E("hub.token", "random bytes unavailable", nil))
	}
	token := core.HexEncode(b)
	if w := fs.WriteMode(path, token, hubTokenFileMode); !w.OK {
		return "", w
	}
	return token, core.Ok(nil)
}

// laravelURLReject returns a non-empty reason when the brain→Laravel URL
// must be rejected. RFC.serve.md §7.3.4: only a loopback host (any
// scheme) or a wss:// URL (any host) is permitted — a non-loopback ws://
// carries the bearer in cleartext.
//
//	laravelURLReject("ws://localhost:9876/ws")  // ""
//	laravelURLReject("wss://api.lthn.ai/ws")     // ""
//	laravelURLReject("ws://api.lthn.ai/ws")      // "non-loopback ws:// (cleartext bearer)"
func laravelURLReject(raw string) string {
	if core.HasPrefix(raw, "wss://") {
		return ""
	}
	host := laravelHost(raw)
	if hostIsLoopback(host) {
		return ""
	}
	return "non-loopback ws:// (cleartext bearer); use wss:// or a loopback host"
}

// laravelHost extracts the host[:port] from a ws://host:port/path or
// wss://host:port/path URL, stripping any trailing path.
//
//	laravelHost("ws://localhost:9876/ws") // "localhost:9876"
func laravelHost(raw string) string {
	s := core.TrimPrefix(raw, "wss://")
	s = core.TrimPrefix(s, "ws://")
	if idx := core.Index(s, "/"); idx >= 0 {
		s = s[:idx]
	}
	return s
}

// hostIsLoopback reports whether host[:port] binds the loopback
// interface. The literal name "localhost" and any IP that parses into the
// loopback range (127.0.0.0/8 or ::1) count; every other DNS name is
// rejected — a substring "127." test would wrongly accept "127.evil.com"
// and let a config value redirect the hub off-box (SSRF).
//
//	hostIsLoopback("localhost:9876")    // true
//	hostIsLoopback("127.0.0.1:9876")    // true
//	hostIsLoopback("127.0.0.2:9876")    // true  (loopback range)
//	hostIsLoopback("[::1]:9876")        // true
//	hostIsLoopback("127.evil.com:9876") // false (DNS name, not an IP)
//	hostIsLoopback("api.lthn.ai")       // false
func hostIsLoopback(host string) bool {
	h := host
	if core.HasPrefix(h, "[") {
		// Bracketed IPv6, optionally with ":port" after the "]".
		if idx := core.Index(h, "]"); idx >= 0 {
			h = h[1:idx]
		} else {
			h = core.TrimPrefix(h, "[")
		}
	} else if idx := core.Index(h, ":"); idx >= 0 {
		h = h[:idx]
	}
	// The only DNS name that counts as loopback is the literal "localhost";
	// every other name (e.g. "127.evil.com") must be rejected so a config
	// value can't redirect the hub off-box (SSRF). A literal IP counts only
	// if it parses into the loopback range (127.0.0.0/8 or ::1) — a textual
	// "127." prefix would wrongly accept the hostname "127.evil.com".
	if h == "localhost" {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

// defaultHubTokenFile is the default bearer token-file location under the
// core workspace root.
func defaultHubTokenFile() string {
	return core.JoinPath(agentic.CoreRoot(), "hub", "hub.token")
}

// defaultHubAuditPath is the default JSONL audit-edge location under the
// core workspace root.
func defaultHubAuditPath() string {
	return core.JoinPath(agentic.CoreRoot(), "hub", "audit.jsonl")
}

// optStringOr returns the opts value for key, or fallback when empty.
func optStringOr(opts core.Options, key, fallback string) string {
	if v := core.Trim(opts.String(key)); v != "" {
		return v
	}
	return fallback
}

// publicSuffix annotates the bind log line when --public is set.
func publicSuffix(public bool) string {
	if public {
		return ", PUBLIC opt-in"
	}
	return ""
}

// auditMetaString reads a string field from an audit Meta map, returning
// "" when absent or non-string.
func auditMetaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta[key].(string); ok {
		return v
	}
	return ""
}

// toBytes coerces a core.Fs.Read Result value (string or []byte) to a
// string for trimming.
func toBytes(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	}
	return ""
}
