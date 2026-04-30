// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	core "dappco.re/go"
)

// c := core.New(core.WithService(monitor.Register))
// service := c.Service("monitor")
// core.Println(service.OK) // true
func Register(c *core.Core) core.Result {
	service := New(MonitorOptions{})
	service.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})
	return core.Result{Value: service, OK: true}
}
