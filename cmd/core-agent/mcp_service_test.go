// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/core"
	"dappco.re/go/mcp/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_Register_Good(t *testing.T) {
	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(mcp.Register),
	)

	result := c.Service("mcp")

	require.True(t, result.OK)
	_, ok := result.Value.(*mcp.Service)
	assert.True(t, ok)
}

func TestMCP_Register_Bad(t *testing.T) {
	c := core.New(core.WithOption("name", "core-agent"))

	result := c.Service("mcp")

	assert.False(t, result.OK)
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

	require.True(t, result.OK)
	service := result.Value.(*mcp.Service)
	assert.Len(t, service.Subsystems(), 3)
}
