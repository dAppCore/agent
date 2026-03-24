package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"dappco.re/go/core"
	"dappco.re/go/core/process"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/brain"
	"dappco.re/go/agent/pkg/lib"
	"dappco.re/go/agent/pkg/monitor"
	"forge.lthn.ai/core/mcp/pkg/mcp"
)

func main() {
	r := core.New(
		core.WithOption("name", "core-agent"),
		core.WithService(agentic.Register),
		core.WithService(monitor.Register),
		core.WithService(brain.Register),
	)
	if !r.OK {
		core.Error("failed to create core", "err", r.Value)
		os.Exit(1)
	}
	c := r.Value.(*core.Core)

	// Version set at build time: go build -ldflags "-X main.version=0.15.0"
	if version != "" {
		c.App().Version = version
	} else {
		c.App().Version = "dev"
	}

	// version — print version and build info
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

	// check — verify workspace, deps, and config are healthy
	c.Command("check", core.Command{
		Description: "Verify workspace, deps, and config",
		Action: func(opts core.Options) core.Result {
			fs := c.Fs()

			core.Print(nil, "core-agent %s health check", c.App().Version)
			core.Print(nil, "")

			// Binary location
			core.Print(nil, "  binary:    %s", os.Args[0])

			// Agents config
			agentsPath := core.Path("Code", ".core", "agents.yaml")
			if fs.IsFile(agentsPath) {
				core.Print(nil, "  agents:    %s (ok)", agentsPath)
			} else {
				core.Print(nil, "  agents:    %s (MISSING)", agentsPath)
			}

			// Workspace dir
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

			// Core dep version
			core.Print(nil, "  core:      dappco.re/go/core@v%s", c.App().Version)

			// Env keys
			core.Print(nil, "  env keys:  %d loaded", len(core.EnvKeys()))

			core.Print(nil, "")
			core.Print(nil, "ok")
			return core.Result{OK: true}
		},
	})

	// extract — test workspace template extraction
	c.Command("extract", core.Command{
		Description: "Extract a workspace template to a directory",
		Action: func(opts core.Options) core.Result {
			tmpl := opts.String("_arg")
			if tmpl == "" {
				tmpl = "default"
			}
			target := opts.String("target")
			if target == "" {
				target = core.Path("Code", ".core", "workspace", "test-extract")
			}

			data := &lib.WorkspaceData{
				Repo:   "test-repo",
				Branch: "dev",
				Task:   "test extraction",
				Agent:  "codex",
			}

			core.Print(nil, "extracting template %q to %s", tmpl, target)
			if err := lib.ExtractWorkspace(tmpl, target, data); err != nil {
				return core.Result{Value: err, OK: false}
			}

			// List what was created
			fs := &core.Fs{}
			r := fs.List(target)
			if r.OK {
				for _, e := range r.Value.([]os.DirEntry) {
					marker := " "
					if e.IsDir() {
						marker = "/"
					}
					core.Print(nil, "  %s%s", e.Name(), marker)
				}
			}

			core.Print(nil, "done")
			return core.Result{OK: true}
		},
	})

	// --- Forge + Workspace CLI commands ---
	registerForgeCommands(c)
	registerWorkspaceCommands(c)
	// registerUpdateCommand(c) — parked until version moves to module root

	// --- CLI commands for feature testing ---

	prep := agentic.NewPrep()

	// prep — test workspace preparation (clone + prompt)
	c.Command("prep", core.Command{
		Description: "Prepare a workspace: clone repo, build prompt",
		Action: func(opts core.Options) core.Result {
			repo := opts.String("_arg")
			if repo == "" {
				core.Print(nil, "usage: core-agent prep <repo> --issue=N|--pr=N|--branch=X --task=\"...\"")
				return core.Result{OK: false}
			}

			input := agentic.PrepInput{
				Repo:     repo,
				Org:      opts.String("org"),
				Task:     opts.String("task"),
				Template: opts.String("template"),
				Persona:  opts.String("persona"),
				DryRun:   opts.Bool("dry-run"),
			}

			// Parse identifier from flags
			if v := opts.String("issue"); v != "" {
				n := 0
				for _, ch := range v {
					if ch >= '0' && ch <= '9' {
						n = n*10 + int(ch-'0')
					}
				}
				input.Issue = n
			}
			if v := opts.String("pr"); v != "" {
				n := 0
				for _, ch := range v {
					if ch >= '0' && ch <= '9' {
						n = n*10 + int(ch-'0')
					}
				}
				input.PR = n
			}
			if v := opts.String("branch"); v != "" {
				input.Branch = v
			}
			if v := opts.String("tag"); v != "" {
				input.Tag = v
			}

			// Default to branch "dev" if no identifier
			if input.Issue == 0 && input.PR == 0 && input.Branch == "" && input.Tag == "" {
				input.Branch = "dev"
			}

			_, out, err := prep.TestPrepWorkspace(context.Background(), input)
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			core.Print(nil, "workspace: %s", out.WorkspaceDir)
			core.Print(nil, "repo:      %s", out.RepoDir)
			core.Print(nil, "branch:    %s", out.Branch)
			core.Print(nil, "resumed:   %v", out.Resumed)
			core.Print(nil, "memories:  %d", out.Memories)
			core.Print(nil, "consumers: %d", out.Consumers)
			if out.Prompt != "" {
				core.Print(nil, "")
				core.Print(nil, "--- prompt (%d chars) ---", len(out.Prompt))
				core.Print(nil, "%s", out.Prompt)
			}
			return core.Result{OK: true}
		},
	})

	// status — list workspace statuses
	c.Command("status", core.Command{
		Description: "List agent workspace statuses",
		Action: func(opts core.Options) core.Result {
			wsRoot := agentic.WorkspaceRoot()
			fsys := c.Fs()
			r := fsys.List(wsRoot)
			if !r.OK {
				core.Print(nil, "no workspaces found at %s", wsRoot)
				return core.Result{OK: true}
			}

			entries := r.Value.([]os.DirEntry)
			if len(entries) == 0 {
				core.Print(nil, "no workspaces")
				return core.Result{OK: true}
			}

			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				statusFile := core.JoinPath(wsRoot, e.Name(), "status.json")
				if sr := fsys.Read(statusFile); sr.OK {
					core.Print(nil, "  %s", e.Name())
				}
			}
			return core.Result{OK: true}
		},
	})

	// prompt — build and show an agent prompt without cloning
	c.Command("prompt", core.Command{
		Description: "Build and display an agent prompt for a repo",
		Action: func(opts core.Options) core.Result {
			repo := opts.String("_arg")
			if repo == "" {
				core.Print(nil, "usage: core-agent prompt <repo> --task=\"...\"")
				return core.Result{OK: false}
			}

			org := opts.String("org")
			if org == "" {
				org = "core"
			}
			task := opts.String("task")
			if task == "" {
				task = "Review and report findings"
			}

			repoPath := core.JoinPath(core.Env("DIR_HOME"), "Code", org, repo)

			input := agentic.PrepInput{
				Repo:     repo,
				Org:      org,
				Task:     task,
				Template: opts.String("template"),
				Persona:  opts.String("persona"),
			}

			prompt, memories, consumers := prep.TestBuildPrompt(context.Background(), input, "dev", repoPath)
			core.Print(nil, "memories:  %d", memories)
			core.Print(nil, "consumers: %d", consumers)
			core.Print(nil, "")
			core.Print(nil, "%s", prompt)
			return core.Result{OK: true}
		},
	})

	// env — dump all Env keys
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

	// Retrieve service instances from conclave for MCP tool registration
	agenticSvc, _ := core.ServiceFor[*agentic.PrepSubsystem](c, "agentic")
	monitorSvc, _ := core.ServiceFor[*monitor.Subsystem](c, "monitor")
	brainSvc, _ := core.ServiceFor[*brain.DirectSubsystem](c, "brain")

	// Process service (lifecycle management)
	procFactory := process.NewService(process.Options{})
	procResult, procErr := procFactory(c)
	if procErr == nil {
		if procSvc, ok := procResult.(*process.Service); ok {
			_ = process.SetDefault(procSvc)
		}
	}

	// MCP service — wires subsystems for tool registration
	initMCP := func() (*mcp.Service, error) {
		mcpSvc, err := mcp.New(mcp.Options{
			Subsystems: []mcp.Subsystem{brainSvc, agenticSvc, monitorSvc},
		})
		if err != nil {
			return nil, core.E("main", "create MCP service", err)
		}
		monitorSvc.SetNotifier(mcpSvc)
		return mcpSvc, nil
	}

	// Signal-aware context for clean shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// mcp — stdio transport (Claude Code integration)
	c.Command("mcp", core.Command{
		Description: "Start the MCP server on stdio",
		Action: func(opts core.Options) core.Result {
			mcpSvc, err := initMCP()
			if err != nil {
				return core.Result{Value: err, OK: false}
			}
			c.ServiceStartup(ctx, nil)
			if err := mcpSvc.Run(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			c.ServiceShutdown(context.Background())
			return core.Result{OK: true}
		},
	})

	// serve — persistent HTTP daemon (Charon, CI, cross-agent)
	c.Command("serve", core.Command{
		Description: "Start as a persistent HTTP daemon",
		Action: func(opts core.Options) core.Result {
			mcpSvc, err := initMCP()
			if err != nil {
				return core.Result{Value: err, OK: false}
			}

			addr := core.Env("MCP_HTTP_ADDR")
			if addr == "" {
				addr = "0.0.0.0:9101"
			}

			healthAddr := core.Env("HEALTH_ADDR")
			if healthAddr == "" {
				healthAddr = "0.0.0.0:9102"
			}

			pidFile := core.Path(".core", "core-agent.pid")

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

			c.ServiceStartup(ctx, nil)
			daemon.SetReady(true)
			core.Print(os.Stderr, "core-agent serving on %s (health: %s, pid: %s)", addr, healthAddr, pidFile)

			os.Setenv("MCP_HTTP_ADDR", addr)

			if err := mcpSvc.Run(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			c.ServiceShutdown(context.Background())
			return core.Result{OK: true}
		},
	})

	// run task — single task e2e (prep → spawn → wait → done)
	c.Command("run/task", core.Command{
		Description: "Run a single task end-to-end",
		Action: func(opts core.Options) core.Result {
			repo := opts.String("repo")
			agent := opts.String("agent")
			task := opts.String("task")
			issueStr := opts.String("issue")
			org := opts.String("org")

			if repo == "" || task == "" {
				core.Print(nil, "usage: core-agent run task --repo=<repo> --task=\"...\" --agent=codex [--issue=N] [--org=core]")
				return core.Result{OK: false}
			}
			if agent == "" {
				agent = "codex"
			}
			if org == "" {
				org = "core"
			}

			issue := 0
			if issueStr != "" {
				if n, err := strconv.Atoi(issueStr); err == nil {
					issue = n
				}
			}

			core.Print(os.Stderr, "core-agent run task")
			core.Print(os.Stderr, "  repo:  %s/%s", org, repo)
			core.Print(os.Stderr, "  agent: %s", agent)
			if issue > 0 {
				core.Print(os.Stderr, "  issue: #%d", issue)
			}
			core.Print(os.Stderr, "  task:  %s", task)
			core.Print(os.Stderr, "")

			// Dispatch and wait
			result := agenticSvc.DispatchSync(ctx, agentic.DispatchSyncInput{
				Org:   org,
				Repo:  repo,
				Agent: agent,
				Task:  task,
				Issue: issue,
			})

			if !result.OK {
				core.Print(os.Stderr, "FAILED: %v", result.Error)
				return core.Result{Value: result.Error, OK: false}
			}

			core.Print(os.Stderr, "DONE: %s", result.Status)
			if result.PRURL != "" {
				core.Print(os.Stderr, "  PR: %s", result.PRURL)
			}
			return core.Result{OK: true}
		},
	})

	// run orchestrator — standalone queue runner without MCP stdio
	c.Command("run/orchestrator", core.Command{
		Description: "Run the queue orchestrator (standalone, no MCP)",
		Action: func(opts core.Options) core.Result {
			c.ServiceStartup(ctx, nil)

			core.Print(os.Stderr, "core-agent orchestrator running (pid %s)", core.Env("PID"))
			core.Print(os.Stderr, "  workspace: %s", agentic.WorkspaceRoot())
			core.Print(os.Stderr, "  watching queue, draining on 30s tick + completion poke")

			// Block until signal
			<-ctx.Done()
			core.Print(os.Stderr, "orchestrator shutting down")
			c.ServiceShutdown(context.Background())
			return core.Result{OK: true}
		},
	})

	// Run CLI — resolves os.Args to command path
	result := c.Cli().Run()
	if !result.OK {
		if err, ok := result.Value.(error); ok {
			core.Error(err.Error())
		}
		os.Exit(1)
	}
}
