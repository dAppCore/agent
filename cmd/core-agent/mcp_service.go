// SPDX-License-Identifier: EUPL-1.2

package main

import (
	core "dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp"
)

// c := core.New(core.WithService(registerMCPService))
// service := c.Service("mcp")
func registerMCPService(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: core.E("main.registerMCPService", "core is required", nil), OK: false}
	}

	var registeredSubsystems []mcp.Subsystem

	appendSubsystem := func(name string) {
		serviceResult := c.Service(name)
		if !serviceResult.OK {
			return
		}
		subsystem, ok := serviceResult.Value.(mcp.Subsystem)
		if !ok {
			return
		}
		registeredSubsystems = append(registeredSubsystems, subsystem)
	}
	appendSubsystem("agentic")
	appendSubsystem("monitor")
	appendSubsystem("brain")

	service, err := mcp.New(mcp.Options{
		Subsystems: registeredSubsystems,
	})
	if err != nil {
		return core.Result{Value: core.E("main.registerMCPService", "create mcp service", err), OK: false}
	}
	return core.Result{Value: service, OK: true}
}
