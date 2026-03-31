// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

// c := core.New(
//
//	core.WithService(agentic.ProcessRegister),
//	core.WithService(agentic.Register),
//
// )
// prep := c.Service("agentic")
func Register(c *core.Core) core.Result {
	subsystem := NewPrep()
	subsystem.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	config := subsystem.loadAgentsConfig()
	c.Config().Set("agents.concurrency", config.Concurrency)
	c.Config().Set("agents.rates", config.Rates)
	c.Config().Set("agents.dispatch", config.Dispatch)

	//	c.Config().Enabled("auto-qa")     // true — run QA after completion
	//	c.Config().Enabled("auto-pr")     // true — create PR on QA pass
	//	c.Config().Enabled("auto-merge")  // true — verify + merge PR
	//	c.Config().Enabled("auto-ingest") // true — create issues from findings
	c.Config().Enable("auto-qa")
	c.Config().Enable("auto-pr")
	c.Config().Enable("auto-merge")
	c.Config().Enable("auto-ingest")

	return core.Result{Value: subsystem, OK: true}
}
