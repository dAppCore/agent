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
	c.Command("generate", core.Command{Description: "Generate content from a prompt using the platform content pipeline", Action: s.cmdGenerate})
	c.Command("brain/ingest", core.Command{Description: "Bulk ingest memories into OpenBrain", Action: s.cmdBrainIngest})
	c.Command("brain/seed-memory", core.Command{Description: "Import markdown memories into OpenBrain from a project memory directory", Action: s.cmdBrainSeedMemory})
	c.Command("plan-cleanup", core.Command{Description: "Permanently delete archived plans past the retention period", Action: s.cmdPlanCleanup})
	c.Command("status", core.Command{Description: "List agent workspace statuses", Action: s.cmdStatus})
	c.Command("prompt", core.Command{Description: "Build and display an agent prompt for a repo", Action: s.cmdPrompt})
	c.Command("extract", core.Command{Description: "Extract a workspace template to a directory", Action: s.cmdExtract})
	s.registerPlanCommands()
	s.registerTaskCommands()
}

// ctx := s.commandContext()
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

	issue := parseIntString(issueValue)

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
		failureError := result.Error
		if failureError == nil {
			failureError = core.E("agentic.runTask", "dispatch failed", nil)
		}
		core.Print(nil, "FAILED: %v", failureError)
		return core.Result{Value: failureError, OK: false}
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
		prepInput.Issue = parseIntString(value)
	}
	if value := options.String("pr"); value != "" {
		prepInput.PR = parseIntString(value)
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

	_, prepOutput, err := s.PrepareWorkspace(context.Background(), prepInput)
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

func (s *PrepSubsystem) cmdGenerate(options core.Options) core.Result {
	prompt := optionStringValue(options, "prompt", "_arg")
	briefID := optionStringValue(options, "brief_id", "brief-id")
	template := optionStringValue(options, "template")
	if prompt == "" && (briefID == "" || template == "") {
		core.Print(nil, "usage: core-agent generate --prompt=\"Draft a release note\" [--brief-id=brief_1 --template=help-article] [--provider=claude] [--config='{\"max_tokens\":4000}']")
		return core.Result{Value: core.E("agentic.cmdGenerate", "prompt or brief-id/template is required", nil), OK: false}
	}

	result := s.handleContentGenerate(s.commandContext(), core.NewOptions(
		core.Option{Key: "prompt", Value: prompt},
		core.Option{Key: "brief_id", Value: briefID},
		core.Option{Key: "template", Value: template},
		core.Option{Key: "provider", Value: options.String("provider")},
		core.Option{Key: "config", Value: options.String("config")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdGenerate", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(ContentGenerateOutput)
	if !ok {
		err := core.E("agentic.cmdGenerate", "invalid content generate output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Result.Provider != "" {
		core.Print(nil, "provider: %s", output.Result.Provider)
	}
	if output.Result.Model != "" {
		core.Print(nil, "model:    %s", output.Result.Model)
	}
	if output.Result.Status != "" {
		core.Print(nil, "status:   %s", output.Result.Status)
	}
	if output.Result.Content != "" {
		core.Print(nil, "content:  %s", output.Result.Content)
	}
	if output.Result.InputTokens > 0 || output.Result.OutputTokens > 0 {
		core.Print(nil, "tokens:   %d in / %d out", output.Result.InputTokens, output.Result.OutputTokens)
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

	prompt, memories, consumers := s.BuildPrompt(context.Background(), prepInput, "dev", repoPath)
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

// parseIntString("issue-42") // 42
func parseIntString(s string) int {
	n := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n
}
