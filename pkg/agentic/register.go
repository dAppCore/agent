// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// It creates the PrepSubsystem, wires Core, registers lifecycle hooks,
// and registers IPC handlers — all during Core construction.
//
//	core.New(
//	    core.WithService(agentic.Register),
//	)
func Register(c *core.Core) core.Result {
	prep := NewPrep()
	prep.core = c

	c.Service("agentic", core.Service{
		OnStart: func() core.Result {
			prep.StartRunner()
			return core.Result{OK: true}
		},
		OnStop: func() core.Result {
			prep.frozen = true
			return core.Result{OK: true}
		},
	})

	RegisterHandlers(c, prep)

	return core.Result{OK: true}
}
