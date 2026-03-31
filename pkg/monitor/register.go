// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	core "dappco.re/go/core"
)

// c := core.New(core.WithService(monitor.Register))
// service := c.Service("monitor")
// core.Println(service.OK) // true
func Register(c *core.Core) core.Result {
	service := New(Options{})
	service.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	return core.Result{Value: service, OK: true}
}
