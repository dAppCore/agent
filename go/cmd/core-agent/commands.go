// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
)

type applicationCommandSet struct {
	coreApp *core.Core
}

var startupArgv = core.Args

var applicationPrint = func(format string, args ...any) {
	core.Print(nil, format, args...)
}

// args := startupArgs()
// _ = c.Cli().Run("version")
func startupArgs() []string {
	return applyLogLevel(core.FilterArgs(startupArgv()[1:]))
}

// args := applyLogLevel([]string{"version", "-q"})
// args := applyLogLevel([]string{"--debug", "mcp"})
func applyLogLevel(args []string) []string {
	var cleaned []string
	for _, arg := range args {
		switch arg {
		case "--quiet", "-q":
			core.SetLevel(core.LevelError)
		case "--debug", "-d":
			core.SetLevel(core.LevelDebug)
		default:
			cleaned = append(cleaned, arg)
		}
	}
	return cleaned
}

// result := registerApplicationCommands(c)
// core.Println(result.OK)
func registerApplicationCommands(c *core.Core) core.Result {
	commands := applicationCommandSet{coreApp: c}

	if result := c.Command("version", core.Command{
		Description: "Print version and build info",
		Action:      commands.version,
	}); !result.OK {
		return result
	}

	if result := c.Command("check", core.Command{
		Description: "Verify workspace, deps, and config",
		Action:      commands.check,
	}); !result.OK {
		return result
	}

	if result := c.Command("env", core.Command{
		Description: "Show all core.Env() keys and values",
		Action:      commands.env,
	}); !result.OK {
		return result
	}
	return core.Result{OK: true}
}

func (commands applicationCommandSet) version(_ core.Options) core.Result {
	applicationPrint("core-agent %s", commands.coreApp.App().Version)
	applicationPrint("  go:       %s", core.Env("GO"))
	applicationPrint("  os:       %s/%s", core.Env("OS"), core.Env("ARCH"))
	applicationPrint("  home:     %s", agentic.HomeDir())
	applicationPrint("  hostname: %s", core.Env("HOSTNAME"))
	applicationPrint("  pid:      %s", core.Env("PID"))
	applicationPrint("  channel:  %s", updateChannel())
	return core.Result{OK: true}
}

func (commands applicationCommandSet) check(_ core.Options) core.Result {
	fs := commands.coreApp.Fs()
	applicationPrint("core-agent %s health check", commands.coreApp.App().Version)
	applicationPrint("")
	applicationPrint("  binary:    core-agent")

	agentsPath := core.JoinPath(agentic.CoreRoot(), "agents.yaml")
	if fs.IsFile(agentsPath) {
		applicationPrint("  agents:    %s (ok)", agentsPath)
	} else {
		applicationPrint("  agents:    %s (MISSING)", agentsPath)
	}

	workspaceRoot := agentic.WorkspaceRoot()
	if fs.IsDir(workspaceRoot) {
		statusFiles := agentic.WorkspaceStatusPaths()
		applicationPrint("  workspace: %s (%d workspaces)", workspaceRoot, len(statusFiles))
	} else {
		applicationPrint("  workspace: %s (MISSING)", workspaceRoot)
	}

	applicationPrint("  services:  %d registered", len(commands.coreApp.Services()))
	applicationPrint("  actions:   %d registered", len(commands.coreApp.Actions()))
	applicationPrint("  commands:  %d registered", len(commands.coreApp.Commands()))
	applicationPrint("  env keys:  %d loaded", len(core.EnvKeys()))
	applicationPrint("")
	applicationPrint("ok")
	return core.Result{OK: true}
}

func (commands applicationCommandSet) env(_ core.Options) core.Result {
	keys := core.EnvKeys()
	for _, key := range keys {
		applicationPrint("  %-15s %s", key, core.Env(key))
	}
	return core.Result{OK: true}
}
