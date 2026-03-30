// SPDX-License-Identifier: EUPL-1.2

// c := core.New(core.WithOption("name", "core-agent"))
// registerApplicationCommands(c)

package agentic

import (
	"context"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
)

// c.Command("run/task", core.Command{Description: "Run a single task end-to-end", Action: s.cmdRunTask})
// c.Command("prep", core.Command{Description: "Prepare a workspace: clone repo, build prompt", Action: s.cmdPrep})
func (s *PrepSubsystem) registerCommands(ctx context.Context) {
	s.startupContext = ctx
	c := s.Core()
	c.Command("run/task", core.Command{Description: "Run a single task end-to-end", Action: s.cmdRunTask})
	c.Command("run/orchestrator", core.Command{Description: "Run the queue orchestrator (standalone, no MCP)", Action: s.cmdOrchestrator})
	c.Command("prep", core.Command{Description: "Prepare a workspace: clone repo, build prompt", Action: s.cmdPrep})
	c.Command("status", core.Command{Description: "List agent workspace statuses", Action: s.cmdStatus})
	c.Command("prompt", core.Command{Description: "Build and display an agent prompt for a repo", Action: s.cmdPrompt})
	c.Command("extract", core.Command{Description: "Extract a workspace template to a directory", Action: s.cmdExtract})
}

// ctx := s.commandContext()
// _ = ctx.Err()
func (s *PrepSubsystem) commandContext() context.Context {
	if s.startupContext != nil {
		return s.startupContext
	}
	return context.Background()
}

func (s *PrepSubsystem) cmdRunTask(options core.Options) core.Result {
	return s.runTask(s.commandContext(), options)
}

func (s *PrepSubsystem) runTask(ctx context.Context, options core.Options) core.Result {
	repo := options.String("repo")
	agent := options.String("agent")
	task := options.String("task")
	issueValue := options.String("issue")
	org := options.String("org")

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

	issue := parseIntStr(issueValue)

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

func (s *PrepSubsystem) cmdPrep(options core.Options) core.Result {
	repo := options.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent prep <repo> --issue=N|--pr=N|--branch=X --task=\"...\"")
		return core.Result{Value: core.E("agentic.cmdPrep", "repo is required", nil), OK: false}
	}

	prepInput := PrepInput{
		Repo:     repo,
		Org:      options.String("org"),
		Task:     options.String("task"),
		Template: options.String("template"),
		Persona:  options.String("persona"),
		DryRun:   options.Bool("dry-run"),
	}

	if value := options.String("issue"); value != "" {
		prepInput.Issue = parseIntStr(value)
	}
	if value := options.String("pr"); value != "" {
		prepInput.PR = parseIntStr(value)
	}
	if value := options.String("branch"); value != "" {
		prepInput.Branch = value
	}
	if value := options.String("tag"); value != "" {
		prepInput.Tag = value
	}

	if prepInput.Issue == 0 && prepInput.PR == 0 && prepInput.Branch == "" && prepInput.Tag == "" {
		prepInput.Branch = "dev"
	}

	_, prepOutput, err := s.TestPrepWorkspace(context.Background(), prepInput)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "workspace: %s", prepOutput.WorkspaceDir)
	core.Print(nil, "repo:      %s", prepOutput.RepoDir)
	core.Print(nil, "branch:    %s", prepOutput.Branch)
	core.Print(nil, "resumed:   %v", prepOutput.Resumed)
	core.Print(nil, "memories:  %d", prepOutput.Memories)
	core.Print(nil, "consumers: %d", prepOutput.Consumers)
	if prepOutput.Prompt != "" {
		core.Print(nil, "")
		core.Print(nil, "--- prompt (%d chars) ---", len(prepOutput.Prompt))
		core.Print(nil, "%s", prepOutput.Prompt)
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdStatus(_ core.Options) core.Result {
	workspaceRoot := WorkspaceRoot()
	filesystem := s.Core().Fs()
	listResult := filesystem.List(workspaceRoot)
	if !listResult.OK {
		core.Print(nil, "no workspaces found at %s", workspaceRoot)
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

func (s *PrepSubsystem) cmdPrompt(options core.Options) core.Result {
	repo := options.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent prompt <repo> --task=\"...\"")
		return core.Result{Value: core.E("agentic.cmdPrompt", "repo is required", nil), OK: false}
	}

	org := options.String("org")
	if org == "" {
		org = "core"
	}
	task := options.String("task")
	if task == "" {
		task = "Review and report findings"
	}

	repoPath := core.JoinPath(HomeDir(), "Code", org, repo)

	prepInput := PrepInput{
		Repo:     repo,
		Org:      org,
		Task:     task,
		Template: options.String("template"),
		Persona:  options.String("persona"),
	}

	prompt, memories, consumers := s.TestBuildPrompt(context.Background(), prepInput, "dev", repoPath)
	core.Print(nil, "memories:  %d", memories)
	core.Print(nil, "consumers: %d", consumers)
	core.Print(nil, "")
	core.Print(nil, "%s", prompt)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdExtract(options core.Options) core.Result {
	templateName := options.String("_arg")
	if templateName == "" {
		templateName = "default"
	}
	target := options.String("target")
	if target == "" {
		target = core.JoinPath(WorkspaceRoot(), "test-extract")
	}

	workspaceData := &lib.WorkspaceData{
		Repo:   "test-repo",
		Branch: "dev",
		Task:   "test extraction",
		Agent:  "codex",
	}

	core.Print(nil, "extracting template %q to %s", templateName, target)
	if result := lib.ExtractWorkspace(templateName, target, workspaceData); !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Result{Value: core.E("agentic.cmdExtract", core.Concat("extract workspace template ", templateName), err), OK: false}
		}
		return core.Result{Value: core.E("agentic.cmdExtract", core.Concat("extract workspace template ", templateName), nil), OK: false}
	}

	filesystem := s.Core().Fs()
	paths := core.PathGlob(core.JoinPath(target, "*"))
	for _, p := range paths {
		name := core.PathBase(p)
		marker := " "
		if filesystem.IsDir(p) {
			marker = "/"
		}
		core.Print(nil, "  %s%s", name, marker)
	}

	core.Print(nil, "done")
	return core.Result{OK: true}
}

// parseIntStr("issue-42") // 42
func parseIntStr(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}
