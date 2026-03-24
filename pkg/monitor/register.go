// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// It creates the monitor subsystem, wires Core for IPC,
// and registers lifecycle hooks.
//
//	core.New(
//	    core.WithService(monitor.Register),
//	)
func Register(c *core.Core) core.Result {
	mon := New()
	mon.core = c

	c.Service("monitor", core.Service{
		OnStart: func() core.Result {
			mon.Start(c.Context())
			return core.Result{OK: true}
		},
	})

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

	return core.Result{OK: true}
}
