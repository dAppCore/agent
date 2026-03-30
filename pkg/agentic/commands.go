// SPDX-License-Identifier: EUPL-1.2

// CLI commands registered by the agentic service during OnStartup.

package agentic

import (
	"context"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
)

// registerCommands adds agentic CLI commands to Core's command tree.
func (s *PrepSubsystem) registerCommands(ctx context.Context) {
	s.commandCtx = ctx
	c := s.Core()
	c.Command("run/task", core.Command{Description: "Run a single task end-to-end", Action: s.cmdRunTask})
	c.Command("run/orchestrator", core.Command{Description: "Run the queue orchestrator (standalone, no MCP)", Action: s.cmdOrchestrator})
	c.Command("prep", core.Command{Description: "Prepare a workspace: clone repo, build prompt", Action: s.cmdPrep})
	c.Command("status", core.Command{Description: "List agent workspace statuses", Action: s.cmdStatus})
	c.Command("prompt", core.Command{Description: "Build and display an agent prompt for a repo", Action: s.cmdPrompt})
	c.Command("extract", core.Command{Description: "Extract a workspace template to a directory", Action: s.cmdExtract})
}

// commandContext returns the startup context captured during command registration.
//
//	ctx := s.commandContext()
//	_ = ctx.Err()
func (s *PrepSubsystem) commandContext() context.Context {
	if s.commandCtx != nil {
		return s.commandCtx
	}
	return context.Background()
}

func (s *PrepSubsystem) cmdRunTask(opts core.Options) core.Result {
	return s.runTask(s.commandContext(), opts)
}

func (s *PrepSubsystem) runTask(ctx context.Context, opts core.Options) core.Result {
	repo := opts.String("repo")
	agent := opts.String("agent")
	task := opts.String("task")
	issueStr := opts.String("issue")
	org := opts.String("org")

	if repo == "" || task == "" {
		core.Print(nil, "usage: core-agent run task --repo=<repo> --task=\"...\" --agent=codex [--issue=N] [--org=core]")
		return core.Result{Value: core.E("agentic.runTask", "repo and task are required", nil), OK: false}
	}
	if agent == "" {
		agent = "codex"
	}
	if org == "" {
		org = "core"
	}

	issue := parseIntStr(issueStr)

	core.Print(nil, "core-agent run task")
	core.Print(nil, "  repo:  %s/%s", org, repo)
	core.Print(nil, "  agent: %s", agent)
	if issue > 0 {
		core.Print(nil, "  issue: #%d", issue)
	}
	core.Print(nil, "  task:  %s", task)
	core.Print(nil, "")

	result := s.DispatchSync(ctx, DispatchSyncInput{
		Org: org, Repo: repo, Agent: agent, Task: task, Issue: issue,
	})

	if !result.OK {
		failureErr := result.Err
		if failureErr == nil {
			failureErr = core.E("agentic.runTask", "dispatch failed", nil)
		}
		core.Print(nil, "FAILED: %v", failureErr)
		return core.Result{Value: failureErr, OK: false}
	}

	core.Print(nil, "DONE: %s", result.Status)
	if result.PRURL != "" {
		core.Print(nil, "  PR: %s", result.PRURL)
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdOrchestrator(_ core.Options) core.Result {
	ctx := s.commandContext()
	core.Print(nil, "core-agent orchestrator running (pid %s)", core.Env("PID"))
	core.Print(nil, "  workspace: %s", WorkspaceRoot())
	core.Print(nil, "  watching queue, draining on 30s tick + completion poke")

	<-ctx.Done()
	core.Print(nil, "orchestrator shutting down")
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdPrep(opts core.Options) core.Result {
	repo := opts.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent prep <repo> --issue=N|--pr=N|--branch=X --task=\"...\"")
		return core.Result{Value: core.E("agentic.cmdPrep", "repo is required", nil), OK: false}
	}

	input := PrepInput{
		Repo:     repo,
		Org:      opts.String("org"),
		Task:     opts.String("task"),
		Template: opts.String("template"),
		Persona:  opts.String("persona"),
		DryRun:   opts.Bool("dry-run"),
	}

	if v := opts.String("issue"); v != "" {
		input.Issue = parseIntStr(v)
	}
	if v := opts.String("pr"); v != "" {
		input.PR = parseIntStr(v)
	}
	if v := opts.String("branch"); v != "" {
		input.Branch = v
	}
	if v := opts.String("tag"); v != "" {
		input.Tag = v
	}

	if input.Issue == 0 && input.PR == 0 && input.Branch == "" && input.Tag == "" {
		input.Branch = "dev"
	}

	_, out, err := s.TestPrepWorkspace(context.Background(), input)
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
}

func (s *PrepSubsystem) cmdStatus(opts core.Options) core.Result {
	wsRoot := WorkspaceRoot()
	fsys := s.Core().Fs()
	r := fsys.List(wsRoot)
	if !r.OK {
		core.Print(nil, "no workspaces found at %s", wsRoot)
		return core.Result{OK: true}
	}

	statusFiles := WorkspaceStatusPaths()
	if len(statusFiles) == 0 {
		core.Print(nil, "no workspaces")
		return core.Result{OK: true}
	}

	for _, sf := range statusFiles {
		core.Print(nil, "  %s", WorkspaceName(core.PathDir(sf)))
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdPrompt(opts core.Options) core.Result {
	repo := opts.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent prompt <repo> --task=\"...\"")
		return core.Result{Value: core.E("agentic.cmdPrompt", "repo is required", nil), OK: false}
	}

	org := opts.String("org")
	if org == "" {
		org = "core"
	}
	task := opts.String("task")
	if task == "" {
		task = "Review and report findings"
	}

	repoPath := core.JoinPath(HomeDir(), "Code", org, repo)

	input := PrepInput{
		Repo:     repo,
		Org:      org,
		Task:     task,
		Template: opts.String("template"),
		Persona:  opts.String("persona"),
	}

	prompt, memories, consumers := s.TestBuildPrompt(context.Background(), input, "dev", repoPath)
	core.Print(nil, "memories:  %d", memories)
	core.Print(nil, "consumers: %d", consumers)
	core.Print(nil, "")
	core.Print(nil, "%s", prompt)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdExtract(opts core.Options) core.Result {
	tmpl := opts.String("_arg")
	if tmpl == "" {
		tmpl = "default"
	}
	target := opts.String("target")
	if target == "" {
		target = core.JoinPath(WorkspaceRoot(), "test-extract")
	}

	data := &lib.WorkspaceData{
		Repo:   "test-repo",
		Branch: "dev",
		Task:   "test extraction",
		Agent:  "codex",
	}

	core.Print(nil, "extracting template %q to %s", tmpl, target)
	if result := lib.ExtractWorkspace(tmpl, target, data); !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Result{Value: core.E("agentic.cmdExtract", core.Concat("extract workspace template ", tmpl), err), OK: false}
		}
		return core.Result{Value: core.E("agentic.cmdExtract", core.Concat("extract workspace template ", tmpl), nil), OK: false}
	}

	fsys := s.Core().Fs()
	paths := core.PathGlob(core.JoinPath(target, "*"))
	for _, p := range paths {
		name := core.PathBase(p)
		marker := " "
		if fsys.IsDir(p) {
			marker = "/"
		}
		core.Print(nil, "  %s%s", name, marker)
	}

	core.Print(nil, "done")
	return core.Result{OK: true}
}

// parseIntStr extracts digits from a string and returns the integer value.
func parseIntStr(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}
