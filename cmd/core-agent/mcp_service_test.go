// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/mcp/pkg/mcp"
)

func TestMCP_Register_Good(t *testing.T) {
	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(mcp.Register),
	)

	result := c.Service("mcp")

	core.RequireTrue(t, result.OK)
	_, ok := result.Value.(*mcp.Service)
	core.AssertTrue(t, ok)
}

func TestMCP_Register_Bad_Case(t *testing.T) {
	c := core.New(core.WithOption("name", "core-agent"))

	result := c.Service("mcp")

	core.AssertFalse(t, result.OK)
}

func TestMCP_Register_Ugly(t *testing.T) {
	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
		core.WithService(mcp.Register),
	)

	result := c.Service("mcp")

	core.RequireTrue(t, result.OK)
	service := result.Value.(*mcp.Service)
	core.AssertLen(t, service.Subsystems(), 3)
}
