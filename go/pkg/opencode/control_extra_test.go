// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"testing"

	core "dappco.re/go"
	"github.com/gin-gonic/gin"
)

// TestControl_NewControlGroup_Good — the control group reports its route name
// and base path.
func TestControl_NewControlGroup_Good(t *testing.T) {
	g := NewControlGroup(&Service{})
	core.AssertEqual(t, "opencode", g.Name())
	core.AssertEqual(t, "/v1/api/opencode", g.BasePath())
}

// TestControl_RegisterRoutes_Good — RegisterRoutes wires the sandbox + profile
// routes onto the engine.
func TestControl_RegisterRoutes_Good(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g := NewControlGroup(&Service{})
	engine := gin.New()
	g.RegisterRoutes(engine.Group(""))
	core.AssertTrue(t, len(engine.Routes()) > 0)
}
