// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Returns the monitor Subsystem — WithService auto-registers it.
//
//	core.New(
//	    core.WithService(monitor.Register),
//	)
func Register(c *core.Core) core.Result {
	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})

	// Register IPC handler for agent lifecycle events
	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
		switch ev := msg.(type) {
		case messages.AgentCompleted:
			mon.handleAgentCompleted(ev)
		case messages.AgentStarted:
			mon.handleAgentStarted(ev)
		}
		return core.Result{OK: true}
	})

	return core.Result{Value: mon, OK: true}
}
