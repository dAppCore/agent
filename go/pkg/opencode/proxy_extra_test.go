// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// TestProxy_SandboxProxyGroup — the sandbox reverse-proxy registry installs,
// reports, and drops forwarding targets; invalid target URLs are ignored.
func TestProxy_SandboxProxyGroup(t *testing.T) {
	g := NewSandboxProxyGroup()
	core.AssertEqual(t, "sandbox", g.Name())
	core.AssertEqual(t, "/v1/api/sandbox", g.BasePath())

	core.AssertFalse(t, g.Has("oc-1"))
	g.Set("oc-1", "http://127.0.0.1:51823", "")
	core.AssertTrue(t, g.Has("oc-1"))
	g.Set("oc-2", "http://127.0.0.1:51824", "Basic xyz")
	core.AssertTrue(t, g.Has("oc-2"))
	g.Set("oc-3", "://bad-url", "")
	core.AssertFalse(t, g.Has("oc-3"))
	g.Delete("oc-1")
	core.AssertFalse(t, g.Has("oc-1"))

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	g.RegisterRoutes(engine.Group(""))
	core.AssertTrue(t, len(engine.Routes()) > 0)
}
