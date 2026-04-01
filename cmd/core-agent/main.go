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
	"dappco.re/go/agent/pkg/setup"
)

func main() {
	if err := runCoreAgent(); err != nil {
		core.Error("core-agent failed", "err", err)
		syscall.Exit(1)
	}
}

// app := newCoreAgent()
// core.Println(app.App().Name)    // "core-agent"
// core.Println(app.App().Version) // "dev" or linked version
func newCoreAgent() *core.Core {
	coreApp := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
		core.WithService(runner.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
		core.WithService(setup.Register),
		core.WithService(registerMCPService),
	)
	coreApp.App().Version = applicationVersion()

	coreApp.Cli().SetBanner(func(_ *core.Cli) string {
		return core.Concat("core-agent ", coreApp.App().Version, " — agentic orchestration for the Core ecosystem")
	})

	registerApplicationCommands(coreApp)

	return coreApp
}

// agentpkg.Version = "0.15.0"
// applicationVersion() // "0.15.0"
func applicationVersion() string {
	if agentpkg.Version != "" {
		return agentpkg.Version
	}
	return "dev"
}

//	if err := runCoreAgent(); err != nil {
//		core.Error("core-agent failed", "err", err)
//	}
func runCoreAgent() error {
	return runApp(newCoreAgent(), startupArgs())
}

// app := newCoreAgent()
// _ = runApp(app, []string{"version"})
func runApp(coreApp *core.Core, cliArgs []string) error {
	if coreApp == nil {
		return core.E("main.runApp", "core is required", nil)
	}

	defer coreApp.ServiceShutdown(context.Background())

	result := coreApp.ServiceStartup(coreApp.Context(), nil)
	if !result.OK {
		return resultError("main.runApp", "startup failed", result)
	}

	if cli := coreApp.Cli(); cli != nil {
		result = cli.Run(cliArgs...)
		if !result.OK {
			return resultError("main.runApp", "cli failed", result)
		}
	}

	return nil
}

// result := core.Result{OK: false, Value: core.E("main.runApp", "startup failed", nil)}
// err := resultError("main.runApp", "startup failed", result)
func resultError(op, msg string, result core.Result) error {
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok && err != nil {
		return err
	}
	return core.E(op, msg, nil)
}
