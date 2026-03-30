// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	core "dappco.re/go/core"
)

// Register exposes the direct OpenBrain subsystem through `core.WithService`.
//
//	c := core.New(core.WithService(brain.Register))
//	subsystem, _ := core.ServiceFor[*brain.DirectSubsystem](c, "brain")
//	core.Println(subsystem.Name()) // "brain"
func Register(c *core.Core) core.Result {
	subsystem := NewDirect()
	return core.Result{Value: subsystem, OK: true}
}
