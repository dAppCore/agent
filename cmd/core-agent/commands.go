// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go/core"
)

type appCommandSet struct {
	core *core.Core
}

// registerAppCommands adds app-level CLI commands (version, check, env).
// These are not owned by any service — they're the binary's own commands.
//
//	core-agent version   — build info
//	core-agent check     — health check
//	core-agent env       — environment variables
func registerAppCommands(c *core.Core) {
	commands := appCommandSet{core: c}

	c.Command("version", core.Command{
		Description: "Print version and build info",
		Action:      commands.version,
	})

	c.Command("check", core.Command{
		Description: "Verify workspace, deps, and config",
		Action:      commands.check,
	})

	c.Command("env", core.Command{
		Description: "Show all core.Env() keys and values",
		Action:      commands.env,
	})
}

func (commands appCommandSet) version(_ core.Options) core.Result {
	core.Print(nil, "core-agent %s", commands.core.App().Version)
	core.Print(nil, "  go:       %s", core.Env("GO"))
	core.Print(nil, "  os:       %s/%s", core.Env("OS"), core.Env("ARCH"))
	core.Print(nil, "  home:     %s", core.Env("DIR_HOME"))
	core.Print(nil, "  hostname: %s", core.Env("HOSTNAME"))
	core.Print(nil, "  pid:      %s", core.Env("PID"))
	core.Print(nil, "  channel:  %s", updateChannel())
	return core.Result{OK: true}
}

func (commands appCommandSet) check(_ core.Options) core.Result {
	fs := commands.core.Fs()
	core.Print(nil, "core-agent %s health check", commands.core.App().Version)
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

	core.Print(nil, "  services:  %d registered", len(commands.core.Services()))
	core.Print(nil, "  actions:   %d registered", len(commands.core.Actions()))
	core.Print(nil, "  commands:  %d registered", len(commands.core.Commands()))
	core.Print(nil, "  env keys:  %d loaded", len(core.EnvKeys()))
	core.Print(nil, "")
	core.Print(nil, "ok")
	return core.Result{OK: true}
}

func (commands appCommandSet) env(_ core.Options) core.Result {
	keys := core.EnvKeys()
	for _, key := range keys {
		core.Print(nil, "  %-15s %s", key, core.Env(key))
	}
	return core.Result{OK: true}
}
