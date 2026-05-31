// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"

	"dappco.re/go"
	agentpkg "dappco.re/go/agent"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/agent/pkg/opencode"
	"dappco.re/go/agent/pkg/runner"
	"dappco.re/go/agent/pkg/setup"
	coremcp "dappco.re/go/mcp/pkg/mcp"
)

func main() {
	if err := runCoreAgent(); err != nil {
		core.Error(core.Concat(detectBinaryName(), " failed"), "err", err)
		core.Exit(1)
	}
}

// detectBinaryName returns the basename of os.Args[0] so the same
// source ships as either `core-agent` or as any sibling in the
// lthn-{mlx,cuda,amd,agent} binary family (per
// project/lthn/RFC.system-architecture.md). Empty / unrecognised
// argv[0] falls back to "core-agent" — the legacy default.
//
//	core-agent              → "core-agent"
//	/usr/local/bin/lthn-agent → "lthn-agent"
func detectBinaryName() string {
	args := core.Args()
	if len(args) == 0 {
		return "core-agent"
	}
	if base := core.PathBase(args[0]); base != "" {
		return base
	}
	return "core-agent"
}

// app := newCoreAgent()
// core.Println(app.App().Name)    // "core-agent"
// core.Println(app.App().Version) // "dev" or linked version
func newCoreAgent() *core.Core {
	coreApp, result := newCoreAgentResult()
	if !result.OK {
		panic(result.Value)
	}
	return coreApp
}

func newCoreAgentResult() (*core.Core, core.Result) {
	coreApp := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
		core.WithService(runner.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
		core.WithName("opencode", opencode.NewService(opencode.Options{})),
		core.WithService(setup.Register),
		core.WithService(registerLemmaSubsystem),
		core.WithService(coremcp.Register),
	)
	coreApp.App().Version = applicationVersion()

	coreApp.Cli().SetBanner(func(_ *core.Cli) string {
		return core.Concat("core-agent ", coreApp.App().Version, " — agentic orchestration for the Core ecosystem")
	})

	if result := registerApplicationCommands(coreApp); !result.OK {
		return coreApp, result
	}

	return coreApp, core.Result{OK: true}
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
var runCoreAgent = func() error {
	coreApp, result := newCoreAgentResult()
	if !result.OK {
		return resultError("main.newCoreAgent", "command registration failed", result)
	}
	// Override the in-process name + banner with the invoked binary
	// name so the same source ships as core-agent or any lthn-agent
	// sibling without per-binary main.go duplication. Test paths use
	// newCoreAgent()/newCoreAgentResult() directly and keep the
	// canonical "core-agent" name unchanged.
	binaryName := detectBinaryName()
	coreApp.App().Name = binaryName
	coreApp.Cli().SetBanner(func(_ *core.Cli) string {
		return core.Concat(binaryName, " ", coreApp.App().Version, " — agentic orchestration for the Core ecosystem")
	})
	return runApp(coreApp, startupArgs())
}

// app := newCoreAgent()
// result := runApp(app, []string{"version"})
var runApp = func(coreApp *core.Core, cliArgs []string) error {
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
var resultError = func(op, msg string, result core.Result) error {
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok && err != nil {
		return err
	}
	return core.E(op, msg, nil)
}
