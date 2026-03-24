// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Creates the monitor subsystem, registers via RegisterService,
// and wires IPC handlers for agent lifecycle events.
//
//	core.New(
//	    core.WithService(monitor.Register),
//	)
func Register(c *core.Core) core.Result {
	mon := New()
	mon.core = c

	c.RegisterService("monitor", mon)

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
