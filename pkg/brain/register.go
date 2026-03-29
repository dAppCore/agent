// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	core "dappco.re/go/core"
)

// Register exposes the direct OpenBrain subsystem through core.WithService.
//
//	c := core.New(core.WithService(brain.Register))
//	sub, _ := core.ServiceFor[*brain.DirectSubsystem](c, "brain")
func Register(c *core.Core) core.Result {
	brn := NewDirect()
	return core.Result{Value: brn, OK: true}
}
