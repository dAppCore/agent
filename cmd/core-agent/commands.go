// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"bytes"
	"flag"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/core"
)

type applicationCommandSet struct {
	core *core.Core
}

// startupArgs applies early log flags, then returns args for c.Cli().Run().
//
//	args := startupArgs()
//	_ = c.Cli().Run(args...)
func startupArgs() []string {
	previous := flag.CommandLine
	commandLine := flag.NewFlagSet("core-agent", flag.ContinueOnError)
	commandLine.SetOutput(&bytes.Buffer{})
	commandLine.BoolFunc("quiet", "", func(string) error {
		core.SetLevel(core.LevelError)
		return nil
	})
	commandLine.BoolFunc("q", "", func(string) error {
		core.SetLevel(core.LevelError)
		return nil
	})
	commandLine.BoolFunc("debug", "", func(string) error {
		core.SetLevel(core.LevelDebug)
		return nil
	})
	commandLine.BoolFunc("d", "", func(string) error {
		core.SetLevel(core.LevelDebug)
		return nil
	})

	flag.CommandLine = commandLine
	defer func() {
		flag.CommandLine = previous
	}()

	flag.Parse()
	return applyLogLevel(commandLine.Args())
}

// applyLogLevel strips log-level flags from args and applies the level in-order.
//
//	args := applyLogLevel([]string{"version", "-q"})
//	args := applyLogLevel([]string{"--debug", "mcp"})
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

// c.Command("version", core.Command{Description: "Print version and build info", Action: commands.version})
// c.Command("check", core.Command{Description: "Verify workspace, deps, and config", Action: commands.check})
// c.Command("env", core.Command{Description: "Show all core.Env() keys and values", Action: commands.env})
func registerApplicationCommands(c *core.Core) {
	commands := applicationCommandSet{core: c}

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

func (commands applicationCommandSet) version(_ core.Options) core.Result {
	core.Print(nil, "core-agent %s", commands.core.App().Version)
	core.Print(nil, "  go:       %s", core.Env("GO"))
	core.Print(nil, "  os:       %s/%s", core.Env("OS"), core.Env("ARCH"))
	core.Print(nil, "  home:     %s", agentic.HomeDir())
	core.Print(nil, "  hostname: %s", core.Env("HOSTNAME"))
	core.Print(nil, "  pid:      %s", core.Env("PID"))
	core.Print(nil, "  channel:  %s", updateChannel())
	return core.Result{OK: true}
}

func (commands applicationCommandSet) check(_ core.Options) core.Result {
	fs := commands.core.Fs()
	core.Print(nil, "core-agent %s health check", commands.core.App().Version)
	core.Print(nil, "")
	core.Print(nil, "  binary:    core-agent")

	agentsPath := core.JoinPath(agentic.CoreRoot(), "agents.yaml")
	if fs.IsFile(agentsPath) {
		core.Print(nil, "  agents:    %s (ok)", agentsPath)
	} else {
		core.Print(nil, "  agents:    %s (MISSING)", agentsPath)
	}

	workspaceRoot := agentic.WorkspaceRoot()
	if fs.IsDir(workspaceRoot) {
		statusFiles := agentic.WorkspaceStatusPaths()
		core.Print(nil, "  workspace: %s (%d workspaces)", workspaceRoot, len(statusFiles))
	} else {
		core.Print(nil, "  workspace: %s (MISSING)", workspaceRoot)
	}

	core.Print(nil, "  services:  %d registered", len(commands.core.Services()))
	core.Print(nil, "  actions:   %d registered", len(commands.core.Actions()))
	core.Print(nil, "  commands:  %d registered", len(commands.core.Commands()))
	core.Print(nil, "  env keys:  %d loaded", len(core.EnvKeys()))
	core.Print(nil, "")
	core.Print(nil, "ok")
	return core.Result{OK: true}
}

func (commands applicationCommandSet) env(_ core.Options) core.Result {
	keys := core.EnvKeys()
	for _, key := range keys {
		core.Print(nil, "  %-15s %s", key, core.Env(key))
	}
	return core.Result{OK: true}
}
