// SPDX-Licence-Identifier: EUPL-1.2

// Reverse-proxy mount — a single coreapi.RouteGroup registered
// once at boot. Internally it holds a sandbox-id → ReverseProxy
// table that mutates as opencode sandboxes Start / Stop. Mirrors
// pkg/plugin's ProxyGroup shape; differs in path semantics — we
// strip the /v1/api/sandbox/<id>/ prefix entirely so the upstream
// (opencode-serve) sees clean paths like /global/health, /session.

package opencode

import (
	"net/http/httputil"
	"net/url"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// SandboxProxyGroup implements coreapi.RouteGroup. Registered exactly
// once on the coreapi.Engine; the targets map mutates at runtime as
// opencode sandboxes Start / Stop.
type SandboxProxyGroup struct {
	mu      core.RWMutex
	targets map[string]*httputil.ReverseProxy // keyed by sandbox id
}

// NewSandboxProxyGroup constructs an empty proxy group.
//
// Usage example:
//
//	g := opencode.NewSandboxProxyGroup()
//	engine.Register(g)              // mount /v1/api/sandbox/* once at boot
//	g.Set("oc-7f3a2b1c", "http://127.0.0.1:51823")
func NewSandboxProxyGroup() *SandboxProxyGroup {
	return &SandboxProxyGroup{targets: map[string]*httputil.ReverseProxy{}}
}

// Name satisfies coreapi.RouteGroup. Surfaces in /v1/openapi.
func (g *SandboxProxyGroup) Name() string { return "sandbox" }

// BasePath satisfies coreapi.RouteGroup. All sandbox routes mount
// under /v1/api/sandbox/.
func (g *SandboxProxyGroup) BasePath() string { return "/v1/api/sandbox" }

// RegisterRoutes satisfies coreapi.RouteGroup. The wildcard pattern
// captures `:id/*proxyPath` so the dispatcher can look the target
// up and forward.
//
// Path semantics differ from pkg/plugin: opencode-serve is content
// with clean paths, so we strip /v1/api/sandbox/<id>/ entirely
// before forwarding. The container sees /global/health, /session,
// /provider — never the sandbox-id namespace.
func (g *SandboxProxyGroup) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Any("/:id/*proxyPath", g.dispatch)
}

// Set installs a forwarding target for one sandbox id. Called from
// Service.Start() once the container is healthy. targetURL is
// `http://127.0.0.1:<host-port>` where host-port is the dynamic
// port allocated for this sandbox.
//
// authHeader is the optional Authorization header value injected on
// every forwarded request — opencode-serve enforces HTTP Basic Auth
// when OPENCODE_SERVER_PASSWORD is set, and the reverse-proxy is the
// canonical place to attach the credential so callers (frontend +
// CLI clients) don't need to know the password.
//
// Usage example:
//
//	g.Set("oc-7f3a2b1c", "http://127.0.0.1:51823", svc.authHeader())
func (g *SandboxProxyGroup) Set(id, targetURL, authHeader string) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return
	}
	rp := httputil.NewSingleHostReverseProxy(u)
	// SSE-friendly: httputil.ReverseProxy's default ServeHTTP
	// flushes streaming responses (no Buffered field — flush happens
	// when downstream Writer implements core.Flusher, which gin's
	// ResponseWriter does). No customisation needed for SSE today.
	if authHeader != "" {
		// Rewrite, not Director: Director is deprecated as of Go 1.26 and
		// superseded since 1.20. Rewrite is also the safer of the two — it
		// hands the hook both the inbound and outbound request, so headers
		// arriving from the client cannot be forwarded upstream by accident,
		// which is the hazard Director was replaced for.
		//
		// SetURL reproduces what NewSingleHostReverseProxy's own Director did
		// (scheme, host, path join), and SetXForwarded restores the
		// X-Forwarded-* handling that came with it.
		rp.Rewrite = func(pr *httputil.ProxyRequest) {
			pr.SetURL(u)
			pr.Out.Host = pr.In.Host
			pr.SetXForwarded()
			pr.Out.Header.Set("Authorization", authHeader)
		}
	}
	g.mu.Lock()
	g.targets[id] = rp
	g.mu.Unlock()
}

// Delete drops a sandbox's forwarding entry. Subsequent requests
// to /v1/api/sandbox/<id>/* return 404 with a helpful hint.
//
// Usage example:
//
//	g.Delete("oc-7f3a2b1c")
func (g *SandboxProxyGroup) Delete(id string) {
	g.mu.Lock()
	delete(g.targets, id)
	g.mu.Unlock()
}

// Has reports whether a sandbox is currently mounted.
//
// Usage example:
//
//	if g.Has("oc-7f3a2b1c") { ... }
func (g *SandboxProxyGroup) Has(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.targets[id]
	return ok
}

// dispatch looks the target up by URL param and forwards. The path
// passed to the proxy is *proxyPath (the part after /v1/api/sandbox/<id>),
// so the upstream container sees /global/health, /session/<id>, etc.
//
// The forwarded proxyPath is rejected before it reaches the upstream if
// it carries a "../" traversal segment or a non-printable byte
// (RFC.serve.md §7.3.3). The hub bearer is container-exec-equivalent
// (§7.3.2) — the proxy injects opencode-serve's full credential
// downstream — so a traversal that escaped the sandbox-id namespace
// would be an authenticated reach past the intended surface.
func (g *SandboxProxyGroup) dispatch(c *gin.Context) {
	id := core.TrimCutset(c.Param("id"), "/ ")

	proxyPath := c.Param("proxyPath")
	if reason := proxyPathReject(proxyPath); reason != "" {
		emitControlAudit(EventOpencodeSandboxProxy, "opencode.sandbox.proxy",
			outcomeDenied, newRequestID(), map[string]any{
				"sandbox_id":  id,
				"path_prefix": proxyPathPrefix(proxyPath),
				"error_code":  reason,
			})
		c.JSON(core.StatusBadRequest, gin.H{
			"error": "invalid proxy path",
			"hint":  reason,
		})
		return
	}

	g.mu.RLock()
	rp, ok := g.targets[id]
	g.mu.RUnlock()
	if !ok {
		c.JSON(core.StatusNotFound, gin.H{
			"error": "sandbox not running: " + id,
			"hint":  "start a sandbox via `lthn opencode start` or the Integrations panel",
		})
		return
	}
	// gin's "*proxyPath" wildcard includes the leading slash, e.g.
	// "/global/health". Rewriting Request.URL.Path strips the
	// /v1/api/sandbox/<id> prefix entirely.
	c.Request.URL.Path = proxyPath
	emitControlAudit(EventOpencodeSandboxProxy, "opencode.sandbox.proxy",
		outcomeOK, newRequestID(), map[string]any{
			"sandbox_id":  id,
			"path_prefix": proxyPathPrefix(proxyPath),
		})
	rp.ServeHTTP(c.Writer, c.Request)
}

// proxyPathReject returns a non-empty reason string when the forwarded
// proxyPath must be rejected: a "../" / "/.." / "/../" traversal
// segment, or a non-printable / control byte. An empty return means the
// path is safe to forward.
//
//	proxyPathReject("/global/health") // ""
//	proxyPathReject("/../secret")      // "path_traversal"
//	proxyPathReject("/a\x00b")         // "non_printable"
func proxyPathReject(p string) string {
	if core.Contains(p, "..") {
		return "path_traversal"
	}
	for _, b := range core.AsBytes(p) {
		if b < 0x20 || b == 0x7f {
			return "non_printable"
		}
	}
	return ""
}

// proxyPathPrefix returns the leading path segment for the audit record
// — never the full path (which can carry session ids / query material),
// only the prefix that identifies the upstream surface.
//
//	proxyPathPrefix("/global/health") // "/global"
//	proxyPathPrefix("/session/abc")   // "/session"
//	proxyPathPrefix("/")              // "/"
func proxyPathPrefix(p string) string {
	trimmed := core.TrimPrefix(p, "/")
	if trimmed == "" {
		return "/"
	}
	if idx := core.Index(trimmed, "/"); idx >= 0 {
		return "/" + trimmed[:idx]
	}
	return "/" + trimmed
}
