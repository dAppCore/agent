// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_RegisterMCPService_Good(t *testing.T) {
	result := registerMCPService(core.New(core.WithOption("name", "core-agent")))

	require.True(t, result.OK)
	_, ok := result.Value.(*mcp.Service)
	assert.True(t, ok)
}

func TestMCP_RegisterMCPService_Good_WithRegisteredSubsystems(t *testing.T) {
	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
	)

	result := registerMCPService(c)

	require.True(t, result.OK)
	service := result.Value.(*mcp.Service)
	assert.Len(t, service.Subsystems(), 3)
}
