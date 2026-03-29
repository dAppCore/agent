// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/agent/pkg/runner"
	"dappco.re/go/core"
	"forge.lthn.ai/core/mcp/pkg/mcp"
	"github.com/stretchr/testify/assert"
)

func withVersion(t *testing.T, value string) {
	t.Helper()
	oldVersion := version
	version = value
	t.Cleanup(func() { version = oldVersion })
}

func TestMain_NewCoreAgent_Good_RegistersRuntime(t *testing.T) {
	withVersion(t, "0.15.0")

	c := newCoreAgent()

	assert.Equal(t, "core-agent", c.App().Name)
	assert.Equal(t, "0.15.0", c.App().Version)
	assert.Contains(t, c.Services(), "process")
	assert.Contains(t, c.Services(), "agentic")
	assert.Contains(t, c.Services(), "runner")
	assert.Contains(t, c.Services(), "monitor")
	assert.Contains(t, c.Services(), "brain")
	assert.Contains(t, c.Services(), "mcp")
	assert.Contains(t, c.Commands(), "version")
	assert.Contains(t, c.Commands(), "check")
	assert.Contains(t, c.Commands(), "env")
	assert.Contains(t, c.Actions(), "process.run")

	_, ok := core.ServiceFor[*agentic.PrepSubsystem](c, "agentic")
	assert.True(t, ok)
	_, ok = core.ServiceFor[*runner.Service](c, "runner")
	assert.True(t, ok)
	_, ok = core.ServiceFor[*monitor.Subsystem](c, "monitor")
	assert.True(t, ok)
	_, ok = core.ServiceFor[*brain.DirectSubsystem](c, "brain")
	assert.True(t, ok)
	_, ok = core.ServiceFor[*mcp.Service](c, "mcp")
	assert.True(t, ok)
}

func TestMain_NewCoreAgent_Good_BannerUsesVersion(t *testing.T) {
	withVersion(t, "0.15.0")

	c := newCoreAgent()

	assert.Equal(t, "core-agent 0.15.0 — agentic orchestration for the Core ecosystem", c.Cli().Banner())
}

func TestMain_NewCoreAgent_Ugly_DevVersionFallback(t *testing.T) {
	withVersion(t, "")

	c := newCoreAgent()

	assert.Equal(t, "dev", c.App().Version)
	assert.Equal(t, "core-agent dev — agentic orchestration for the Core ecosystem", c.Cli().Banner())
}
