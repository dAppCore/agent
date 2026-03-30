// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	core "dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp"
)

// registerMCPService builds the MCP service from registered AX subsystems.
//
//	c := core.New(core.WithService(registerMCPService))
//	_, ok := core.ServiceFor[*mcp.Service](c, "mcp")
func registerMCPService(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: core.E("main.registerMCPService", "core is required", nil), OK: false}
	}

	var registeredSubsystems []mcp.Subsystem

	if agenticSubsystem, ok := core.ServiceFor[*agentic.PrepSubsystem](c, "agentic"); ok {
		registeredSubsystems = append(registeredSubsystems, agenticSubsystem)
	}
	if monitorSubsystem, ok := core.ServiceFor[*monitor.Subsystem](c, "monitor"); ok {
		registeredSubsystems = append(registeredSubsystems, monitorSubsystem)
	}
	if brainSubsystem, ok := core.ServiceFor[*brain.DirectSubsystem](c, "brain"); ok {
		registeredSubsystems = append(registeredSubsystems, brainSubsystem)
	}

	service, err := mcp.New(mcp.Options{
		Subsystems: registeredSubsystems,
	})
	if err != nil {
		return core.Result{Value: core.E("main.registerMCPService", "create mcp service", err), OK: false}
	}
	return core.Result{Value: service, OK: true}
}
