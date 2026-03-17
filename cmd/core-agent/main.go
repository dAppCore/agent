package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"forge.lthn.ai/core/agent/pkg/agentic"
	"forge.lthn.ai/core/agent/pkg/brain"
	"forge.lthn.ai/core/agent/pkg/monitor"
	"forge.lthn.ai/core/cli/pkg/cli"
	"forge.lthn.ai/core/go-process"
	"forge.lthn.ai/core/go/pkg/core"
	"forge.lthn.ai/core/mcp/pkg/mcp"
)

func main() {
	if err := cli.Init(cli.Options{
		AppName: "core-agent",
		Version: "0.2.0",
	}); err != nil {
		log.Fatal(err)
	}

	// Shared setup for both mcp and serve commands
	initServices := func() (*mcp.Service, *monitor.Subsystem, error) {
		c, err := core.New(core.WithName("process", process.NewService(process.Options{})))
		if err != nil {
			return nil, nil, cli.Wrap(err, "init core")
		}
		procSvc, err := core.ServiceFor[*process.Service](c, "process")
		if err != nil {
			return nil, nil, cli.Wrap(err, "get process service")
		}
		process.SetDefault(procSvc)

		mon := monitor.New()
		prep := agentic.NewPrep()
		prep.SetCompletionNotifier(mon)

		mcpSvc, err := mcp.New(
			mcp.WithSubsystem(brain.NewDirect()),
			mcp.WithSubsystem(prep),
			mcp.WithSubsystem(mon),
		)
		if err != nil {
			return nil, nil, cli.Wrap(err, "create MCP service")
		}

		return mcpSvc, mon, nil
	}

	// mcp — stdio transport (Claude Code integration)
	mcpCmd := cli.NewCommand("mcp", "Start the MCP server on stdio", "", func(cmd *cli.Command, args []string) error {
		mcpSvc, mon, err := initServices()
		if err != nil {
			return err
		}
		mon.Start(cmd.Context())
		return mcpSvc.Run(cmd.Context())
	})

	// serve — persistent HTTP daemon (Charon, CI, cross-agent)
	serveCmd := cli.NewCommand("serve", "Start as a persistent HTTP daemon", "", func(cmd *cli.Command, args []string) error {
		mcpSvc, mon, err := initServices()
		if err != nil {
			return err
		}

		// Determine address
		addr := os.Getenv("MCP_HTTP_ADDR")
		if addr == "" {
			addr = "0.0.0.0:9101"
		}

		// Determine health address
		healthAddr := os.Getenv("HEALTH_ADDR")
		if healthAddr == "" {
			healthAddr = "0.0.0.0:9102"
		}

		// Set up daemon with PID file, health check, and registry
		home, _ := os.UserHomeDir()
		pidFile := filepath.Join(home, ".core", "core-agent.pid")

		daemon := process.NewDaemon(process.DaemonOptions{
			PIDFile:    pidFile,
			HealthAddr: healthAddr,
			Registry:   process.DefaultRegistry(),
			RegistryEntry: process.DaemonEntry{
				Code:    "core",
				Daemon:  "agent",
				Project: "core-agent",
				Binary:  "core-agent",
			},
		})

		if err := daemon.Start(); err != nil {
			return cli.Wrap(err, "daemon start")
		}

		// Start monitor
		mon.Start(cmd.Context())

		// Mark ready
		daemon.SetReady(true)
		fmt.Fprintf(os.Stderr, "core-agent serving on %s (health: %s, pid: %s)\n", addr, healthAddr, pidFile)

		// Set env so mcp.Run picks HTTP transport
		os.Setenv("MCP_HTTP_ADDR", addr)

		// Run MCP server (blocks until context cancelled)
		return mcpSvc.Run(cmd.Context())
	})

	cli.RootCmd().AddCommand(mcpCmd)
	cli.RootCmd().AddCommand(serveCmd)

	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
