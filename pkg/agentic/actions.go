// SPDX-License-Identifier: EUPL-1.2

// Named Action handlers for the agentic service.
// Each handler adapts (ctx, Options) → Result to call the existing MCP tool method.
// Registered during OnStartup — the Action registry IS the capability map.
//
//	c.Action("agentic.dispatch").Run(ctx, opts)
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
//	r := c.Action("agentic.dispatch").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	    core.Option{Key: "task", Value: "Fix tests"},
//	))
func (s *PrepSubsystem) handleDispatch(ctx context.Context, opts core.Options) core.Result {
	input := DispatchInput{
		Repo:  opts.String("repo"),
		Task:  opts.String("task"),
		Agent: opts.String("agent"),
		Issue: opts.Int("issue"),
	}
	_, out, err := s.dispatch(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handlePrep prepares a workspace without dispatching an agent.
//
//	r := c.Action("agentic.prep").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	    core.Option{Key: "issue", Value: 42},
//	))
func (s *PrepSubsystem) handlePrep(ctx context.Context, opts core.Options) core.Result {
	input := PrepInput{
		Repo:  opts.String("repo"),
		Org:   opts.String("org"),
		Issue: opts.Int("issue"),
	}
	_, out, err := s.prepWorkspace(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleStatus lists workspace statuses.
//
//	r := c.Action("agentic.status").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleStatus(ctx context.Context, opts core.Options) core.Result {
	input := StatusInput{
		Workspace: opts.String("workspace"),
		Limit:     opts.Int("limit"),
		Status:    opts.String("status"),
	}
	_, out, err := s.status(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleResume resumes a blocked workspace.
//
//	r := c.Action("agentic.resume").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	))
func (s *PrepSubsystem) handleResume(ctx context.Context, opts core.Options) core.Result {
	input := ResumeInput{
		Workspace: opts.String("workspace"),
		Answer:    opts.String("answer"),
	}
	_, out, err := s.resume(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleScan scans forge repos for actionable issues.
//
//	r := c.Action("agentic.scan").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleScan(ctx context.Context, opts core.Options) core.Result {
	input := ScanInput{
		Org:   opts.String("org"),
		Limit: opts.Int("limit"),
	}
	_, out, err := s.scan(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleWatch watches a workspace for completion.
//
//	r := c.Action("agentic.watch").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	))
func (s *PrepSubsystem) handleWatch(ctx context.Context, opts core.Options) core.Result {
	input := WatchInput{
		PollInterval: opts.Int("poll_interval"),
		Timeout:      opts.Int("timeout"),
	}
	if workspace := opts.String("workspace"); workspace != "" {
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
//	r := c.Action("agentic.prompt").Run(ctx, core.NewOptions(
//	    core.Option{Key: "slug", Value: "coding"},
//	))
func (s *PrepSubsystem) handlePrompt(_ context.Context, opts core.Options) core.Result {
	return lib.Prompt(opts.String("slug"))
}

// handleTask reads an embedded task plan by slug.
//
//	r := c.Action("agentic.task").Run(ctx, core.NewOptions(
//	    core.Option{Key: "slug", Value: "bug-fix"},
//	))
func (s *PrepSubsystem) handleTask(_ context.Context, opts core.Options) core.Result {
	return lib.Task(opts.String("slug"))
}

// handleFlow reads an embedded flow by slug.
//
//	r := c.Action("agentic.flow").Run(ctx, core.NewOptions(
//	    core.Option{Key: "slug", Value: "go"},
//	))
func (s *PrepSubsystem) handleFlow(_ context.Context, opts core.Options) core.Result {
	return lib.Flow(opts.String("slug"))
}

// handlePersona reads an embedded persona by path.
//
//	r := c.Action("agentic.persona").Run(ctx, core.NewOptions(
//	    core.Option{Key: "path", Value: "code/backend-architect"},
//	))
func (s *PrepSubsystem) handlePersona(_ context.Context, opts core.Options) core.Result {
	return lib.Persona(opts.String("path"))
}

// --- Pipeline ---

// handleComplete runs the named completion task.
//
//	r := c.Action("agentic.complete").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/srv/.core/workspace/core/go-io/task-42"},
//	))
func (s *PrepSubsystem) handleComplete(ctx context.Context, opts core.Options) core.Result {
	return s.Core().Task("agent.completion").Run(ctx, s.Core(), opts)
}

// handleQA runs build+test on a completed workspace.
//
//	r := c.Action("agentic.qa").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleQA(ctx context.Context, opts core.Options) core.Result {
	// Feature flag gate — skip QA if disabled
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-qa") {
		return core.Result{Value: true, OK: true}
	}
	wsDir := opts.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.qa", "workspace is required", nil), OK: false}
	}
	passed := s.runQA(wsDir)
	if !passed {
		if st, err := ReadStatus(wsDir); err == nil {
			st.Status = "failed"
			st.Question = "QA check failed — build or tests did not pass"
			writeStatusResult(wsDir, st)
		}
	}
	// Emit QA result for observability (monitor picks this up)
	if s.ServiceRuntime != nil {
		st, _ := ReadStatus(wsDir)
		repo := ""
		if st != nil {
			repo = st.Repo
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
//	r := c.Action("agentic.auto-pr").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleAutoPR(ctx context.Context, opts core.Options) core.Result {
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-pr") {
		return core.Result{OK: true}
	}
	wsDir := opts.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.auto-pr", "workspace is required", nil), OK: false}
	}
	s.autoCreatePR(wsDir)

	// Emit PRCreated for observability
	if s.ServiceRuntime != nil {
		if st, err := ReadStatus(wsDir); err == nil && st.PRURL != "" {
			s.Core().ACTION(messages.PRCreated{
				Repo:   st.Repo,
				Branch: st.Branch,
				PRURL:  st.PRURL,
				PRNum:  extractPRNumber(st.PRURL),
			})
		}
	}
	return core.Result{OK: true}
}

// handleVerify verifies and auto-merges a PR.
//
//	r := c.Action("agentic.verify").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleVerify(ctx context.Context, opts core.Options) core.Result {
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-merge") {
		return core.Result{OK: true}
	}
	wsDir := opts.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.verify", "workspace is required", nil), OK: false}
	}
	s.autoVerifyAndMerge(wsDir)

	// Emit merge/review events for observability
	if s.ServiceRuntime != nil {
		if st, err := ReadStatus(wsDir); err == nil {
			if st.Status == "merged" {
				s.Core().ACTION(messages.PRMerged{
					Repo:  st.Repo,
					PRURL: st.PRURL,
					PRNum: extractPRNumber(st.PRURL),
				})
			} else if st.Question != "" {
				s.Core().ACTION(messages.PRNeedsReview{
					Repo:   st.Repo,
					PRURL:  st.PRURL,
					PRNum:  extractPRNumber(st.PRURL),
					Reason: st.Question,
				})
			}
		}
	}
	return core.Result{OK: true}
}

// handleIngest creates issues from agent findings.
//
//	r := c.Action("agentic.ingest").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "/path/to/workspace"},
//	))
func (s *PrepSubsystem) handleIngest(ctx context.Context, opts core.Options) core.Result {
	wsDir := opts.String("workspace")
	if wsDir == "" {
		return core.Result{Value: core.E("agentic.ingest", "workspace is required", nil), OK: false}
	}
	s.ingestFindings(wsDir)
	return core.Result{OK: true}
}

// handlePoke drains the dispatch queue.
//
//	r := c.Action("agentic.poke").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handlePoke(ctx context.Context, opts core.Options) core.Result {
	s.Poke()
	return core.Result{OK: true}
}

// handleMirror mirrors agent branches to GitHub.
//
//	r := c.Action("agentic.mirror").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	))
func (s *PrepSubsystem) handleMirror(ctx context.Context, opts core.Options) core.Result {
	input := MirrorInput{
		Repo: opts.String("repo"),
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
//	r := c.Action("agentic.issue.get").Run(ctx, core.NewOptions(
//	    core.Option{Key: "repo", Value: "go-io"},
//	    core.Option{Key: "number", Value: "42"},
//	))
func (s *PrepSubsystem) handleIssueGet(ctx context.Context, opts core.Options) core.Result {
	return s.cmdIssueGet(opts)
}

// handleIssueList lists forge issues.
//
//	r := c.Action("agentic.issue.list").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	))
func (s *PrepSubsystem) handleIssueList(ctx context.Context, opts core.Options) core.Result {
	return s.cmdIssueList(opts)
}

// handleIssueCreate creates a forge issue.
//
//	r := c.Action("agentic.issue.create").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	    core.Option{Key: "title", Value: "Bug report"},
//	))
func (s *PrepSubsystem) handleIssueCreate(ctx context.Context, opts core.Options) core.Result {
	return s.cmdIssueCreate(opts)
}

// handlePRGet retrieves a forge PR.
//
//	r := c.Action("agentic.pr.get").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	    core.Option{Key: "number", Value: "12"},
//	))
func (s *PrepSubsystem) handlePRGet(ctx context.Context, opts core.Options) core.Result {
	return s.cmdPRGet(opts)
}

// handlePRList lists forge PRs.
//
//	r := c.Action("agentic.pr.list").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	))
func (s *PrepSubsystem) handlePRList(ctx context.Context, opts core.Options) core.Result {
	return s.cmdPRList(opts)
}

// handlePRMerge merges a forge PR.
//
//	r := c.Action("agentic.pr.merge").Run(ctx, core.NewOptions(
//	    core.Option{Key: "_arg", Value: "go-io"},
//	    core.Option{Key: "number", Value: "12"},
//	))
func (s *PrepSubsystem) handlePRMerge(ctx context.Context, opts core.Options) core.Result {
	return s.cmdPRMerge(opts)
}

// --- Review ---

// handleReviewQueue runs CodeRabbit review on a workspace.
//
//	r := c.Action("agentic.review-queue").Run(ctx, core.NewOptions(
//	    core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	))
func (s *PrepSubsystem) handleReviewQueue(ctx context.Context, opts core.Options) core.Result {
	input := ReviewQueueInput{
		Limit:    opts.Int("limit"),
		Reviewer: opts.String("reviewer"),
		DryRun:   opts.Bool("dry_run"),
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
//	r := c.Action("agentic.epic").Run(ctx, core.NewOptions(
//	    core.Option{Key: "task", Value: "Update all repos to v0.8.0"},
//	))
func (s *PrepSubsystem) handleEpic(ctx context.Context, opts core.Options) core.Result {
	input := EpicInput{
		Repo:  opts.String("repo"),
		Org:   opts.String("org"),
		Title: opts.String("title"),
		Body:  opts.String("body"),
	}
	_, out, err := s.createEpic(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// handleWorkspaceQuery answers workspace state queries from Core QUERY calls.
//
//	r := c.QUERY(agentic.WorkspaceQuery{Name: "core/go-io/task-42"})
//	r := c.QUERY(agentic.WorkspaceQuery{Status: "blocked"})
func (s *PrepSubsystem) handleWorkspaceQuery(_ *core.Core, q core.Query) core.Result {
	wq, ok := q.(WorkspaceQuery)
	if !ok {
		return core.Result{}
	}
	if wq.Name != "" {
		return s.workspaces.Get(wq.Name)
	}
	if wq.Status != "" {
		var names []string
		s.workspaces.Each(func(name string, st *WorkspaceStatus) {
			if st.Status == wq.Status {
				names = append(names, name)
			}
		})
		return core.Result{Value: names, OK: true}
	}
	return core.Result{Value: s.workspaces, OK: true}
}
