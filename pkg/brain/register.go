// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Brain has no lifecycle hooks — it's a stateless API proxy.
//
//	core.New(
//	    core.WithService(brain.Register),
//	)
func Register(c *core.Core) core.Result {
	brn := NewDirect()

	c.Service("brain", core.Service{})

	// Store instance for MCP tool registration
	c.Config().Set("brain.instance", brn)

	return core.Result{OK: true}
}
