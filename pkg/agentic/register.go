// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Returns the PrepSubsystem instance — WithService auto-discovers the name
// from the package path and registers it.
//
//	core.New(
//	    core.WithService(agentic.ProcessRegister),
//	    core.WithService(agentic.Register),
//	)
func Register(c *core.Core) core.Result {
	subsystem := NewPrep()
	subsystem.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	// Load agents config once into Core shared config
	config := subsystem.loadAgentsConfig()
	c.Config().Set("agents.concurrency", config.Concurrency)
	c.Config().Set("agents.rates", config.Rates)
	c.Config().Set("agents.dispatch", config.Dispatch)

	// Pipeline feature flags — all enabled by default.
	// Disable with c.Config().Disable("auto-qa") etc.
	//
	//	c.Config().Enabled("auto-qa")     // true — run QA after completion
	//	c.Config().Enabled("auto-pr")     // true — create PR on QA pass
	//	c.Config().Enabled("auto-merge")  // true — verify + merge PR
	//	c.Config().Enabled("auto-ingest") // true — create issues from findings
	c.Config().Enable("auto-qa")
	c.Config().Enable("auto-pr")
	c.Config().Enable("auto-merge")
	c.Config().Enable("auto-ingest")

	// IPC handlers auto-discovered via HandleIPCEvents interface on PrepSubsystem.
	// No manual RegisterHandlers call needed — WithService wires it.

	return core.Result{Value: subsystem, OK: true}
}
