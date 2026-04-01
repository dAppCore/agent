// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	agentpkg "dappco.re/go/agent"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/agent/pkg/runner"
	"dappco.re/go/agent/pkg/setup"
	"dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp"
	"github.com/stretchr/testify/assert"
)

func withVersion(t *testing.T, value string) {
	t.Helper()
	oldVersion := agentpkg.Version
	agentpkg.Version = value
	t.Cleanup(func() { agentpkg.Version = oldVersion })
}

func TestMain_NewCoreAgent_Good(t *testing.T) {
	withVersion(t, "0.15.0")

	c := newCoreAgent()

	assert.Equal(t, "core-agent", c.App().Name)
	assert.Equal(t, "0.15.0", c.App().Version)
	assert.Contains(t, c.Services(), "process")
	assert.Contains(t, c.Services(), "agentic")
	assert.Contains(t, c.Services(), "runner")
	assert.Contains(t, c.Services(), "monitor")
	assert.Contains(t, c.Services(), "brain")
	assert.Contains(t, c.Services(), "setup")
	assert.Contains(t, c.Services(), "mcp")
	assert.Contains(t, c.Commands(), "version")
	assert.Contains(t, c.Commands(), "check")
	assert.Contains(t, c.Commands(), "env")
	assert.Contains(t, c.Commands(), "mcp")
	assert.Contains(t, c.Commands(), "serve")
	assert.Contains(t, c.Actions(), "process.run")

	service := c.Service("agentic")
	assert.True(t, service.OK)
	assert.IsType(t, &agentic.PrepSubsystem{}, service.Value)
	service = c.Service("runner")
	assert.True(t, service.OK)
	assert.IsType(t, &runner.Service{}, service.Value)
	service = c.Service("monitor")
	assert.True(t, service.OK)
	assert.IsType(t, &monitor.Subsystem{}, service.Value)
	service = c.Service("brain")
	assert.True(t, service.OK)
	assert.IsType(t, &brain.DirectSubsystem{}, service.Value)
	service = c.Service("setup")
	assert.True(t, service.OK)
	assert.IsType(t, &setup.Service{}, service.Value)
	service = c.Service("mcp")
	assert.True(t, service.OK)
	assert.IsType(t, &mcp.Service{}, service.Value)
}

func TestMain_NewCoreAgentBanner_Good(t *testing.T) {
	withVersion(t, "0.15.0")

	c := newCoreAgent()

	assert.Equal(t, "core-agent 0.15.0 — agentic orchestration for the Core ecosystem", c.Cli().Banner())
}

func TestMain_RunApp_Good(t *testing.T) {
	withVersion(t, "0.15.0")

	assert.NoError(t, runApp(newTestCore(t), []string{"version"}))
}

func TestMain_RunApp_Bad(t *testing.T) {
	assert.EqualError(t, runApp(nil, []string{"version"}), "main.runApp: core is required")
}

func TestMain_ResultError_Ugly(t *testing.T) {
	err := resultError("main.runApp", "cli failed", core.Result{})
	assert.EqualError(t, err, "main.runApp: cli failed")
}

func TestMain_NewCoreAgentFallback_Ugly(t *testing.T) {
	withVersion(t, "")

	c := newCoreAgent()

	assert.Equal(t, "dev", c.App().Version)
	assert.Equal(t, "core-agent dev — agentic orchestration for the Core ecosystem", c.Cli().Banner())
}
