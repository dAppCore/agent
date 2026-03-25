package main

import (
	"os"

	"dappco.re/go/core"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"dappco.re/go/mcp/pkg/mcp"
)

func main() {
	c := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.ProcessRegister),
		core.WithService(agentic.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
		core.WithService(mcp.Register),
	)

	// Version set at build time: go build -ldflags "-X main.version=0.15.0"
	if version != "" {
		c.App().Version = version
	} else {
		c.App().Version = "dev"
	}

	// App-level commands (not owned by any service)
	c.Command("version", core.Command{
		Description: "Print version and build info",
		Action: func(opts core.Options) core.Result {
			core.Print(nil, "core-agent %s", c.App().Version)
			core.Print(nil, "  go:       %s", core.Env("GO"))
			core.Print(nil, "  os:       %s/%s", core.Env("OS"), core.Env("ARCH"))
			core.Print(nil, "  home:     %s", core.Env("DIR_HOME"))
			core.Print(nil, "  hostname: %s", core.Env("HOSTNAME"))
			core.Print(nil, "  pid:      %s", core.Env("PID"))
			core.Print(nil, "  channel:  %s", updateChannel())
			return core.Result{OK: true}
		},
	})

	c.Command("check", core.Command{
		Description: "Verify workspace, deps, and config",
		Action: func(opts core.Options) core.Result {
			fs := c.Fs()
			core.Print(nil, "core-agent %s health check", c.App().Version)
			core.Print(nil, "")
			core.Print(nil, "  binary:    %s", os.Args[0])

			agentsPath := core.Path("Code", ".core", "agents.yaml")
			if fs.IsFile(agentsPath) {
				core.Print(nil, "  agents:    %s (ok)", agentsPath)
			} else {
				core.Print(nil, "  agents:    %s (MISSING)", agentsPath)
			}

			wsRoot := core.Path("Code", ".core", "workspace")
			if fs.IsDir(wsRoot) {
				r := fs.List(wsRoot)
				count := 0
				if r.OK {
					count = len(r.Value.([]os.DirEntry))
				}
				core.Print(nil, "  workspace: %s (%d entries)", wsRoot, count)
			} else {
				core.Print(nil, "  workspace: %s (MISSING)", wsRoot)
			}

			core.Print(nil, "  core:      dappco.re/go/core@v%s", c.App().Version)
			core.Print(nil, "  env keys:  %d loaded", len(core.EnvKeys()))
			core.Print(nil, "")
			core.Print(nil, "ok")
			return core.Result{OK: true}
		},
	})

	c.Command("env", core.Command{
		Description: "Show all core.Env() keys and values",
		Action: func(opts core.Options) core.Result {
			keys := core.EnvKeys()
			for _, k := range keys {
				core.Print(nil, "  %-15s %s", k, core.Env(k))
			}
			return core.Result{OK: true}
		},
	})

	// All commands registered by services during OnStartup
	// registerFlowCommands(c) — on feat/flow-system branch

	// Run: ServiceStartup → Cli → ServiceShutdown → os.Exit if error
	c.Run()
}
