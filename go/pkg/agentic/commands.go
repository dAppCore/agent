// SPDX-License-Identifier: EUPL-1.2

// c := core.New(core.WithOption("name", "core-agent"))
// registerApplicationCommands(c)

package agentic

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lib"
	"dappco.re/go/agent/pkg/lib/flow"
	"gopkg.in/yaml.v3"
)

// c.Command("run/task", core.Command{Description: "Run a single task end-to-end", Action: s.cmdRunTask})
// c.Command("prep", core.Command{Description: "Prepare a workspace: clone repo, build prompt", Action: s.cmdPrep})
func (s *PrepSubsystem) registerCommands(ctx context.Context) core.Result {
	s.startupContext = ctx
	c := s.Core()
	if result := s.registerRepoSyncSupport(); !result.OK {
		return result
	}
	if r := c.Command("run/task", core.Command{Description: "Run a single task end-to-end", Action: s.cmdRunTask}); !r.OK {
		return r
	}
	if r := c.Command("agentic:run/task", core.Command{Description: "Run a single task end-to-end", Action: s.cmdRunTask}); !r.OK {
		return r
	}
	if r := c.Command("run/flow", core.Command{Description: "Show a flow definition from disk or the embedded library", Action: s.cmdRunFlow}); !r.OK {
		return r
	}
	if r := c.Command("agentic:run/flow", core.Command{Description: "Show a flow definition from disk or the embedded library", Action: s.cmdRunFlow}); !r.OK {
		return r
	}
	if r := c.Command("flow/preview", core.Command{Description: "Preview a flow definition with optional variable substitution", Action: s.cmdFlowPreview}); !r.OK {
		return r
	}
	if r := c.Command("agentic:flow/preview", core.Command{Description: "Preview a flow definition with optional variable substitution", Action: s.cmdFlowPreview}); !r.OK {
		return r
	}
	if r := c.Command("dispatch/sync", core.Command{Description: "Dispatch a single task synchronously and block until it completes", Action: s.cmdDispatchSync}); !r.OK {
		return r
	}
	if r := c.Command("agentic:dispatch/sync", core.Command{Description: "Dispatch a single task synchronously and block until it completes", Action: s.cmdDispatchSync}); !r.OK {
		return r
	}
	if r := c.Command("run/orchestrator", core.Command{Description: "Run the queue orchestrator (standalone, no MCP)", Action: s.cmdOrchestrator}); !r.OK {
		return r
	}
	if r := c.Command("agentic:run/orchestrator", core.Command{Description: "Run the queue orchestrator (standalone, no MCP)", Action: s.cmdOrchestrator}); !r.OK {
		return r
	}
	if r := c.Command("dispatch", core.Command{Description: "Dispatch queued agents", Action: s.cmdDispatch}); !r.OK {
		return r
	}
	if r := c.Command("agentic:dispatch", core.Command{Description: "Dispatch queued agents", Action: s.cmdDispatch}); !r.OK {
		return r
	}
	if r := c.Command("dispatch/start", core.Command{Description: "Start the dispatch queue runner", Action: s.cmdDispatchStart}); !r.OK {
		return r
	}
	if r := c.Command("agentic:dispatch/start", core.Command{Description: "Start the dispatch queue runner", Action: s.cmdDispatchStart}); !r.OK {
		return r
	}
	if r := c.Command("dispatch/shutdown", core.Command{Description: "Freeze the dispatch queue gracefully", Action: s.cmdDispatchShutdown}); !r.OK {
		return r
	}
	if r := c.Command("agentic:dispatch/shutdown", core.Command{Description: "Freeze the dispatch queue gracefully", Action: s.cmdDispatchShutdown}); !r.OK {
		return r
	}
	if r := c.Command("dispatch/shutdown-now", core.Command{Description: "Hard stop the dispatch queue and kill running agents", Action: s.cmdDispatchShutdownNow}); !r.OK {
		return r
	}
	if r := c.Command("agentic:dispatch/shutdown-now", core.Command{Description: "Hard stop the dispatch queue and kill running agents", Action: s.cmdDispatchShutdownNow}); !r.OK {
		return r
	}
	if r := c.Command("poke", core.Command{Description: "Drain the dispatch queue immediately", Action: s.cmdPoke}); !r.OK {
		return r
	}
	if r := c.Command("agentic:poke", core.Command{Description: "Drain the dispatch queue immediately", Action: s.cmdPoke}); !r.OK {
		return r
	}
	if r := c.Command("prep", core.Command{Description: "Prepare a workspace: clone repo, build prompt", Action: s.cmdPrep}); !r.OK {
		return r
	}
	if r := c.Command("prep-workspace", core.Command{Description: "Prepare a workspace: clone repo, build prompt", Action: s.cmdPrep}); !r.OK {
		return r
	}
	if r := c.Command("agentic:prep-workspace", core.Command{Description: "Prepare a workspace: clone repo, build prompt", Action: s.cmdPrep}); !r.OK {
		return r
	}
	if r := c.Command("resume", core.Command{Description: "Resume a blocked or completed workspace", Action: s.cmdResume}); !r.OK {
		return r
	}
	if r := c.Command("agentic:resume", core.Command{Description: "Resume a blocked or completed workspace", Action: s.cmdResume}); !r.OK {
		return r
	}
	if r := c.Command("generate", core.Command{Description: "Generate content from a prompt using the platform content pipeline", Action: s.cmdGenerate}); !r.OK {
		return r
	}
	if r := c.Command("agentic:generate", core.Command{Description: "Generate content from a prompt using the platform content pipeline", Action: s.cmdGenerate}); !r.OK {
		return r
	}
	if r := c.Command("content/generate", core.Command{Description: "Generate content from a prompt using the platform content pipeline", Action: s.cmdGenerate}); !r.OK {
		return r
	}
	if r := c.Command("agentic:content/generate", core.Command{Description: "Generate content from a prompt using the platform content pipeline", Action: s.cmdGenerate}); !r.OK {
		return r
	}
	if r := c.Command("content/schema/generate", core.Command{Description: "Generate SEO schema JSON-LD for article, FAQ, or how-to content", Action: s.cmdContentSchemaGenerate}); !r.OK {
		return r
	}
	if r := c.Command("agentic:content/schema/generate", core.Command{Description: "Generate SEO schema JSON-LD for article, FAQ, or how-to content", Action: s.cmdContentSchemaGenerate}); !r.OK {
		return r
	}
	if r := c.Command("complete", core.Command{Description: "Run the completion pipeline (QA → PR → Verify → Commit → Ingest → Poke)", Action: s.cmdComplete}); !r.OK {
		return r
	}
	if r := c.Command("agentic:complete", core.Command{Description: "Run the completion pipeline (QA → PR → Verify → Commit → Ingest → Poke)", Action: s.cmdComplete}); !r.OK {
		return r
	}
	if r := c.Command("scan", core.Command{Description: "Scan Forge repos for actionable issues", Action: s.cmdScan}); !r.OK {
		return r
	}
	if r := c.Command("agentic:scan", core.Command{Description: "Scan Forge repos for actionable issues", Action: s.cmdScan}); !r.OK {
		return r
	}
	if r := c.Command("mirror", core.Command{Description: "Mirror Forge repos to GitHub", Action: s.cmdMirror}); !r.OK {
		return r
	}
	if r := c.Command("agentic:mirror", core.Command{Description: "Mirror Forge repos to GitHub", Action: s.cmdMirror}); !r.OK {
		return r
	}
	if r := c.Command("brain/ingest", core.Command{Description: "Bulk ingest memories into OpenBrain", Action: s.cmdBrainIngest}); !r.OK {
		return r
	}
	if r := c.Command("brain:ingest", core.Command{Description: "Bulk ingest memories into OpenBrain", Action: s.cmdBrainIngest}); !r.OK {
		return r
	}
	if r := c.Command("brain/recall", core.Command{Description: "Recall memories from OpenBrain", Action: s.cmdBrainRecall}); !r.OK {
		return r
	}
	if r := c.Command("brain:recall", core.Command{Description: "Recall memories from OpenBrain", Action: s.cmdBrainRecall}); !r.OK {
		return r
	}
	if r := c.Command("brain/remember", core.Command{Description: "Store a memory in OpenBrain", Action: s.cmdBrainRemember}); !r.OK {
		return r
	}
	if r := c.Command("brain:remember", core.Command{Description: "Store a memory in OpenBrain", Action: s.cmdBrainRemember}); !r.OK {
		return r
	}
	if r := c.Command("brain/seed-memory", core.Command{Description: "Import markdown memories into OpenBrain from a project memory file or directory", Action: s.cmdBrainSeedMemory}); !r.OK {
		return r
	}
	if r := c.Command("brain:seed-memory", core.Command{Description: "Import markdown memories into OpenBrain from a project memory file or directory", Action: s.cmdBrainSeedMemory}); !r.OK {
		return r
	}
	if r := c.Command("brain/list", core.Command{Description: "List memories in OpenBrain", Action: s.cmdBrainList}); !r.OK {
		return r
	}
	if r := c.Command("brain:list", core.Command{Description: "List memories in OpenBrain", Action: s.cmdBrainList}); !r.OK {
		return r
	}
	if r := c.Command("brain/forget", core.Command{Description: "Forget a memory in OpenBrain", Action: s.cmdBrainForget}); !r.OK {
		return r
	}
	if r := c.Command("brain:forget", core.Command{Description: "Forget a memory in OpenBrain", Action: s.cmdBrainForget}); !r.OK {
		return r
	}
	if r := c.Command("epic", core.Command{Description: "Create sub-issues from an epic plan", Action: s.cmdEpic}); !r.OK {
		return r
	}
	if r := c.Command("agentic:epic", core.Command{Description: "Create sub-issues from an epic plan", Action: s.cmdEpic}); !r.OK {
		return r
	}
	if r := c.Command("plan-cleanup", core.Command{Description: "Archive old completed plans and delete stale archives past the retention period", Action: s.cmdPlanCleanup}); !r.OK {
		return r
	}
	if r := c.Command("agentic:plan-cleanup", core.Command{Description: "Archive old completed plans and delete stale archives past the retention period", Action: s.cmdPlanCleanup}); !r.OK {
		return r
	}
	if r := c.Command("pr-manage", core.Command{Description: "Manage open PRs (merge, close, review)", Action: s.cmdPRManage}); !r.OK {
		return r
	}
	if r := c.Command("agentic:pr-manage", core.Command{Description: "Manage open PRs (merge, close, review)", Action: s.cmdPRManage}); !r.OK {
		return r
	}
	if r := c.Command("review-queue", core.Command{Description: "Process the CodeRabbit review queue", Action: s.cmdReviewQueue}); !r.OK {
		return r
	}
	if r := c.Command("agentic:review-queue", core.Command{Description: "Process the CodeRabbit review queue", Action: s.cmdReviewQueue}); !r.OK {
		return r
	}
	if r := c.Command("status", core.Command{Description: "List agent workspace statuses", Action: s.cmdStatus}); !r.OK {
		return r
	}
	if r := c.Command("agentic:status", core.Command{Description: "List agent workspace statuses", Action: s.cmdStatus}); !r.OK {
		return r
	}
	if r := c.Command("prompt", core.Command{Description: "Build and display an agent prompt for a repo", Action: s.cmdPrompt}); !r.OK {
		return r
	}
	if r := c.Command("agentic:prompt", core.Command{Description: "Build and display an agent prompt for a repo", Action: s.cmdPrompt}); !r.OK {
		return r
	}
	if r := c.Command("prompt_version", core.Command{Description: "Read the current prompt snapshot for a workspace", Action: s.cmdPromptVersion}); !r.OK {
		return r
	}
	if r := c.Command("agentic:prompt_version", core.Command{Description: "Read the current prompt snapshot for a workspace", Action: s.cmdPromptVersion}); !r.OK {
		return r
	}
	if r := c.Command("prompt/version", core.Command{Description: "Read the current prompt snapshot for a workspace", Action: s.cmdPromptVersion}); !r.OK {
		return r
	}
	if r := c.Command("agentic:prompt/version", core.Command{Description: "Read the current prompt snapshot for a workspace", Action: s.cmdPromptVersion}); !r.OK {
		return r
	}
	if r := c.Command("extract", core.Command{Description: "Extract data from agent output or scaffold a workspace template", Action: s.cmdExtract}); !r.OK {
		return r
	}
	if r := c.Command("agentic:extract", core.Command{Description: "Extract data from agent output or scaffold a workspace template", Action: s.cmdExtract}); !r.OK {
		return r
	}
	for _, register := range []func() core.Result{
		s.registerPlanCommands,
		s.registerCommitCommands,
		s.registerSessionCommands,
		s.registerPhaseCommands,
		s.registerTaskCommands,
		s.registerSprintCommands,
		s.registerStateCommands,
		s.registerCoreCommands,
		s.registerFleetCommands,
		s.registerPipelineCommands,
		s.registerLanguageCommands,
		s.registerSetupCommands,
	} {
		if result := register(); !result.OK {
			return result
		}
	}
	return core.Result{OK: true}
}

// ctx := s.commandContext()
func (s *PrepSubsystem) commandContext() context.Context {
	if s.startupContext != nil {
		return s.startupContext
	}
	return context.Background()
}

func (s *PrepSubsystem) cmdRunTask(options core.Options) core.Result {
	return s.runDispatchSync(s.commandContext(), options, "run task", "agentic.runTask")
}

func (s *PrepSubsystem) cmdRunFlow(options core.Options) core.Result {
	return s.runFlowExecutionCommand(options, "run flow")
}

func (s *PrepSubsystem) cmdFlowPreview(options core.Options) core.Result {
	return s.runFlowCommand(options, "flow preview")
}

func (s *PrepSubsystem) runFlowCommand(options core.Options, commandLabel string) core.Result {
	flowPath := optionStringValue(options, "_arg", `path`, "slug")
	if flowPath == "" {
		core.Print(nil, "usage: core-agent %s <path-or-slug> [--dry-run] [--var=key=value] [--vars='{\"key\":\"value\"}'] [--variables='{\"key\":\"value\"}']", commandLabel)
		return core.Result{Value: core.E("agentic.cmdRunFlow", "flow path or slug is required", nil), OK: false}
	}

	dryRun := optionBoolValue(options, "dry_run", "dry-run")
	variables := optionStringMapValue(options, "var", "vars", "variables")

	flowResult := readFlowDocument(flowPath, variables)
	if !flowResult.OK {
		core.Print(nil, "error: %v", flowResult.Value)
		return core.Result{Value: flowResult.Value, OK: false}
	}

	document, ok := flowResult.Value.(flowRunDocument)
	if !ok {
		err := core.E("agentic.cmdRunFlow", "invalid flow definition", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output := FlowRunOutput{
		Success:     true,
		Source:      document.Source,
		Name:        document.Definition.Name,
		Description: document.Definition.Description,
		Steps:       len(document.Definition.Steps),
		Parsed:      document.Parsed,
	}

	core.Print(nil, "flow:  %s", document.Source)
	if dryRun {
		core.Print(nil, "dry-run: true")
	}
	if len(variables) > 0 {
		core.Print(nil, "vars: %d", len(variables))
	}
	if document.Parsed {
		if document.Definition.Name != "" {
			core.Print(nil, "name:  %s", document.Definition.Name)
		}
		if document.Definition.Description != "" {
			core.Print(nil, "desc:  %s", document.Definition.Description)
		}
		if len(document.Definition.Steps) == 0 {
			core.Print(nil, "steps: 0")
			return core.Result{Value: output, OK: true}
		}

		core.Print(nil, "steps: %d", len(document.Definition.Steps))
		resolvedSteps := s.printFlowSteps(document, "", variables, map[string]bool{document.Source: true})
		output.ResolvedSteps = resolvedSteps
		if resolvedSteps != len(document.Definition.Steps) {
			core.Print(nil, "resolved steps: %d", resolvedSteps)
		}
		return core.Result{Value: output, OK: true}
	}

	if document.Content != "" {
		core.Print(nil, "content: %d chars", len(document.Content))
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdDispatchSync(options core.Options) core.Result {
	return s.runDispatchSync(s.commandContext(), options, "dispatch sync", "agentic.runDispatchSync")
}

func (s *PrepSubsystem) runDispatchSync(ctx context.Context, options core.Options, commandLabel, errorName string) core.Result {
	repo := options.String("repo")
	agent := options.String("agent")
	task := options.String("task")
	issueValue := options.String("issue")
	org := options.String("org")

	if repo == "" || task == "" {
		core.Print(nil, "usage: core-agent %s --repo=<repo> --task=\"...\" --agent=codex [--issue=N] [--org=core]", commandLabel)
		return core.Result{Value: core.E(errorName, "repo and task are required", nil), OK: false}
	}
	if agent == "" {
		agent = "codex"
	}
	if org == "" {
		org = "core"
	}

	issue := parseIntString(issueValue)

	core.Print(nil, "core-agent %s", commandLabel)
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
			failureError = core.E(errorName, "dispatch failed", nil)
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
	return s.runDispatchLoop("orchestrator")
}

func (s *PrepSubsystem) cmdDispatch(_ core.Options) core.Result {
	return s.runDispatchLoop("dispatch")
}

func (s *PrepSubsystem) cmdDispatchStart(_ core.Options) core.Result {
	_, output, err := dispatchStart(s, s.commandContext(), nil, ShutdownInput{})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Message != "" {
		core.Print(nil, "%s", output.Message)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdDispatchShutdown(_ core.Options) core.Result {
	_, output, err := shutdownGraceful(s, s.commandContext(), nil, ShutdownInput{})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Message != "" {
		core.Print(nil, "%s", output.Message)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdDispatchShutdownNow(_ core.Options) core.Result {
	_, output, err := shutdownNow(s, s.commandContext(), nil, ShutdownInput{})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Message != "" {
		core.Print(nil, "%s", output.Message)
	}
	if output.Running > 0 || output.Queued > 0 {
		core.Print(nil, "running: %d", output.Running)
		core.Print(nil, "queued:  %d", output.Queued)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPoke(_ core.Options) core.Result {
	s.Poke()
	core.Print(nil, "queue poke requested")
	return core.Result{OK: true}
}

func (s *PrepSubsystem) runDispatchLoop(label string) core.Result {
	ctx := s.commandContext()
	core.Print(nil, "core-agent %s running (pid %s)", label, core.Env("PID"))
	core.Print(nil, "  workspace: %s", WorkspaceRoot())
	core.Print(nil, "  watching queue, draining on 30s tick + completion poke")

	<-ctx.Done()
	core.Print(nil, "%s shutting down", label)
	return core.Result{OK: true}
}

// emitCommandJSON prints v as JSON when --json is set, returning true if it
// did (the caller then returns without its human-formatted output). The
// agentic verbs serve two callers: a human at the terminal (default, formatted)
// and the desktop CLI adapter (--json, machine-parseable) — the same split
// pkg/calibrate relies on for lthn-mlx.
func emitCommandJSON(options core.Options, v any) bool {
	if !optionBoolValue(options, "json") {
		return false
	}
	core.Print(nil, "%s", core.JSONMarshalString(v))
	return true
}

func (s *PrepSubsystem) cmdPrep(options core.Options) core.Result {
	repo := options.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent prep <repo> --issue=N|--pr=N|--branch=X --task=\"...\" [--plan-template=bug-fix] [--variables='{\"key\":\"value\"}']")
		return core.Result{Value: core.E("agentic.cmdPrep", "repo is required", nil), OK: false}
	}

	prepInput := prepInputFromCommandOptions(options)
	prepInput.Repo = repo

	if prepInput.Issue == 0 && prepInput.PR == 0 && prepInput.Branch == "" && prepInput.Tag == "" {
		prepInput.Branch = "dev"
	}

	_, prepOutput, err := PrepareWorkspace(s, context.Background(), prepInput)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if emitCommandJSON(options, prepOutput) {
		return core.Result{Value: prepOutput, OK: true}
	}

	core.Print(nil, "workspace: %s", prepOutput.WorkspaceDir)
	core.Print(nil, "repo:      %s", prepOutput.RepoDir)
	core.Print(nil, "branch:    %s", prepOutput.Branch)
	if prepOutput.PromptVersion != "" {
		core.Print(nil, "prompt:    %s", prepOutput.PromptVersion)
	}
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

func (s *PrepSubsystem) cmdResume(options core.Options) core.Result {
	workspace := optionStringValue(options, "workspace", "_arg")
	if workspace == "" {
		core.Print(nil, "usage: core-agent resume <workspace> [--answer=\"...\"] [--agent=codex] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdResume", "workspace is required", nil), OK: false}
	}

	result := typedResultValue[ResumeOutput]("agentic.cmdResume", "invalid resume output", s.resume(s.commandContext(), ResumeInput{
		Workspace: workspace,
		Answer:    optionStringValue(options, "answer"),
		Agent:     optionStringValue(options, "agent"),
		DryRun:    optionBoolValue(options, "dry_run", "dry-run"),
	}))
	if !result.OK {
		core.Print(nil, "error: %s", result.Error())
		return result
	}
	output, _ := result.Value.(ResumeOutput)

	if emitCommandJSON(options, output) {
		return core.Result{Value: output, OK: true}
	}

	core.Print(nil, "workspace:  %s", output.Workspace)
	core.Print(nil, "agent:      %s", output.Agent)
	if output.PID > 0 {
		core.Print(nil, "pid:        %d", output.PID)
	}
	if output.OutputFile != "" {
		core.Print(nil, "output:     %s", output.OutputFile)
	}
	if output.Prompt != "" {
		core.Print(nil, "")
		core.Print(nil, "--- prompt (%d chars) ---", len(output.Prompt))
		core.Print(nil, "%s", output.Prompt)
	}
	return core.Result{Value: output, OK: true}
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

func (s *PrepSubsystem) cmdContentSchemaGenerate(options core.Options) core.Result {
	schemaType := optionStringValue(options, "schema_type", "schema-type", "type", "kind")
	title := optionStringValue(options, "title", "headline")
	if schemaType == "" || title == "" {
		core.Print(nil, "usage: core-agent content schema generate --type=howto --title=\"Set up the workspace\" [--description=\"...\"] [--url=\"https://example.test/setup\"] [--author=\"Virgil\"] [--questions='[{\"question\":\"...\",\"answer\":\"...\"}]'] [--steps='[{\"name\":\"...\",\"text\":\"...\"}]']")
		return core.Result{Value: core.E("agentic.cmdContentSchemaGenerate", "schema type and title are required", nil), OK: false}
	}

	result := s.handleContentSchemaGenerate(s.commandContext(), core.NewOptions(
		core.Option{Key: "schema_type", Value: schemaType},
		core.Option{Key: "title", Value: title},
		core.Option{Key: "description", Value: optionStringValue(options, "description")},
		core.Option{Key: "url", Value: optionStringValue(options, "url", "link")},
		core.Option{Key: "author", Value: optionStringValue(options, "author")},
		core.Option{Key: "published_at", Value: optionStringValue(options, "published_at", "published-at", "date_published")},
		core.Option{Key: "modified_at", Value: optionStringValue(options, "modified_at", "modified-at", "date_modified")},
		core.Option{Key: "language", Value: optionStringValue(options, "language", "in_language", "in-language")},
		core.Option{Key: "image", Value: optionStringValue(options, "image", "image_url", "image-url")},
		core.Option{Key: "questions", Value: optionAnyValue(options, "questions", "faq")},
		core.Option{Key: "steps", Value: optionAnyValue(options, "steps")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdContentSchemaGenerate", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(ContentSchemaOutput)
	if !ok {
		err := core.E("agentic.cmdContentSchemaGenerate", "invalid content schema output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "schema type: %s", output.SchemaType)
	core.Print(nil, "schema json: %s", output.SchemaJSON)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdComplete(options core.Options) core.Result {
	ctx := s.commandContext()
	result := s.handleComplete(ctx, options)
	if !result.OK {
		err := commandResultError("agentic.cmdComplete", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	workspace := optionStringValue(options, "workspace", "_arg")
	cleanupResult := s.cleanupWorkspaceBranch(ctx, workspace)
	if !cleanupResult.OK {
		core.Warn("agentic.cmdComplete: branch cleanup failed", "workspace", workspace, "reason", cleanupResult.Value)
	}

	return result
}

func (s *PrepSubsystem) cmdScan(options core.Options) core.Result {
	result := s.handleScan(s.commandContext(), core.NewOptions(
		core.Option{Key: "org", Value: optionStringValue(options, "org")},
		core.Option{Key: "labels", Value: optionStringSliceValue(options, "labels")},
		core.Option{Key: "limit", Value: optionIntValue(options, "limit")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdScan", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(ScanOutput)
	if !ok {
		err := core.E("agentic.cmdScan", "invalid scan output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if emitCommandJSON(options, output) {
		return core.Result{Value: output, OK: true}
	}

	core.Print(nil, "count: %d", output.Count)
	for _, issue := range output.Issues {
		if len(issue.Labels) > 0 {
			core.Print(nil, "  %s#%d %s [%s]", issue.Repo, issue.Number, issue.Title, core.Join(",", issue.Labels...))
			continue
		}
		core.Print(nil, "  %s#%d %s", issue.Repo, issue.Number, issue.Title)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdMirror(options core.Options) core.Result {
	result := s.handleMirror(s.commandContext(), core.NewOptions(
		core.Option{Key: "repo", Value: optionStringValue(options, "repo", "_arg")},
		core.Option{Key: "dry_run", Value: optionBoolValue(options, "dry_run", "dry-run")},
		core.Option{Key: "max_files", Value: optionIntValue(options, "max_files", "max-files")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdMirror", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(MirrorOutput)
	if !ok {
		err := core.E("agentic.cmdMirror", "invalid mirror output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "count: %d", output.Count)
	for _, item := range output.Synced {
		core.Print(nil, "  %s commits=%d files=%d", item.Repo, item.CommitsAhead, item.FilesChanged)
		if item.PRURL != "" {
			core.Print(nil, "    pr: %s", item.PRURL)
		}
		if item.Skipped != "" {
			core.Print(nil, "    %s", item.Skipped)
		}
	}
	for _, skipped := range output.Skipped {
		core.Print(nil, "skipped: %s", skipped)
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdBrainList(options core.Options) core.Result {
	result := s.Core().Action("brain.list").Run(s.commandContext(), core.NewOptions(
		core.Option{Key: "project", Value: optionStringValue(options, "project")},
		core.Option{Key: "type", Value: optionStringValue(options, "type")},
		core.Option{Key: "agent_id", Value: optionStringValue(options, "agent_id", "agent")},
		core.Option{Key: "limit", Value: optionIntValue(options, "limit")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdBrainList", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	payload, ok := result.Value.(map[string]any)
	if !ok {
		jsonResult := core.JSONMarshalString(result.Value)
		if jsonResult == "" {
			err := core.E("agentic.cmdBrainList", "invalid brain list output", nil)
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}
		var decoded any
		if parseResult := core.JSONUnmarshalString(jsonResult, &decoded); !parseResult.OK {
			err, _ := parseResult.Value.(error)
			if err == nil {
				err = core.E("agentic.cmdBrainList", "invalid brain list output", nil)
			}
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}
		payload, ok = decoded.(map[string]any)
		if !ok {
			err := core.E("agentic.cmdBrainList", "invalid brain list output", nil)
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}
	}

	output := brainListOutputFromPayload(payload)
	core.Print(nil, "count: %d", output.Count)
	if len(output.Memories) == 0 {
		core.Print(nil, "no memories")
		return core.Result{Value: output, OK: true}
	}

	for _, memory := range output.Memories {
		if memory.Project != "" || memory.AgentID != "" || memory.Confidence != 0 {
			core.Print(nil, "  %s %-12s %s %s %.2f", memory.ID, memory.Type, memory.Project, memory.AgentID, memory.Confidence)
		} else {
			core.Print(nil, "  %s %-12s", memory.ID, memory.Type)
		}
		if memory.SupersedesCount > 0 {
			core.Print(nil, "    supersedes: %d", memory.SupersedesCount)
		}
		if memory.DeletedAt != "" {
			core.Print(nil, "    deleted_at: %s", memory.DeletedAt)
		}
		if memory.Content != "" {
			core.Print(nil, "    %s", memory.Content)
		}
	}

	return core.Result{Value: output, OK: true}
}

// result := c.Command("brain/remember").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "content", Value: "Use named actions."},
//	core.Option{Key: "type", Value: "convention"},
//
// ))
func (s *PrepSubsystem) cmdBrainRemember(options core.Options) core.Result {
	content := optionStringValue(options, "content", "_arg")
	memoryType := optionStringValue(options, "type")
	if core.Trim(content) == "" || memoryType == "" {
		core.Print(nil, "usage: core-agent brain remember <content> --type=observation [--tags=architecture,convention] [--project=agent] [--confidence=0.8] [--supersedes=mem-123] [--expires-in=24]")
		return core.Result{Value: core.E("agentic.cmdBrainRemember", "content and type are required", nil), OK: false}
	}

	result := s.Core().Action("brain.remember").Run(s.commandContext(), core.NewOptions(
		core.Option{Key: "content", Value: content},
		core.Option{Key: "type", Value: memoryType},
		core.Option{Key: "tags", Value: optionStringSliceValue(options, "tags")},
		core.Option{Key: "project", Value: optionStringValue(options, "project")},
		core.Option{Key: "confidence", Value: optionStringValue(options, "confidence")},
		core.Option{Key: "supersedes", Value: optionStringValue(options, "supersedes")},
		core.Option{Key: "expires_in", Value: optionIntValue(options, "expires_in", "expires-in")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdBrainRemember", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	jsonResult := core.JSONMarshalString(result.Value)
	if jsonResult == "" {
		err := core.E("agentic.cmdBrainRemember", "invalid brain remember output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	var decoded map[string]any
	if parseResult := core.JSONUnmarshalString(jsonResult, &decoded); !parseResult.OK {
		err := core.E("agentic.cmdBrainRemember", "invalid brain remember output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	memoryID := stringValue(decoded["memoryId"])
	if memoryID == "" {
		memoryID = stringValue(decoded["memory_id"])
	}
	core.Print(nil, "remembered: %s", memoryID)
	if timestamp := stringValue(decoded["timestamp"]); timestamp != "" {
		core.Print(nil, "timestamp:  %s", timestamp)
	}
	return core.Result{Value: decoded, OK: true}
}

// result := c.Command("brain/recall").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "query", Value: "workspace handoff context"},
//
// ))
func (s *PrepSubsystem) cmdBrainRecall(options core.Options) core.Result {
	query := optionStringValue(options, "query", "_arg")
	if query == "" {
		core.Print(nil, "usage: core-agent brain recall <query> [--top-k=10] [--project=agent] [--type=architecture] [--agent=virgil] [--min-confidence=0.7]")
		return core.Result{Value: core.E("agentic.cmdBrainRecall", "query is required", nil), OK: false}
	}

	result := s.Core().Action("brain.recall").Run(s.commandContext(), core.NewOptions(
		core.Option{Key: "query", Value: query},
		core.Option{Key: "top_k", Value: optionIntValue(options, "top_k", "top-k")},
		core.Option{Key: "project", Value: optionStringValue(options, "project")},
		core.Option{Key: "type", Value: optionStringValue(options, "type")},
		core.Option{Key: "agent_id", Value: optionStringValue(options, "agent_id", "agent")},
		core.Option{Key: "min_confidence", Value: optionStringValue(options, "min_confidence", "min-confidence")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdBrainRecall", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := brainRecallOutputFromResult(result.Value)
	if !ok {
		err := core.E("agentic.cmdBrainRecall", "invalid brain recall output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "count: %d", output.Count)
	if len(output.Memories) == 0 {
		core.Print(nil, "no memories")
		return core.Result{Value: output, OK: true}
	}

	for _, memory := range output.Memories {
		if memory.Project != "" || memory.AgentID != "" || memory.Confidence != 0 {
			core.Print(nil, "  %s %-12s %s %s %.2f", memory.ID, memory.Type, memory.Project, memory.AgentID, memory.Confidence)
		} else {
			core.Print(nil, "  %s %-12s", memory.ID, memory.Type)
		}
		if memory.Content != "" {
			core.Print(nil, "    %s", memory.Content)
		}
	}

	return core.Result{Value: output, OK: true}
}

// result := c.Command("brain/forget").Run(ctx, core.NewOptions(core.Option{Key: "_arg", Value: "mem-1"}))
func (s *PrepSubsystem) cmdBrainForget(options core.Options) core.Result {
	id := optionStringValue(options, "id", "_arg")
	if id == "" {
		core.Print(nil, "usage: core-agent brain forget <memory-id> [--reason=\"superseded\"]")
		return core.Result{Value: core.E("agentic.cmdBrainForget", "memory id is required", nil), OK: false}
	}

	result := s.Core().Action("brain.forget").Run(s.commandContext(), core.NewOptions(
		core.Option{Key: "id", Value: id},
		core.Option{Key: "reason", Value: optionStringValue(options, "reason")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdBrainForget", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "forgotten: %s", id)
	if reason := optionStringValue(options, "reason"); reason != "" {
		core.Print(nil, "reason:    %s", reason)
	}
	return core.Result{Value: result.Value, OK: true}
}

type brainRecallOutput struct {
	Count    int                 `json:"count"`
	Memories []brainRecallMemory `json:"memories"`
}

type brainRecallMemory struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Content    string   `json:"content"`
	Project    string   `json:"project"`
	AgentID    string   `json:"agent_id"`
	Confidence float64  `json:"confidence"`
	DeletedAt  string   `json:"deleted_at,omitempty"`
	Tags       []string `json:"tags"`
}

func brainRecallOutputFromResult(value any) (brainRecallOutput, bool) {
	switch typed := value.(type) {
	case brainRecallOutput:
		return typed, true
	case *brainRecallOutput:
		if typed == nil {
			return brainRecallOutput{}, false
		}
		return *typed, true
	default:
		jsonResult := core.JSONMarshalString(value)
		if jsonResult == "" {
			return brainRecallOutput{}, false
		}
		var output brainRecallOutput
		if parseResult := core.JSONUnmarshalString(jsonResult, &output); !parseResult.OK {
			return brainRecallOutput{}, false
		}
		return output, true
	}
}

func (s *PrepSubsystem) cmdStatus(options core.Options) core.Result {
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

	requestedWorkspace := optionStringValue(options, "workspace", "_arg")
	requestedStatus := optionStringValue(options, "status")
	limit := optionIntValue(options, "limit")
	matched := 0

	for _, sf := range statusFiles {
		workspaceDir := core.PathDir(sf)
		workspaceName := WorkspaceName(workspaceDir)
		if !statusInputMatchesWorkspace(requestedWorkspace, workspaceDir, workspaceName) {
			continue
		}

		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}

		if !statusInputMatchesStatus(requestedStatus, workspaceStatus.Status) {
			continue
		}

		core.Print(nil, "  %-8s %-8s %-10s %s", workspaceStatus.Status, workspaceStatus.Agent, workspaceStatus.Repo, workspaceName)
		if workspaceStatus.Question != "" {
			core.Print(nil, "    question: %s", workspaceStatus.Question)
		}
		matched++
		if limit > 0 && matched >= limit {
			break
		}
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdPrompt(options core.Options) core.Result {
	repo := options.String("_arg")
	if repo == "" {
		core.Print(nil, "usage: core-agent prompt <repo> --task=\"...\" [--plan-template=bug-fix] [--variables='{\"key\":\"value\"}']")
		return core.Result{Value: core.E("agentic.cmdPrompt", "repo is required", nil), OK: false}
	}

	prepInput := prepInputFromCommandOptions(options)
	prepInput.Repo = repo
	if prepInput.Org == "" {
		prepInput.Org = "core"
	}
	if prepInput.Task == "" {
		prepInput.Task = "Review and report findings"
	}

	repoPath := core.JoinPath(HomeDir(), "Code", prepInput.Org, prepInput.Repo)

	prompt, memories, consumers := s.BuildPrompt(context.Background(), prepInput, "dev", repoPath)
	core.Print(nil, "memories:  %d", memories)
	core.Print(nil, "consumers: %d", consumers)
	core.Print(nil, "")
	core.Print(nil, "%s", prompt)
	return core.Result{OK: true}
}

func prepInputFromCommandOptions(options core.Options) PrepInput {
	commandOptions := core.NewOptions(options.Items()...)
	if commandOptions.String("repo") == "" {
		if repo := optionStringValue(options, "_arg"); repo != "" {
			commandOptions.Set("repo", repo)
		}
	}
	return prepInputFromOptions(commandOptions)
}

func (s *PrepSubsystem) cmdPromptVersion(options core.Options) core.Result {
	workspace := optionStringValue(options, "workspace", "_arg")
	if workspace == "" {
		core.Print(nil, "usage: core-agent prompt version <workspace>")
		return core.Result{Value: core.E("agentic.cmdPromptVersion", "workspace is required", nil), OK: false}
	}

	result := s.handlePromptVersion(s.commandContext(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspace},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdPromptVersion", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(PromptVersionOutput)
	if !ok {
		err := core.E("agentic.cmdPromptVersion", "invalid prompt version output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "workspace: %s", output.Workspace)
	core.Print(nil, "hash:      %s", output.Snapshot.Hash)
	if output.Snapshot.CreatedAt != "" {
		core.Print(nil, "created:   %s", output.Snapshot.CreatedAt)
	}
	core.Print(nil, "chars:     %d", len(output.Snapshot.Content))
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdExtract(options core.Options) core.Result {
	sourcePath := optionStringValue(options, "source", "input", "file")
	templateName := options.String("_arg")
	if templateName == "" {
		templateName = "default"
	}
	target := options.String("target")

	if sourcePath == "" && fs.Exists(templateName) && fs.IsFile(templateName) {
		sourcePath = templateName
		templateName = ""
	}

	if sourcePath != "" {
		readResult := fs.Read(sourcePath)
		if !readResult.OK {
			err, _ := readResult.Value.(error)
			if err == nil {
				err = core.E("agentic.cmdExtract", core.Concat("read agent output ", sourcePath), nil)
			}
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}

		extracted := extractAgentOutputContent(readResult.Value.(string))
		if extracted == "" {
			err := core.E("agentic.cmdExtract", "agent output did not contain extractable content", nil)
			core.Print(nil, "error: %v", err)
			return core.Result{Value: err, OK: false}
		}

		if target != "" {
			if writeResult := fs.WriteMode(target, extracted, 0644); !writeResult.OK {
				err, _ := writeResult.Value.(error)
				if err == nil {
					err = core.E("agentic.cmdExtract", core.Concat("write extracted output ", target), nil)
				}
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}
			core.Print(nil, "written: %s", target)
		} else {
			core.Print(nil, "%s", extracted)
		}

		return core.Result{Value: extracted, OK: true}
	}

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

func extractAgentOutputContent(content string) string {
	trimmed := core.Trim(content)
	if trimmed == "" {
		return ""
	}

	if core.HasPrefix(trimmed, "{") || core.HasPrefix(trimmed, "[") {
		return trimmed
	}

	blocks := core.Split(content, "```")
	for index := 1; index < len(blocks); index += 2 {
		block := core.Trim(blocks[index])
		if block == "" {
			continue
		}

		lines := core.SplitN(block, "\n", 2)
		if len(lines) == 2 {
			language := core.Trim(lines[0])
			body := core.Trim(lines[1])
			if language != "" && !core.Contains(language, " ") && body != "" {
				block = body
			}
		}

		block = core.Trim(block)
		if block != "" {
			return block
		}
	}

	return ""
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

type FlowRunOutput struct {
	Success       bool                `json:"success"`
	Source        string              `json:"source,omitempty"`
	Name          string              `json:"name,omitempty"`
	Description   string              `json:"description,omitempty"`
	Steps         int                 `json:"steps,omitempty"`
	ResolvedSteps int                 `json:"resolved_steps,omitempty"`
	Parsed        bool                `json:"parsed,omitempty"`
	Executed      int                 `json:"executed,omitempty"`
	Passed        int                 `json:"passed,omitempty"`
	Failed        int                 `json:"failed,omitempty"`
	StepResults   []FlowRunStepOutput `json:"step_results,omitempty"`
}

type flowDefinition struct {
	Name        string               `yaml:"name"`
	Description string               `yaml:"description"`
	Inputs      []flow.Input         `yaml:"inputs"`
	Steps       []flowDefinitionStep `yaml:"steps"`
}

type flowDefinitionStep struct {
	Name            string               `yaml:"name"`
	Cmd             string               `yaml:"cmd"`
	Args            []string             `yaml:"args"`
	Run             string               `yaml:"run"`
	Flow            string               `yaml:"flow"`
	With            map[string]string    `yaml:"with"`
	Agent           string               `yaml:"agent"`
	Prompt          string               `yaml:"prompt"`
	Template        string               `yaml:"template"`
	Timeout         string               `yaml:"timeout"`
	When            string               `yaml:"when"`
	Output          string               `yaml:"output"`
	Gate            string               `yaml:"gate"`
	ContinueOnError bool                 `yaml:"continueOnError"`
	Parallel        []flowDefinitionStep `yaml:"parallel"`
}

type flowRunDocument struct {
	Source     string
	Content    string
	Parsed     bool
	Definition flowDefinition
}

func readFlowDocument(path string, variables map[string]string) core.Result {
	if readResult := fs.Read(path); readResult.OK {
		content := applyTemplateVariables(readResult.Value.(string), variables)
		definition, err := parseFlowDefinition(content)
		if err != nil {
			if flowInputLooksYaml(path) {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{Value: flowRunDocument{
				Source:  path,
				Content: content,
				Parsed:  false,
			}, OK: true}
		}
		return core.Result{Value: flowRunDocument{
			Source:     path,
			Content:    content,
			Parsed:     true,
			Definition: definition,
		}, OK: true}
	}

	flowResult := lib.Flow(flowSlugFromPath(path))
	if !flowResult.OK {
		if err, ok := flowResult.Value.(error); ok {
			return core.Result{Value: core.E("agentic.cmdRunFlow", core.Concat("flow not found: ", path), err), OK: false}
		}
		return core.Result{Value: core.E("agentic.cmdRunFlow", core.Concat("flow not found: ", path), nil), OK: false}
	}

	content := applyTemplateVariables(flowResult.Value.(string), variables)
	definition, err := parseFlowDefinition(content)
	if err != nil {
		return core.Result{Value: flowRunDocument{
			Source:  core.Concat("embedded:", flowSlugFromPath(path)),
			Content: content,
			Parsed:  false,
		}, OK: true}
	}
	return core.Result{Value: flowRunDocument{
		Source:     core.Concat("embedded:", flowSlugFromPath(path)),
		Content:    content,
		Parsed:     true,
		Definition: definition,
	}, OK: true}
}

var parseFlowDefinition = func(content string) (flowDefinition, error) {
	var definition flowDefinition
	if err := yaml.Unmarshal([]byte(content), &definition); err != nil {
		return flowDefinition{}, core.E("agentic.parseFlowDefinition", "invalid flow definition", err)
	}
	if definition.Name == "" {
		return flowDefinition{}, core.E("agentic.parseFlowDefinition", "invalid flow definition", nil)
	}
	return definition, nil
}

func flowInputLooksYaml(path string) bool {
	return core.HasSuffix(path, ".yaml") || core.HasSuffix(path, ".yml")
}

func flowSlugFromPath(path string) string {
	slug := core.PathBase(path)
	for _, suffix := range []string{".yaml", ".yml", ".md"} {
		slug = core.TrimSuffix(slug, suffix)
	}
	return slug
}

func flowStepSummary(step flowDefinitionStep) string {
	label := core.Trim(step.Name)
	if label == "" {
		label = core.Trim(step.Flow)
	}
	if label == "" {
		label = core.Trim(step.Cmd)
	}
	if label == "" {
		label = core.Trim(step.Agent)
	}
	if label == "" {
		label = core.Trim(step.Run)
	}
	if label == "" {
		label = "step"
	}

	switch {
	case step.Flow != "":
		return core.Concat(label, ": flow ", step.Flow)
	case step.Cmd != "":
		return core.Concat(label, ": cmd ", flowStepCommandLine(step))
	case step.Agent != "":
		return core.Concat(label, ": agent ", step.Agent)
	case step.Run != "":
		return core.Concat(label, ": run ", step.Run)
	case step.Gate != "":
		return core.Concat(label, ": gate ", step.Gate)
	default:
		return label
	}
}

func (s *PrepSubsystem) printFlowSteps(document flowRunDocument, indent string, variables map[string]string, visited map[string]bool) int {
	total := 0
	for index, step := range document.Definition.Steps {
		core.Print(nil, "%s%d. %s", indent, index+1, flowStepSummary(step))
		total++

		if step.Flow != "" {
			resolved := s.resolveFlowReference(document.Source, step.Flow, variables)
			if !resolved.OK {
				continue
			}

			nested, ok := resolved.Value.(flowRunDocument)
			if !ok || nested.Source == "" {
				continue
			}
			if visited[nested.Source] {
				core.Print(nil, "%s  cycle: %s", indent, nested.Source)
				continue
			}

			core.Print(nil, "%s  resolved: %s", indent, nested.Source)
			visited[nested.Source] = true
			total += s.printFlowSteps(nested, core.Concat(indent, "  "), variables, visited)
			delete(visited, nested.Source)
		}

		if len(step.Parallel) > 0 {
			core.Print(nil, "%s  parallel:", indent)
			for parallelIndex, parallelStep := range step.Parallel {
				core.Print(nil, "%s    %d. %s", indent, parallelIndex+1, flowStepSummary(parallelStep))
			}
		}
	}

	return total
}

func (s *PrepSubsystem) resolveFlowReference(baseSource, reference string, variables map[string]string) core.Result {
	trimmedReference := core.Trim(reference)
	if trimmedReference == "" {
		return core.Result{Value: core.E("agentic.resolveFlowReference", "flow reference is required", nil), OK: false}
	}

	candidates := []string{trimmedReference}

	if root := flowRootPath(baseSource); root != "" {
		candidate := core.JoinPath(root, trimmedReference)
		if candidate != trimmedReference {
			candidates = append(candidates, candidate)
		}
	}

	repoCandidate := core.JoinPath("pkg", "lib", "flow", trimmedReference)
	if repoCandidate != trimmedReference {
		candidates = append(candidates, repoCandidate)
	}

	for _, candidate := range candidates {
		result := readFlowDocument(candidate, variables)
		if result.OK {
			return result
		}

		err, ok := result.Value.(error)
		if !ok || !core.Contains(err.Error(), "flow not found:") {
			return result
		}
	}

	return core.Result{Value: core.E("agentic.resolveFlowReference", core.Concat("flow not found: ", trimmedReference), nil), OK: false}
}

func flowRootPath(source string) string {
	trimmed := core.Trim(core.Replace(source, "\\", "/"))
	if trimmed == "" {
		return ""
	}

	segments := core.Split(trimmed, "/")
	for index := 0; index+2 < len(segments); index++ {
		if segments[index] == "pkg" && segments[index+1] == "lib" && segments[index+2] == "flow" {
			return core.JoinPath(segments[:index+3]...)
		}
	}

	dir := core.PathDir(trimmed)
	if dir != "" {
		return dir
	}

	return ""
}

type brainListOutput struct {
	Count    int                    `json:"count"`
	Memories []brainListOutputEntry `json:"memories"`
}

type brainListOutputEntry struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	Content         string   `json:"content"`
	Project         string   `json:"project"`
	AgentID         string   `json:"agent_id"`
	Confidence      float64  `json:"confidence"`
	SupersedesCount int      `json:"supersedes_count,omitempty"`
	DeletedAt       string   `json:"deleted_at,omitempty"`
	Tags            []string `json:"tags"`
}

func brainListOutputFromPayload(payload map[string]any) brainListOutput {
	output := brainListOutput{}
	switch count := payload["count"].(type) {
	case float64:
		output.Count = int(count)
	case int:
		output.Count = count
	}
	if memories, ok := payload["memories"].([]any); ok {
		for _, item := range memories {
			entryMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			entry := brainListOutputEntry{
				ID:      brainListStringValue(entryMap["id"]),
				Type:    brainListStringValue(entryMap["type"]),
				Content: brainListStringValue(entryMap["content"]),
				Project: brainListStringValue(entryMap["project"]),
				AgentID: brainListStringValue(entryMap["agent_id"]),
			}
			switch confidence := entryMap["confidence"].(type) {
			case float64:
				entry.Confidence = confidence
			case int:
				entry.Confidence = float64(confidence)
			}
			if entry.Confidence == 0 {
				switch confidence := entryMap["score"].(type) {
				case float64:
					entry.Confidence = confidence
				case int:
					entry.Confidence = float64(confidence)
				}
			}
			switch supersedesCount := entryMap["supersedes_count"].(type) {
			case float64:
				entry.SupersedesCount = int(supersedesCount)
			case int:
				entry.SupersedesCount = supersedesCount
			}
			entry.DeletedAt = brainListStringValue(entryMap["deleted_at"])
			if tags, ok := entryMap["tags"].([]any); ok {
				for _, tag := range tags {
					entry.Tags = append(entry.Tags, brainListStringValue(tag))
				}
			}
			output.Memories = append(output.Memories, entry)
		}
	}
	if output.Count == 0 {
		output.Count = len(output.Memories)
	}
	return output
}

func brainListStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return core.Sprint(typed)
	case int64:
		return core.Sprint(typed)
	case float64:
		return core.Sprint(typed)
	}
	return ""
}
