// SPDX-License-Identifier: EUPL-1.2

// Named Action handlers for the agentic service.
// Each handler adapts (ctx, Options) → Result to call the existing MCP tool method.
// Registered during OnStartup — the Action registry IS the capability map.
//
//	c.Action("agentic.dispatch").Run(ctx, options)
//	c.Actions() // all registered capabilities

package agentic

import (
	"context"

	"dappco.re/go/agent/pkg/lib"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// --- Dispatch & Workspace ---

// handleDispatch dispatches a subagent to work on a repo task.
//
//	result := c.Action("agentic.dispatch").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	    core.Option{Key: "task", Value: "Fix tests"},
//	))
func (s *PrepSubsystem) handleDispatch(ctx context.Context, options core.Options) core.Result {
	input := DispatchInput{
		Repo:  options.String("repo"),
		Task:  options.String("task"),
		Agent: options.String("agent"),
		Issue: options.Int("issue"),
	}
	_, out, err := s.dispatch(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handlePrep prepares a workspace without dispatching an agent.
//
//	result := c.Action("agentic.prep").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	    core.Option{Key: "issue", Value: 42},
//	))
func (s *PrepSubsystem) handlePrep(ctx context.Context, options core.Options) core.Result {
	input := PrepInput{
		Repo:  options.String("repo"),
		Org:   options.String("org"),
		Issue: options.Int("issue"),
	}
	_, out, err := s.prepWorkspace(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleStatus lists workspace statuses.
//
//	result := c.Action("agentic.status").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleStatus(ctx context.Context, options core.Options) core.Result {
	input := StatusInput{
		Workspace: options.String("workspace"),
		Limit:     options.Int("limit"),
		Status:    options.String("status"),
	}
	_, out, err := s.status(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleResume resumes a blocked workspace.
//
//	result := c.Action("agentic.resume").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	))
func (s *PrepSubsystem) handleResume(ctx context.Context, options core.Options) core.Result {
	input := ResumeInput{
		Workspace: options.String("workspace"),
		Answer:    options.String("answer"),
	}
	_, out, err := s.resume(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleScan scans forge repos for actionable issues.
//
//	result := c.Action("agentic.scan").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleScan(ctx context.Context, options core.Options) core.Result {
	input := ScanInput{
		Org:   options.String("org"),
		Limit: options.Int("limit"),
	}
	_, out, err := s.scan(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleWatch watches a workspace for completion.
//
//	result := c.Action("agentic.watch").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	))
func (s *PrepSubsystem) handleWatch(ctx context.Context, options core.Options) core.Result {
	input := WatchInput{
		PollInterval: options.Int("poll_interval"),
		Timeout:      options.Int("timeout"),
	}
	if workspace := options.String("workspace"); workspace != "" {
		input.Workspaces = []string{workspace}
	}
	_, out, err := s.watch(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handlePrompt reads an embedded prompt by slug.
//
//	result := c.Action("agentic.prompt").Run(ctx, core.NewOptions(
//	    core.Option{Key: "slug", Value: "coding"},
//	))
func (s *PrepSubsystem) handlePrompt(_ context.Context, options core.Options) core.Result {
	return lib.Prompt(options.String("slug"))
}

// handleTask reads an embedded task plan by slug.
//
//	result := c.Action("agentic.task").Run(ctx, core.NewOptions(
//	    core.Option{Key: "slug", Value: "bug-fix"},
//	))
func (s *PrepSubsystem) handleTask(_ context.Context, options core.Options) core.Result {
	return lib.Task(options.String("slug"))
}

// handleFlow reads an embedded flow by slug.
//
//	result := c.Action("agentic.flow").Run(ctx, core.NewOptions(
//	    core.Option{Key: "slug", Value: "go"},
//	))
func (s *PrepSubsystem) handleFlow(_ context.Context, options core.Options) core.Result {
	return lib.Flow(options.String("slug"))
}

// handlePersona reads an embedded persona by path.
//
//	result := c.Action("agentic.persona").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "code/backend-architect"},
//	))
func (s *PrepSubsystem) handlePersona(_ context.Context, options core.Options) core.Result {
	return lib.Persona(options.String("path"))
}

// --- Pipeline ---

// handleComplete runs the named completion task.
//
//	result := c.Action("agentic.complete").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/srv/.core/workspace/core/go-io/task-42"},
//	))
func (s *PrepSubsystem) handleComplete(ctx context.Context, options core.Options) core.Result {
	return s.Core().Task("agent.completion").Run(ctx, s.Core(), options)
}

// handleQA runs build+test on a completed workspace.
//
//	result := c.Action("agentic.qa").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleQA(ctx context.Context, options core.Options) core.Result {
	// Feature flag gate — skip QA if disabled
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-qa") {
		return core.Result{Value: true, OK: true}
	}
	wsDir := options.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.qa", "workspace is required", nil), OK: false}
	}
	passed := s.runQA(wsDir)
	if !passed {
		if result := ReadStatusResult(wsDir); result.OK {
			workspaceStatus, ok := workspaceStatusValue(result)
			if ok {
				workspaceStatus.Status = "failed"
				workspaceStatus.Question = "QA check failed — build or tests did not pass"
				writeStatusResult(wsDir, workspaceStatus)
			}
		}
	}
	// Emit QA result for observability (monitor picks this up)
	if s.ServiceRuntime != nil {
		result := ReadStatusResult(wsDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		repo := ""
		if ok {
			repo = workspaceStatus.Repo
		}
		s.Core().ACTION(messages.QAResult{
			Workspace: WorkspaceName(wsDir),
			Repo:      repo,
			Passed:    passed,
		})
	}
	return core.Result{Value: passed, OK: passed}
}

// handleAutoPR creates a PR for a completed workspace.
//
//	result := c.Action("agentic.auto-pr").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleAutoPR(ctx context.Context, options core.Options) core.Result {
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-pr") {
		return core.Result{OK: true}
	}
	wsDir := options.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.auto-pr", "workspace is required", nil), OK: false}
	}
	s.autoCreatePR(wsDir)

	// Emit PRCreated for observability
	if s.ServiceRuntime != nil {
		result := ReadStatusResult(wsDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if ok && workspaceStatus.PRURL != "" {
			s.Core().ACTION(messages.PRCreated{
				Repo:   workspaceStatus.Repo,
				Branch: workspaceStatus.Branch,
				PRURL:  workspaceStatus.PRURL,
				PRNum:  extractPRNumber(workspaceStatus.PRURL),
			})
		}
	}
	return core.Result{OK: true}
}

// handleVerify verifies and auto-merges a PR.
//
//	result := c.Action("agentic.verify").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleVerify(ctx context.Context, options core.Options) core.Result {
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-merge") {
		return core.Result{OK: true}
	}
	wsDir := options.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.verify", "workspace is required", nil), OK: false}
	}
	s.autoVerifyAndMerge(wsDir)

	// Emit merge/review events for observability
	if s.ServiceRuntime != nil {
		result := ReadStatusResult(wsDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if ok {
			if workspaceStatus.Status == "merged" {
				s.Core().ACTION(messages.PRMerged{
					Repo:  workspaceStatus.Repo,
					PRURL: workspaceStatus.PRURL,
					PRNum: extractPRNumber(workspaceStatus.PRURL),
				})
			} else if workspaceStatus.Question != "" {
				s.Core().ACTION(messages.PRNeedsReview{
					Repo:   workspaceStatus.Repo,
					PRURL:  workspaceStatus.PRURL,
					PRNum:  extractPRNumber(workspaceStatus.PRURL),
					Reason: workspaceStatus.Question,
				})
			}
		}
	}
	return core.Result{OK: true}
}

// handleIngest creates issues from agent findings.
//
//	result := c.Action("agentic.ingest").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleIngest(ctx context.Context, options core.Options) core.Result {
	wsDir := options.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.ingest", "workspace is required", nil), OK: false}
	}
	s.ingestFindings(wsDir)
	return core.Result{OK: true}
}

// handlePoke drains the dispatch queue.
//
//	result := c.Action("agentic.poke").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handlePoke(ctx context.Context, _ core.Options) core.Result {
	s.Poke()
	return core.Result{OK: true}
}

// handleMirror mirrors agent branches to GitHub.
//
//	result := c.Action("agentic.mirror").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	))
func (s *PrepSubsystem) handleMirror(ctx context.Context, options core.Options) core.Result {
	input := MirrorInput{
		Repo: options.String("repo"),
	}
	_, out, err := s.mirror(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// --- Forge ---

// handleIssueGet retrieves a forge issue.
//
//	result := c.Action("agentic.issue.get").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	    core.Option{Key: "number", Value: "42"},
//	))
func (s *PrepSubsystem) handleIssueGet(ctx context.Context, options core.Options) core.Result {
	return s.cmdIssueGet(options)
}

// handleIssueList lists forge issues.
//
//	result := c.Action("agentic.issue.list").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	))
func (s *PrepSubsystem) handleIssueList(ctx context.Context, options core.Options) core.Result {
	return s.cmdIssueList(options)
}

// handleIssueCreate creates a forge issue.
//
//	result := c.Action("agentic.issue.create").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	    core.Option{Key: "title", Value: "Bug report"},
//	))
func (s *PrepSubsystem) handleIssueCreate(ctx context.Context, options core.Options) core.Result {
	return s.cmdIssueCreate(options)
}

// handlePRGet retrieves a forge PR.
//
//	result := c.Action("agentic.pr.get").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	    core.Option{Key: "number", Value: "12"},
//	))
func (s *PrepSubsystem) handlePRGet(ctx context.Context, options core.Options) core.Result {
	return s.cmdPRGet(options)
}

// handlePRList lists forge PRs.
//
//	result := c.Action("agentic.pr.list").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	))
func (s *PrepSubsystem) handlePRList(ctx context.Context, options core.Options) core.Result {
	return s.cmdPRList(options)
}

// handlePRMerge merges a forge PR.
//
//	result := c.Action("agentic.pr.merge").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	    core.Option{Key: "number", Value: "12"},
//	))
func (s *PrepSubsystem) handlePRMerge(ctx context.Context, options core.Options) core.Result {
	return s.cmdPRMerge(options)
}

// --- Review ---

// handleReviewQueue runs CodeRabbit review on a workspace.
//
//	result := c.Action("agentic.review-queue").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	))
func (s *PrepSubsystem) handleReviewQueue(ctx context.Context, options core.Options) core.Result {
	input := ReviewQueueInput{
		Limit:    options.Int("limit"),
		Reviewer: options.String("reviewer"),
		DryRun:   options.Bool("dry_run"),
	}
	_, out, err := s.reviewQueue(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// --- Epic ---

// handleEpic creates an epic (multi-repo task breakdown).
//
//	result := c.Action("agentic.epic").Run(ctx, core.NewOptions(
//	    core.Option{Key: "task", Value: "Update all repos to v0.8.0"},
//	))
func (s *PrepSubsystem) handleEpic(ctx context.Context, options core.Options) core.Result {
	input := EpicInput{
		Repo:  options.String("repo"),
		Org:   options.String("org"),
		Title: options.String("title"),
		Body:  options.String("body"),
	}
	_, out, err := s.createEpic(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleWorkspaceQuery answers workspace state queries from Core QUERY calls.
//
//	result := c.QUERY(agentic.WorkspaceQuery{Name: "core/go-io/task-42"})
//	result := c.QUERY(agentic.WorkspaceQuery{Status: "blocked"})
func (s *PrepSubsystem) handleWorkspaceQuery(_ *core.Core, query core.Query) core.Result {
	workspaceQuery, ok := query.(WorkspaceQuery)
	if !ok {
		return core.Result{}
	}
	if workspaceQuery.Name != "" {
		return s.workspaces.Get(workspaceQuery.Name)
	}
	if workspaceQuery.Status != "" {
		var names []string
		s.workspaces.Each(func(name string, workspaceStatus *WorkspaceStatus) {
			if workspaceStatus.Status == workspaceQuery.Status {
				names = append(names, name)
			}
		})
		return core.Result{Value: names, OK: true}
	}
	return core.Result{Value: s.workspaces, OK: true}
}
