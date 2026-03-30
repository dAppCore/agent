// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"syscall"

	agentpkg "dappco.re/go/agent"
	"dappco.re/go/core"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/agent/pkg/runner"
)

func main() {
	if err := runCoreAgent(); err != nil {
		core.Error("core-agent failed", "err", err)
		syscall.Exit(1)
	}
}

// newCoreAgent builds the Core app with services and CLI commands wired for startup.
//
//	c := newCoreAgent()
//	core.Println(c.App().Name)    // "core-agent"
//	core.Println(c.App().Version) // "dev" or linked version
func newCoreAgent() *core.Core {
	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
		core.WithService(runner.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
		core.WithName("mcp", registerMCPService),
	)
	c.App().Version = appVersion()

	c.Cli().SetBanner(func(_ *core.Cli) string {
		return core.Concat("core-agent ", c.App().Version, " — agentic orchestration for the Core ecosystem")
	})

	registerAppCommands(c)

	return c
}

// appVersion resolves the build version injected at link time.
//
//	agentpkg.Version = "0.15.0"
//	appVersion() // "0.15.0"
func appVersion() string {
	if agentpkg.Version != "" {
		return agentpkg.Version
	}
	return "dev"
}

// runCoreAgent builds the runtime and executes the CLI with startup flags applied.
//
//	err := runCoreAgent()
func runCoreAgent() error {
	return runApp(newCoreAgent(), startupArgs())
}

// runApp starts services, runs the CLI with explicit args, then shuts down.
//
//	err := runApp(c, []string{"version"})
func runApp(c *core.Core, cliArgs []string) error {
	if c == nil {
		return core.E("main.runApp", "core is required", nil)
	}

	defer c.ServiceShutdown(context.Background())

	result := c.ServiceStartup(c.Context(), nil)
	if !result.OK {
		return resultError("main.runApp", "startup failed", result)
	}

	if cli := c.Cli(); cli != nil {
		result = cli.Run(cliArgs...)
		if !result.OK {
			return resultError("main.runApp", "cli failed", result)
		}
	}

	return nil
}

// resultError extracts the error from a Result or wraps the failure in core.E().
//
//	err := resultError("main.runApp", "startup failed", result)
func resultError(op, msg string, result core.Result) error {
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok && err != nil {
		return err
	}
	return core.E(op, msg, nil)
}
