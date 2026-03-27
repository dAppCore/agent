package main

import (
	"os"

	"dappco.re/go/core"
)

// applyLogLevel scans os.Args for --quiet/-q/--debug before Core starts.
// Must run before c.Run() so ServiceStartup logs respect the level.
//
//	core-agent --quiet version   → errors only
//	core-agent --debug mcp       → full debug output
//	core-agent status            → default (info)
func applyLogLevel() {
	var cleaned []string
	cleaned = append(cleaned, os.Args[0])
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--quiet", "-q":
			core.SetLevel(core.LevelError)
		case "--debug", "-d":
			core.SetLevel(core.LevelDebug)
		default:
			cleaned = append(cleaned, arg)
		}
	}
	os.Args = cleaned
}

// registerAppCommands adds app-level CLI commands (version, check, env).
// These are not owned by any service — they're the binary's own commands.
//
//	core-agent version   — build info
//	core-agent check     — health check
//	core-agent env       — environment variables
func registerAppCommands(c *core.Core) {
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
			core.Print(nil, "  binary:    core-agent")

			agentsPath := core.Path("Code", ".core", "agents.yaml")
			if fs.IsFile(agentsPath) {
				core.Print(nil, "  agents:    %s (ok)", agentsPath)
			} else {
				core.Print(nil, "  agents:    %s (MISSING)", agentsPath)
			}

			wsRoot := core.Path("Code", ".core", "workspace")
			if fs.IsDir(wsRoot) {
				entries := core.PathGlob(core.JoinPath(wsRoot, "*"))
				core.Print(nil, "  workspace: %s (%d entries)", wsRoot, len(entries))
			} else {
				core.Print(nil, "  workspace: %s (MISSING)", wsRoot)
			}

			core.Print(nil, "  services:  %d registered", len(c.Services()))
			core.Print(nil, "  actions:   %d registered", len(c.Actions()))
			core.Print(nil, "  commands:  %d registered", len(c.Commands()))
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
}
