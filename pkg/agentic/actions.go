// SPDX-License-Identifier: EUPL-1.2

//	c.Action("agentic.dispatch").Run(ctx, options)
//	c.Actions()

package agentic

import (
	"context"

	"dappco.re/go/agent/pkg/lib"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// result := c.Action("agentic.dispatch").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "repo", Value: "go-io"},
//	core.Option{Key: "task", Value: "Fix tests"},
//
// ))
func (s *PrepSubsystem) handleDispatch(ctx context.Context, options core.Options) core.Result {
	input := dispatchInputFromOptions(options)
	_, out, err := s.dispatch(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// result := c.Action("agentic.prep").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "repo", Value: "go-io"},
//	core.Option{Key: "issue", Value: 42},
//
// ))
func (s *PrepSubsystem) handlePrep(ctx context.Context, options core.Options) core.Result {
	input := prepInputFromOptions(options)
	_, out, err := s.prepWorkspace(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// result := c.Action("agentic.status").Run(ctx, core.NewOptions())
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

// result := c.Action("agentic.resume").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//
// ))
func (s *PrepSubsystem) handleResume(ctx context.Context, options core.Options) core.Result {
	input := resumeInputFromOptions(options)
	_, out, err := s.resume(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// result := c.Action("agentic.scan").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleScan(ctx context.Context, options core.Options) core.Result {
	input := scanInputFromOptions(options)
	_, out, err := s.scan(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// result := c.Action("agentic.watch").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//
// ))
func (s *PrepSubsystem) handleWatch(ctx context.Context, options core.Options) core.Result {
	input := watchInputFromOptions(options)
	_, out, err := s.watch(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// result := c.Action("agentic.prompt").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "slug", Value: "coding"},
//
// ))
func (s *PrepSubsystem) handlePrompt(_ context.Context, options core.Options) core.Result {
	return lib.Prompt(options.String("slug"))
}

// result := c.Action("agentic.task").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "slug", Value: "bug-fix"},
//
// ))
func (s *PrepSubsystem) handleTask(_ context.Context, options core.Options) core.Result {
	return lib.Task(options.String("slug"))
}

// result := c.Action("agentic.flow").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "slug", Value: "go"},
//
// ))
func (s *PrepSubsystem) handleFlow(_ context.Context, options core.Options) core.Result {
	return lib.Flow(options.String("slug"))
}

// result := c.Action("agentic.persona").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "path", Value: "code/backend-architect"},
//
// ))
func (s *PrepSubsystem) handlePersona(_ context.Context, options core.Options) core.Result {
	return lib.Persona(options.String("path"))
}

// result := c.Action("agentic.complete").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "/srv/.core/workspace/core/go-io/task-42"},
//
// ))
func (s *PrepSubsystem) handleComplete(ctx context.Context, options core.Options) core.Result {
	return s.Core().Task("agent.completion").Run(ctx, s.Core(), options)
}

// result := c.Action("agentic.qa").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "/path/to/workspace"},
//
// ))
func (s *PrepSubsystem) handleQA(ctx context.Context, options core.Options) core.Result {
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-qa") {
		return core.Result{Value: true, OK: true}
	}
	workspaceDir := options.String("workspace")
	if workspaceDir == "" {
		return core.Result{Value: core.E("agentic.qa", "workspace is required", nil), OK: false}
	}
	passed := s.runQA(workspaceDir)
	if !passed {
		if result := ReadStatusResult(workspaceDir); result.OK {
			workspaceStatus, ok := workspaceStatusValue(result)
			if ok {
				workspaceStatus.Status = "failed"
				workspaceStatus.Question = "QA check failed — build or tests did not pass"
				writeStatusResult(workspaceDir, workspaceStatus)
			}
		}
	}
	if s.ServiceRuntime != nil {
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		repo := ""
		if ok {
			repo = workspaceStatus.Repo
		}
		s.Core().ACTION(messages.QAResult{
			Workspace: WorkspaceName(workspaceDir),
			Repo:      repo,
			Passed:    passed,
		})
	}
	return core.Result{Value: passed, OK: passed}
}

// result := c.Action("agentic.auto-pr").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "/path/to/workspace"},
//
// ))
func (s *PrepSubsystem) handleAutoPR(ctx context.Context, options core.Options) core.Result {
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-pr") {
		return core.Result{OK: true}
	}
	workspaceDir := options.String("workspace")
	if workspaceDir == "" {
		return core.Result{Value: core.E("agentic.auto-pr", "workspace is required", nil), OK: false}
	}
	s.autoCreatePR(workspaceDir)

	if s.ServiceRuntime != nil {
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if ok && workspaceStatus.PRURL != "" {
			s.Core().ACTION(messages.PRCreated{
				Repo:   workspaceStatus.Repo,
				Branch: workspaceStatus.Branch,
				PRURL:  workspaceStatus.PRURL,
				PRNum:  extractPullRequestNumber(workspaceStatus.PRURL),
			})
		}
	}
	return core.Result{OK: true}
}

// result := c.Action("agentic.verify").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "/path/to/workspace"},
//
// ))
func (s *PrepSubsystem) handleVerify(ctx context.Context, options core.Options) core.Result {
	if s.ServiceRuntime != nil && !s.Config().Enabled("auto-merge") {
		return core.Result{OK: true}
	}
	workspaceDir := options.String("workspace")
	if workspaceDir == "" {
		return core.Result{Value: core.E("agentic.verify", "workspace is required", nil), OK: false}
	}
	s.autoVerifyAndMerge(workspaceDir)

	if s.ServiceRuntime != nil {
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if ok {
			if workspaceStatus.Status == "merged" {
				s.Core().ACTION(messages.PRMerged{
					Repo:  workspaceStatus.Repo,
					PRURL: workspaceStatus.PRURL,
					PRNum: extractPullRequestNumber(workspaceStatus.PRURL),
				})
			} else if workspaceStatus.Question != "" {
				s.Core().ACTION(messages.PRNeedsReview{
					Repo:   workspaceStatus.Repo,
					PRURL:  workspaceStatus.PRURL,
					PRNum:  extractPullRequestNumber(workspaceStatus.PRURL),
					Reason: workspaceStatus.Question,
				})
			}
		}
	}
	return core.Result{OK: true}
}

// result := c.Action("agentic.ingest").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "/path/to/workspace"},
//
// ))
func (s *PrepSubsystem) handleIngest(ctx context.Context, options core.Options) core.Result {
	workspaceDir := options.String("workspace")
	if workspaceDir == "" {
		return core.Result{Value: core.E("agentic.ingest", "workspace is required", nil), OK: false}
	}
	s.ingestFindings(workspaceDir)
	return core.Result{OK: true}
}

// result := c.Action("agentic.poke").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handlePoke(ctx context.Context, _ core.Options) core.Result {
	if s.ServiceRuntime != nil && s.Core().Action("runner.poke").Exists() {
		return s.Core().Action("runner.poke").Run(ctx, core.NewOptions())
	}
	s.Poke()
	return core.Result{OK: true}
}

// result := c.Action("agentic.mirror").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "repo", Value: "go-io"},
//
// ))
func (s *PrepSubsystem) handleMirror(ctx context.Context, options core.Options) core.Result {
	input := mirrorInputFromOptions(options)
	_, out, err := s.mirror(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// result := c.Action("agentic.issue.get").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "repo", Value: "go-io"},
//	core.Option{Key: "number", Value: "42"},
//
// ))
func (s *PrepSubsystem) handleIssueGet(ctx context.Context, options core.Options) core.Result {
	return s.cmdIssueGet(normaliseForgeActionOptions(options))
}

// result := c.Action("agentic.issue.list").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "_arg", Value: "go-io"},
//
// ))
func (s *PrepSubsystem) handleIssueList(ctx context.Context, options core.Options) core.Result {
	return s.cmdIssueList(normaliseForgeActionOptions(options))
}

// result := c.Action("agentic.issue.create").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "_arg", Value: "go-io"},
//	core.Option{Key: "title", Value: "Bug report"},
//
// ))
func (s *PrepSubsystem) handleIssueCreate(ctx context.Context, options core.Options) core.Result {
	return s.cmdIssueCreate(normaliseForgeActionOptions(options))
}

// result := c.Action("agentic.pr.get").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "_arg", Value: "go-io"},
//	core.Option{Key: "number", Value: "12"},
//
// ))
func (s *PrepSubsystem) handlePRGet(ctx context.Context, options core.Options) core.Result {
	return s.cmdPRGet(normaliseForgeActionOptions(options))
}

// result := c.Action("agentic.pr.list").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "_arg", Value: "go-io"},
//
// ))
func (s *PrepSubsystem) handlePRList(ctx context.Context, options core.Options) core.Result {
	return s.cmdPRList(normaliseForgeActionOptions(options))
}

// result := c.Action("agentic.pr.merge").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "_arg", Value: "go-io"},
//	core.Option{Key: "number", Value: "12"},
//
// ))
func (s *PrepSubsystem) handlePRMerge(ctx context.Context, options core.Options) core.Result {
	return s.cmdPRMerge(normaliseForgeActionOptions(options))
}

// result := c.Action("agentic.review-queue").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//
// ))
func (s *PrepSubsystem) handleReviewQueue(ctx context.Context, options core.Options) core.Result {
	input := reviewQueueInputFromOptions(options)
	_, out, err := s.reviewQueue(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

// result := c.Action("agentic.epic").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "task", Value: "Update all repos to v0.8.0"},
//
// ))
func (s *PrepSubsystem) handleEpic(ctx context.Context, options core.Options) core.Result {
	input := epicInputFromOptions(options)
	_, out, err := s.createEpic(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: out, OK: true}
}

func dispatchInputFromOptions(options core.Options) DispatchInput {
	return DispatchInput{
		Repo:         optionStringValue(options, "repo"),
		Org:          optionStringValue(options, "org"),
		Task:         optionStringValue(options, "task"),
		Agent:        optionStringValue(options, "agent"),
		Template:     optionStringValue(options, "template"),
		PlanTemplate: optionStringValue(options, "plan_template", "plan-template"),
		Variables:    optionStringMapValue(options, "variables"),
		Persona:      optionStringValue(options, "persona"),
		Issue:        optionIntValue(options, "issue"),
		PR:           optionIntValue(options, "pr"),
		Branch:       optionStringValue(options, "branch"),
		Tag:          optionStringValue(options, "tag"),
		DryRun:       optionBoolValue(options, "dry_run", "dry-run"),
	}
}

func prepInputFromOptions(options core.Options) PrepInput {
	return PrepInput{
		Repo:         optionStringValue(options, "repo"),
		Org:          optionStringValue(options, "org"),
		Task:         optionStringValue(options, "task"),
		Agent:        optionStringValue(options, "agent"),
		Issue:        optionIntValue(options, "issue"),
		PR:           optionIntValue(options, "pr"),
		Branch:       optionStringValue(options, "branch"),
		Tag:          optionStringValue(options, "tag"),
		Template:     optionStringValue(options, "template"),
		PlanTemplate: optionStringValue(options, "plan_template", "plan-template"),
		Variables:    optionStringMapValue(options, "variables"),
		Persona:      optionStringValue(options, "persona"),
		DryRun:       optionBoolValue(options, "dry_run", "dry-run"),
	}
}

func resumeInputFromOptions(options core.Options) ResumeInput {
	return ResumeInput{
		Workspace: optionStringValue(options, "workspace"),
		Answer:    optionStringValue(options, "answer"),
		Agent:     optionStringValue(options, "agent"),
		DryRun:    optionBoolValue(options, "dry_run", "dry-run"),
	}
}

func scanInputFromOptions(options core.Options) ScanInput {
	return ScanInput{
		Org:    optionStringValue(options, "org"),
		Labels: optionStringSliceValue(options, "labels"),
		Limit:  optionIntValue(options, "limit"),
	}
}

func watchInputFromOptions(options core.Options) WatchInput {
	workspaces := optionStringSliceValue(options, "workspaces")
	if len(workspaces) == 0 {
		if workspace := optionStringValue(options, "workspace"); workspace != "" {
			workspaces = []string{workspace}
		}
	}
	return WatchInput{
		Workspaces:   workspaces,
		PollInterval: optionIntValue(options, "poll_interval", "poll-interval"),
		Timeout:      optionIntValue(options, "timeout"),
	}
}

func mirrorInputFromOptions(options core.Options) MirrorInput {
	return MirrorInput{
		Repo:     optionStringValue(options, "repo"),
		DryRun:   optionBoolValue(options, "dry_run", "dry-run"),
		MaxFiles: optionIntValue(options, "max_files", "max-files"),
	}
}

func reviewQueueInputFromOptions(options core.Options) ReviewQueueInput {
	return ReviewQueueInput{
		Limit:     optionIntValue(options, "limit"),
		Reviewer:  optionStringValue(options, "reviewer"),
		DryRun:    optionBoolValue(options, "dry_run", "dry-run"),
		LocalOnly: optionBoolValue(options, "local_only", "local-only"),
	}
}

func epicInputFromOptions(options core.Options) EpicInput {
	return EpicInput{
		Repo:     optionStringValue(options, "repo"),
		Org:      optionStringValue(options, "org"),
		Title:    optionStringValue(options, "title"),
		Body:     optionStringValue(options, "body"),
		Tasks:    optionStringSliceValue(options, "tasks"),
		Labels:   optionStringSliceValue(options, "labels"),
		Dispatch: optionBoolValue(options, "dispatch"),
		Agent:    optionStringValue(options, "agent"),
		Template: optionStringValue(options, "template"),
	}
}

func normaliseForgeActionOptions(options core.Options) core.Options {
	normalised := core.NewOptions(options.Items()...)
	if normalised.String("_arg") == "" {
		if repo := optionStringValue(options, "repo"); repo != "" {
			normalised.Set("_arg", repo)
		}
	}
	if number := optionStringValue(options, "number"); number != "" {
		normalised.Set("number", number)
	}
	return normalised
}

func optionStringValue(options core.Options, keys ...string) string {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		if value := stringValue(result.Value); value != "" {
			return value
		}
	}
	return ""
}

func optionIntValue(options core.Options, keys ...string) int {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		switch value := result.Value.(type) {
		case int:
			return value
		case int64:
			return int(value)
		case float64:
			return int(value)
		case string:
			parsed := parseInt(value)
			if parsed != 0 || core.Trim(value) == "0" {
				return parsed
			}
			return parseIntString(value)
		}
	}
	return 0
}

func optionBoolValue(options core.Options, keys ...string) bool {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		switch value := result.Value.(type) {
		case bool:
			return value
		case string:
			switch core.Lower(core.Trim(value)) {
			case "1", "true", "yes", "on":
				return true
			}
		}
	}
	return false
}

func optionStringSliceValue(options core.Options, keys ...string) []string {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		values := stringSliceValue(result.Value)
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func optionStringMapValue(options core.Options, keys ...string) map[string]string {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		values := stringMapValue(result.Value)
		if len(values) > 0 {
			return values
		}
	}
	return nil
}

func optionAnyValue(options core.Options, keys ...string) any {
	for _, key := range keys {
		result := options.Get(key)
		if !result.OK {
			continue
		}
		return normaliseOptionValue(result.Value)
	}
	return nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return core.Sprint(typed)
	case int64:
		return core.Sprint(typed)
	case float64:
		return core.Sprint(int(typed))
	case bool:
		return core.Sprint(typed)
	}
	return ""
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return cleanStrings(typed)
	case []any:
		var values []string
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				values = append(values, text)
			}
		}
		return cleanStrings(values)
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "[") {
			var values []string
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return cleanStrings(values)
			}
			var generic []any
			if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
				return stringSliceValue(generic)
			}
		}
		return cleanStrings(core.Split(trimmed, ","))
	default:
		if text := stringValue(value); text != "" {
			return []string{text}
		}
	}
	return nil
}

func normaliseOptionValue(value any) any {
	switch typed := value.(type) {
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return ""
		}
		if core.HasPrefix(trimmed, "{") {
			var values map[string]any
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return values
			}
		}
		if core.HasPrefix(trimmed, "[") {
			var values []any
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return values
			}
		}
		switch core.Lower(trimmed) {
		case "true":
			return true
		case "false":
			return false
		}
		if parsed := parseInt(trimmed); parsed != 0 || trimmed == "0" {
			return parsed
		}
		return typed
	default:
		return value
	}
}

func stringMapValue(value any) map[string]string {
	switch typed := value.(type) {
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, val := range typed {
			if text := core.Trim(val); text != "" {
				out[key] = text
			}
		}
		return out
	case map[string]any:
		out := make(map[string]string, len(typed))
		for key, val := range typed {
			if text := stringValue(val); text != "" {
				out[key] = text
			}
		}
		return out
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "{") {
			var values map[string]string
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return stringMapValue(values)
			}
			var generic map[string]any
			if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
				return stringMapValue(generic)
			}
		}
	}
	return nil
}

func cleanStrings(values []string) []string {
	var cleaned []string
	for _, value := range values {
		trimmed := core.Trim(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

// result := c.QUERY(agentic.WorkspaceQuery{Name: "core/go-io/task-42"})
// result := c.QUERY(agentic.WorkspaceQuery{Status: "blocked"})
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
