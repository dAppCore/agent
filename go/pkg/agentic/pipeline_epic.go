// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sort"

	core "dappco.re/go"
)

type PipelineEpicCreateInput struct {
	Org        string             `json:"org,omitempty"`
	Repo       string             `json:"repo"`
	Theme      string             `json:"theme,omitempty"`
	DryRun     bool               `json:"dry_run,omitempty"`
	Candidates []PipelineIssueRef `json:"candidates,omitempty"`
}

type PipelineEpicCreateOutput struct {
	Success    bool               `json:"success"`
	Org        string             `json:"org,omitempty"`
	Repo       string             `json:"repo"`
	Candidates []PipelineIssueRef `json:"candidates,omitempty"`
	Epics      []PipelineEpicMeta `json:"epics,omitempty"`
	Existing   []PipelineEpicMeta `json:"existing,omitempty"`
}

type PipelineEpicRunInput struct {
	Org        string            `json:"org,omitempty"`
	Repo       string            `json:"repo"`
	EpicNumber int               `json:"epic_number"`
	Agent      string            `json:"agent,omitempty"`
	Template   string            `json:"template,omitempty"`
	DryRun     bool              `json:"dry_run,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Epic       *PipelineEpicMeta `json:"epic,omitempty"`
}

type PipelineEpicRunOutput struct {
	Success    bool               `json:"success"`
	Org        string             `json:"org,omitempty"`
	Repo       string             `json:"repo"`
	EpicNumber int                `json:"epic_number"`
	Branch     string             `json:"branch,omitempty"`
	Dispatched []PipelineIssueRef `json:"dispatched,omitempty"`
	Skipped    []PipelineIssueRef `json:"skipped,omitempty"`
}

type PipelineEpicStatusOutput struct {
	Success bool             `json:"success"`
	Org     string           `json:"org,omitempty"`
	Repo    string           `json:"repo"`
	Epic    PipelineEpicMeta `json:"epic"`
}

type PipelineEpicSyncOutput struct {
	Success    bool   `json:"success"`
	Org        string `json:"org,omitempty"`
	Repo       string `json:"repo"`
	EpicNumber int    `json:"epic_number"`
	Checked    int    `json:"checked"`
	Total      int    `json:"total"`
	Updated    bool   `json:"updated"`
}

func (s *PrepSubsystem) cmdPipelineEpicCreate(options core.Options) core.Result {
	ctx := s.commandContext()
	repo := pipelineRepoValue(options)
	if repo == "" {
		core.Print(nil, "usage: core-agent pipeline/epic/create <repo> [--org=core] [--theme=security] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineEpicCreate", "repo is required", nil), OK: false}
	}

	output, err := pipelineEpicCreate(s, ctx, PipelineEpicCreateInput{
		Org:    pipelineOrgValue(options),
		Repo:   repo,
		Theme:  optionStringValue(options, "theme"),
		DryRun: optionBoolValue(options, "dry_run", "dry-run"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "repo:       %s/%s", output.Org, output.Repo)
	core.Print(nil, "candidates: %d", len(output.Candidates))
	core.Print(nil, "epics:      %d", len(output.Epics))
	for _, epic := range output.Epics {
		core.Print(nil, "  epic:   #%d %s", epic.Number, epic.Title)
		if epic.Branch != "" {
			core.Print(nil, "  branch: %s", epic.Branch)
		}
		core.Print(nil, "  child:  %d", len(epic.Children))
	}
	if len(output.Existing) > 0 {
		core.Print(nil, "existing:   %d", len(output.Existing))
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPipelineEpicRun(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/epic/run <epic-number> --repo=<repo> [--org=core] [--agent=codex] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineEpicRun", "repo and epic number are required", nil), OK: false}
	}

	output, err := pipelineEpicRun(s, ctx, PipelineEpicRunInput{
		Org:        org,
		Repo:       repo,
		EpicNumber: number,
		Agent:      optionStringValue(options, "agent"),
		Template:   optionStringValue(options, "template"),
		DryRun:     optionBoolValue(options, "dry_run", "dry-run"),
		Limit:      optionIntValue(options, "limit"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "epic:       #%d", output.EpicNumber)
	core.Print(nil, "repo:       %s/%s", output.Org, output.Repo)
	core.Print(nil, "dispatched: %d", len(output.Dispatched))
	core.Print(nil, "skipped:    %d", len(output.Skipped))
	if output.Branch != "" {
		core.Print(nil, "branch:     %s", output.Branch)
	}
	for _, issue := range output.Dispatched {
		core.Print(nil, "  issue: #%d %s", issue.Number, issue.Title)
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPipelineEpicStatus(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/epic/status <epic-number> --repo=<repo> [--org=core]")
		return core.Result{Value: core.E("agentic.cmdPipelineEpicStatus", "repo and epic number are required", nil), OK: false}
	}

	reader := newPipelineForgeMetaReader(s, org)
	epic, err := reader.GetEpicMeta(ctx, repo, number)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output := PipelineEpicStatusOutput{
		Success: true,
		Org:     org,
		Repo:    repo,
		Epic:    epic,
	}

	core.Print(nil, "epic:    #%d %s", epic.Number, epic.Title)
	core.Print(nil, "state:   %s", epic.State)
	core.Print(nil, "branch:  %s", epic.Branch)
	core.Print(nil, "child:   %d", len(epic.Children))
	for _, child := range epic.Children {
		box := "[ ]"
		if child.Checked {
			box = "[x]"
		}
		core.Print(nil, "  %s #%d %-8s %s", box, child.Number, child.State, child.Title)
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPipelineEpicSync(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/epic/sync <epic-number> --repo=<repo> [--org=core] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineEpicSync", "repo and epic number are required", nil), OK: false}
	}

	output, err := pipelineEpicSync(s, ctx, org, repo, number, optionBoolValue(options, "dry_run", "dry-run"))
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "epic:    #%d", output.EpicNumber)
	core.Print(nil, "checked: %d/%d", output.Checked, output.Total)
	core.Print(nil, "updated: %v", output.Updated)
	return core.Result{Value: output, OK: true}
}

var pipelineEpicCreate = func(s *PrepSubsystem, ctx context.Context, input PipelineEpicCreateInput) (PipelineEpicCreateOutput, error) {
	if input.Repo == "" {
		return PipelineEpicCreateOutput{}, core.E("pipelineEpicCreate", "repo is required", nil)
	}
	if s.forgeToken == "" {
		return PipelineEpicCreateOutput{}, core.E("pipelineEpicCreate", "no Forge token configured", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}

	candidates := input.Candidates
	if len(candidates) == 0 {
		issues, err := pipelineListIssues(s, ctx, input.Org, input.Repo, "open")
		if err != nil {
			return PipelineEpicCreateOutput{}, err
		}
		for _, issue := range issues {
			if !pipelineIssueIsImplementationCandidate(issue) {
				continue
			}
			ref := pipelineIssueRefFromRecord(issue)
			if input.Theme != "" && pipelineEpicTheme(ref) != input.Theme {
				continue
			}
			candidates = append(candidates, ref)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Number < candidates[j].Number
	})

	output := PipelineEpicCreateOutput{
		Success:    true,
		Org:        input.Org,
		Repo:       input.Repo,
		Candidates: candidates,
	}

	if len(candidates) < 3 {
		return output, nil
	}

	groups := map[string][]PipelineIssueRef{}
	for _, candidate := range candidates {
		theme := pipelineEpicTheme(candidate)
		if input.Theme != "" {
			theme = input.Theme
		}
		groups[theme] = append(groups[theme], candidate)
	}

	existingIssues, err := pipelineListIssues(s, ctx, input.Org, input.Repo, "open")
	if err != nil {
		return PipelineEpicCreateOutput{}, err
	}
	existingByTitle := map[string]pipelineIssueRecord{}
	for _, issue := range existingIssues {
		if pipelineIssueIsEpic(issue) {
			existingByTitle[issue.Title] = issue
		}
	}

	groupNames := make([]string, 0, len(groups))
	for theme, group := range groups {
		if len(group) >= 3 {
			groupNames = append(groupNames, theme)
		}
	}
	sort.Strings(groupNames)

	reader := newPipelineForgeMetaReader(s, input.Org)
	for _, theme := range groupNames {
		group := groups[theme]
		title := pipelineEpicTitle(input.Repo, theme)
		if existing, ok := existingByTitle[title]; ok {
			meta, metaErr := reader.GetEpicMeta(ctx, input.Repo, existing.Number)
			if metaErr != nil {
				return PipelineEpicCreateOutput{}, metaErr
			}
			output.Existing = append(output.Existing, meta)
			continue
		}

		labels := []string{"agentic", "epic", theme}
		body := pipelineEpicBody(title, "", theme, group)

		meta := PipelineEpicMeta{
			Title:  title,
			State:  "open",
			Branch: "",
		}
		for _, child := range group {
			meta.Children = append(meta.Children, PipelineEpicChildMeta{
				Number:  child.Number,
				Title:   child.Title,
				Checked: false,
				State:   "open",
				URL:     child.URL,
			})
		}

		if input.DryRun {
			meta.Branch = core.Sprintf("epic/dry-run-%s", pipelineSlug(theme))
			output.Epics = append(output.Epics, meta)
			continue
		}

		labelIDs := s.resolveLabelIDs(ctx, input.Org, input.Repo, labels)
		created, createErr := createIssue(s, ctx, input.Org, input.Repo, title, body, labelIDs)
		if createErr != nil {
			return PipelineEpicCreateOutput{}, core.E("pipelineEpicCreate", "failed to create epic issue", createErr)
		}

		branch, branchErr := pipelineCreateEpicBranch(s, ctx, input.Org, input.Repo, created.Number, theme)
		if branchErr != nil {
			return PipelineEpicCreateOutput{}, branchErr
		}

		patchedBody := pipelineEpicBody(title, branch, theme, group)
		if patchErr := pipelinePatchIssue(s, ctx, input.Org, input.Repo, created.Number, map[string]any{"body": patchedBody}); patchErr != nil {
			return PipelineEpicCreateOutput{}, patchErr
		}

		for _, child := range group {
			comment := core.Sprintf("Parent: #%d\nEpic branch: `%s`", created.Number, branch)
			s.commentOnIssue(ctx, input.Org, input.Repo, child.Number, comment)
		}

		meta.Number = created.Number
		meta.URL = created.URL
		meta.Branch = branch
		output.Epics = append(output.Epics, meta)
	}

	return output, nil
}

var pipelineEpicRun = func(s *PrepSubsystem, ctx context.Context, input PipelineEpicRunInput) (PipelineEpicRunOutput, error) {
	if input.Repo == "" || input.EpicNumber <= 0 {
		return PipelineEpicRunOutput{}, core.E("pipelineEpicRun", "repo and epic number are required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Agent == "" {
		input.Agent = "codex"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	meta := input.Epic
	if meta == nil {
		reader := newPipelineForgeMetaReader(s, input.Org)
		readMeta, err := reader.GetEpicMeta(ctx, input.Repo, input.EpicNumber)
		if err != nil {
			return PipelineEpicRunOutput{}, err
		}
		meta = &readMeta
	}

	output := PipelineEpicRunOutput{
		Success:    true,
		Org:        input.Org,
		Repo:       input.Repo,
		EpicNumber: meta.Number,
		Branch:     meta.Branch,
	}

	for _, child := range meta.Children {
		if child.Checked || child.State != "open" {
			output.Skipped = append(output.Skipped, PipelineIssueRef{Number: child.Number, Title: child.Title, State: child.State, URL: child.URL})
			continue
		}
		if !pipelineIssueAutoDispatchAllowed(child.Title) {
			output.Skipped = append(output.Skipped, PipelineIssueRef{Number: child.Number, Title: child.Title, State: "manual-review", URL: child.URL})
			continue
		}
		if input.Limit > 0 && len(output.Dispatched) >= input.Limit {
			break
		}

		ref := PipelineIssueRef{Number: child.Number, Title: child.Title, State: child.State, URL: child.URL}
		if input.DryRun {
			output.Dispatched = append(output.Dispatched, ref)
			continue
		}

		if meta.Branch != "" {
			s.commentOnIssue(ctx, input.Org, input.Repo, child.Number, core.Sprintf("Target branch: `%s` (epic #%d)", meta.Branch, meta.Number))
		}

		_, _, err := dispatch(s, ctx, nil, DispatchInput{
			Org:      input.Org,
			Repo:     input.Repo,
			Task:     child.Title,
			Issue:    child.Number,
			Branch:   meta.Branch,
			Agent:    input.Agent,
			Template: input.Template,
			DryRun:   false,
		})
		if err != nil {
			return PipelineEpicRunOutput{}, core.E("pipelineEpicRun", core.Concat("failed to dispatch child issue #", core.Sprint(child.Number)), err)
		}

		output.Dispatched = append(output.Dispatched, ref)
	}

	return output, nil
}

var pipelineEpicSync = func(s *PrepSubsystem, ctx context.Context, org, repo string, number int, dryRun bool) (PipelineEpicSyncOutput, error) {
	reader := newPipelineForgeMetaReader(s, org)
	meta, err := reader.GetEpicMeta(ctx, repo, number)
	if err != nil {
		return PipelineEpicSyncOutput{}, err
	}

	checkedByNumber := map[int]bool{}
	checkedCount := 0
	for _, child := range meta.Children {
		checked := core.Lower(child.State) == "closed"
		if checked {
			checkedCount++
		}
		checkedByNumber[child.Number] = checked
	}

	lines := core.Split(meta.Body, "\n")
	updated := false
	for i, line := range lines {
		match := pipelineEpicChildPattern.FindStringSubmatch(line)
		if len(match) != 6 {
			continue
		}
		childNumber := parseIntString(match[4])
		wantChecked := checkedByNumber[childNumber]
		currentChecked := core.Lower(match[2]) == "x"
		if wantChecked == currentChecked {
			continue
		}
		mark := " "
		if wantChecked {
			mark = "x"
		}
		lines[i] = core.Concat(match[1], mark, match[3], match[4], match[5])
		updated = true
	}

	if updated && !dryRun {
		if err := pipelinePatchIssue(s, ctx, org, repo, number, map[string]any{"body": core.Join("\n", lines...)}); err != nil {
			return PipelineEpicSyncOutput{}, err
		}
	}

	return PipelineEpicSyncOutput{
		Success:    true,
		Org:        org,
		Repo:       repo,
		EpicNumber: number,
		Checked:    checkedCount,
		Total:      len(meta.Children),
		Updated:    updated,
	}, nil
}

var pipelineCreateEpicBranch = func(s *PrepSubsystem, ctx context.Context, org, repo string, epicNumber int, theme string) (string, error) {
	repository, err := pipelineGetRepo(s, ctx, org, repo)
	if err != nil {
		return "", err
	}

	defaultBranch := repository.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "dev"
	}

	shaURL := core.Sprintf("%s/api/v1/repos/%s/%s/git/refs/heads/%s", s.forgeURL, org, repo, defaultBranch)
	shaResult := HTTPGet(ctx, shaURL, s.forgeToken, "token")
	if !shaResult.OK {
		return "", core.E("pipelineCreateEpicBranch", core.Concat("failed to read ref for ", defaultBranch), nil)
	}

	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := pipelineDecodeJSON(shaResult.Value.(string), &ref, "pipelineCreateEpicBranch", "failed to decode git ref"); err != nil {
		return "", err
	}

	branch := core.Sprintf("epic/%d-%s", epicNumber, pipelineSlug(theme))
	createURL := core.Sprintf("%s/api/v1/repos/%s/%s/git/refs", s.forgeURL, org, repo)
	createResult := HTTPPost(ctx, createURL, core.JSONMarshalString(map[string]any{
		"ref": core.Concat("refs/heads/", branch),
		"sha": ref.Object.SHA,
	}), s.forgeToken, "token")
	if !createResult.OK {
		return "", core.E("pipelineCreateEpicBranch", core.Concat("failed to create branch ", branch), nil)
	}

	return branch, nil
}

func pipelineEpicTheme(issue PipelineIssueRef) string {
	haystack := core.Lower(core.Concat(issue.Title, " ", core.JSONMarshalString(issue.Labels)))
	switch {
	case core.Contains(haystack, "security"):
		return "security"
	case core.Contains(haystack, "testing"), core.Contains(haystack, "test"):
		return "testing"
	case core.Contains(haystack, "docs"), core.Contains(haystack, "doc"):
		return "docs"
	case core.Contains(haystack, "performance"), core.Contains(haystack, "perf"):
		return "performance"
	case core.HasPrefix(core.Lower(issue.Title), "feat("):
		return "features"
	default:
		return "quality"
	}
}

func pipelineEpicTitle(repo, theme string) string {
	return core.Sprintf("epic(%s): %s pipeline", repo, core.Lower(pipelineDisplayTheme(theme)))
}

func pipelineEpicBody(title, branch, theme string, children []PipelineIssueRef) string {
	builder := core.NewBuilder()
	builder.WriteString("## Overview\n\n")
	builder.WriteString(core.Sprintf("%s groups related implementation issues into a tracked epic.\n\n", title))
	if branch != "" {
		builder.WriteString(core.Sprintf("Epic branch: `%s`\n\n", branch))
	}
	builder.WriteString("## Child Issues\n\n")
	builder.WriteString(core.Sprintf("### Phase 1: %s\n", pipelineDisplayTheme(theme)))
	for _, child := range children {
		builder.WriteString(core.Sprintf("- [ ] #%d %s\n", child.Number, child.Title))
	}
	builder.WriteString("\n## Acceptance Criteria\n\n")
	builder.WriteString("- [ ] All child issues are closed\n")
	builder.WriteString("- [ ] Epic checklist is fully synced\n")
	return builder.String()
}

func pipelineIssueAutoDispatchAllowed(title string) bool {
	return !core.HasPrefix(core.Lower(core.Trim(title)), "feat(")
}
