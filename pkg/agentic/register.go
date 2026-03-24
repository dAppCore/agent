// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Creates the PrepSubsystem, registers it via RegisterService (auto-discovers
// Startable/Stoppable), loads config, and wires IPC handlers.
//
//	core.New(
//	    core.WithService(agentic.Register),
//	)
func Register(c *core.Core) core.Result {
	prep := NewPrep()
	prep.core = c

	// Load agents config once into Core shared config
	cfg := prep.loadAgentsConfig()
	c.Config().Set("agents.concurrency", cfg.Concurrency)
	c.Config().Set("agents.rates", cfg.Rates)
	c.Config().Set("agents.dispatch", cfg.Dispatch)

	// Register instance — lifecycle hooks wired via Startable/Stoppable if implemented
	c.RegisterService("agentic", prep)

	RegisterHandlers(c, prep)

	return core.Result{Value: prep, OK: true}
}
