// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	core "dappco.re/go/core"
)

// c := core.New(core.WithService(brain.Register))
// subsystem := c.Service("brain")
// core.Println(subsystem.OK) // true
func Register(c *core.Core) core.Result {
	subsystem := NewDirect()
	subsystem.ServiceRuntime = core.NewServiceRuntime(c, directOptions{})
	return core.Result{Value: subsystem, OK: true}
}
