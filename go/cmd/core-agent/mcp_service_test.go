// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"github.com/dappcore/agent/go/pkg/agentic"
	"github.com/dappcore/agent/go/pkg/brain"
	"github.com/dappcore/agent/go/pkg/monitor"
	"github.com/dappcore/go"
	"github.com/dappcore/mcp/go/pkg/mcp"
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
