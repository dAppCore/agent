// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	"dappco.re/go"
	agentpkg "dappco.re/go/agent"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/agent/pkg/runner"
	"dappco.re/go/agent/pkg/setup"
	"dappco.re/go/mcp/pkg/mcp"
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

	core.AssertEqual(t, "core-agent", c.App().Name)
	core.AssertEqual(t, "0.15.0", c.App().Version)
	core.AssertContains(t, c.Services(), "process")
	core.AssertContains(t, c.Services(), "agentic")
	core.AssertContains(t, c.Services(), "runner")
	core.AssertContains(t, c.Services(), "monitor")
	core.AssertContains(t, c.Services(), "brain")
	core.AssertContains(t, c.Services(), "setup")
	core.AssertContains(t, c.Services(), "mcp")
	core.AssertContains(t, c.Commands(), "version")
	core.AssertContains(t, c.Commands(), "check")
	core.AssertContains(t, c.Commands(), "env")
	core.AssertContains(t, c.Actions(), "process.run")

	service := c.Service("agentic")
	core.AssertTrue(t, service.OK)
	assertIsType(t, &agentic.PrepSubsystem{}, service.Value)
	service = c.Service("runner")
	core.AssertTrue(t, service.OK)
	assertIsType(t, &runner.Service{}, service.Value)
	service = c.Service("monitor")
	core.AssertTrue(t, service.OK)
	assertIsType(t, &monitor.Subsystem{}, service.Value)
	service = c.Service("brain")
	core.AssertTrue(t, service.OK)
	assertIsType(t, &brain.DirectSubsystem{}, service.Value)
	service = c.Service("setup")
	core.AssertTrue(t, service.OK)
	assertIsType(t, &setup.Service{}, service.Value)
	service = c.Service("mcp")
	core.AssertTrue(t, service.OK)
	assertIsType(t, &mcp.Service{}, service.Value)
}

func TestMain_NewCoreAgentBanner_Good(t *testing.T) {
	withVersion(t, "0.15.0")

	c := newCoreAgent()

	core.AssertEqual(t, "core-agent 0.15.0 — agentic orchestration for the Core ecosystem", c.Cli().Banner())
}

func TestMain_RunApp_Good(t *testing.T) {
	withVersion(t, "0.15.0")

	err := runApp(newTestCore(t), []string{"version"})
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestMain_RunApp_Bad(t *testing.T) {
	err := runApp(nil, []string{"version"})
	core.AssertError(t, err, "main.runApp: core is required")
	core.AssertContains(t, err.Error(), "core is required")
}

func TestMain_ResultError_Ugly(t *testing.T) {
	err := resultError("main.runApp", "cli failed", core.Result{})
	core.AssertError(t, err, "main.runApp: cli failed")
	core.AssertContains(t, err.Error(), "cli failed")
}

func TestMain_NewCoreAgentFallback_Ugly(t *testing.T) {
	withVersion(t, "")

	c := newCoreAgent()

	core.AssertEqual(t, "dev", c.App().Version)
	core.AssertEqual(t, "core-agent dev — agentic orchestration for the Core ecosystem", c.Cli().Banner())
}
