// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	core "dappco.re/go/core"
)

// Register wires the monitor service into Core and lets HandleIPCEvents auto-register.
//
//	c := core.New(core.WithService(monitor.Register))
//	mon, _ := core.ServiceFor[*monitor.Subsystem](c, "monitor")
func Register(c *core.Core) core.Result {
	monitorService := New()
	monitorService.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	return core.Result{Value: monitorService, OK: true}
}
