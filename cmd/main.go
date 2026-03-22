package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"dappco.re/go/core"
	"dappco.re/go/core/process"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/monitor"
	"forge.lthn.ai/core/mcp/pkg/mcp"
)

func main() {
	c := core.New(core.Options{
		{Key: "name", Value: "core-agent"},
	})
	c.App().Version = "0.2.0"

	// Shared setup — creates MCP service with all subsystems wired
	initServices := func() (*mcp.Service, *monitor.Subsystem, error) {
		procFactory := process.NewService(process.Options{})
		procResult, err := procFactory(c)
		if err != nil {
			return nil, nil, core.E("main", "init process service", err)
		}
		if procSvc, ok := procResult.(*process.Service); ok {
			_ = process.SetDefault(procSvc)
		}

		mon := monitor.New()
		prep := agentic.NewPrep()
		prep.SetCompletionNotifier(mon)

		mcpSvc, err := mcp.New(mcp.Options{
			Subsystems: []mcp.Subsystem{brain.NewDirect(), prep, mon},
		})
		if err != nil {
			return nil, nil, core.E("main", "create MCP service", err)
		}

		mon.SetNotifier(mcpSvc)
		return mcpSvc, mon, nil
	}

	// Signal-aware context for clean shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// mcp — stdio transport (Claude Code integration)
	c.Command("mcp", core.Command{
		Description: "Start the MCP server on stdio",
		Action: func(opts core.Options) core.Result {
			mcpSvc, mon, err := initServices()
			if err != nil {
				return core.Result{Value: err, OK: false}
			}
			mon.Start(ctx)
			if err := mcpSvc.Run(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	// serve — persistent HTTP daemon (Charon, CI, cross-agent)
	c.Command("serve", core.Command{
		Description: "Start as a persistent HTTP daemon",
		Action: func(opts core.Options) core.Result {
			mcpSvc, mon, err := initServices()
			if err != nil {
				return core.Result{Value: err, OK: false}
			}

			addr := os.Getenv("MCP_HTTP_ADDR")
			if addr == "" {
				addr = "0.0.0.0:9101"
			}

			healthAddr := os.Getenv("HEALTH_ADDR")
			if healthAddr == "" {
				healthAddr = "0.0.0.0:9102"
			}

			home, _ := os.UserHomeDir()
			pidFile := core.Concat(home, "/.core/core-agent.pid")

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
				return core.Result{Value: core.E("main", "daemon start", err), OK: false}
			}

			mon.Start(ctx)
			daemon.SetReady(true)
			core.Print(os.Stderr, "core-agent serving on %s (health: %s, pid: %s)", addr, healthAddr, pidFile)

			os.Setenv("MCP_HTTP_ADDR", addr)

			if err := mcpSvc.Run(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	// Run CLI — resolves os.Args to command path
	r := c.Cli().Run()
	if !r.OK {
		if err, ok := r.Value.(error); ok {
			core.Error(err.Error())
		}
		os.Exit(1)
	}
}
