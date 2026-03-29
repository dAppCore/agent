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
	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})
	return core.Result{Value: mon, OK: true}
}
