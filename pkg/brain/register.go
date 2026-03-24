// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Brain is a stateless API proxy — no lifecycle hooks.
//
//	core.New(
//	    core.WithService(brain.Register),
//	)
func Register(c *core.Core) core.Result {
	brn := NewDirect()
	c.RegisterService("brain", brn)
	return core.Result{Value: brn, OK: true}
}
