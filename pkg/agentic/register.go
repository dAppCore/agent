// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Returns the PrepSubsystem instance — WithService auto-discovers the name
// from the package path and registers it. Startable/Stoppable/HandleIPCEvents
// are auto-discovered by RegisterService.
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

	RegisterHandlers(c, prep)

	return core.Result{Value: prep, OK: true}
}
