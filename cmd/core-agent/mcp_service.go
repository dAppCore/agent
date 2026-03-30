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
//	r := registerMCPService(c)
func registerMCPService(c *core.Core) core.Result {
	if c == nil {
		return core.Result{Value: core.E("main.registerMCPService", "core is required", nil), OK: false}
	}

	var subsystems []mcp.Subsystem

	if prep, ok := core.ServiceFor[*agentic.PrepSubsystem](c, "agentic"); ok {
		subsystems = append(subsystems, prep)
	}
	if mon, ok := core.ServiceFor[*monitor.Subsystem](c, "monitor"); ok {
		subsystems = append(subsystems, mon)
	}
	if brn, ok := core.ServiceFor[*brain.DirectSubsystem](c, "brain"); ok {
		subsystems = append(subsystems, brn)
	}

	svc, err := mcp.New(mcp.Options{
		Subsystems: subsystems,
	})
	if err != nil {
		return core.Result{Value: core.E("main.registerMCPService", "create mcp service", err), OK: false}
	}
	return core.Result{Value: svc, OK: true}
}
