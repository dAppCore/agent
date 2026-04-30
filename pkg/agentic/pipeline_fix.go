// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

type PipelineFixInput struct {
	Org          string `json:"org,omitempty"`
	Repo         string `json:"repo"`
	Number       int    `json:"number"`
	WorkspaceDir string `json:"workspace_dir,omitempty"`
	Commit       bool   `json:"commit,omitempty"`
	Push         bool   `json:"push,omitempty"`
	DryRun       bool   `json:"dry_run,omitempty"`
}

type PipelineFixOutput struct {
	Success   bool   `json:"success"`
	Org       string `json:"org,omitempty"`
	Repo      string `json:"repo"`
	Number    int    `json:"number"`
	Action    string `json:"action"`
	Message   string `json:"message,omitempty"`
	Files     int    `json:"files,omitempty"`
	Committed bool   `json:"committed,omitempty"`
	Pushed    bool   `json:"pushed,omitempty"`
}

func (s *PrepSubsystem) cmdPipelineFixReviews(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/fix/reviews <pr-number> --repo=<repo> [--org=core] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineFixReviews", "repo and pull request number are required", nil), OK: false}
	}

	output, err := pipelineFixReviews(s, ctx, PipelineFixInput{Org: org, Repo: repo, Number: number, DryRun: optionBoolValue(options, "dry_run", "dry-run")})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "pr:      %s/%s#%d", output.Org, output.Repo, output.Number)
	core.Print(nil, "action:  %s", output.Action)
	core.Print(nil, "message: %s", output.Message)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPipelineFixConflicts(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/fix/conflicts <pr-number> --repo=<repo> [--org=core] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineFixConflicts", "repo and pull request number are required", nil), OK: false}
	}

	output, err := pipelineFixConflicts(s, ctx, PipelineFixInput{Org: org, Repo: repo, Number: number, DryRun: optionBoolValue(options, "dry_run", "dry-run")})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "pr:      %s/%s#%d", output.Org, output.Repo, output.Number)
	core.Print(nil, "action:  %s", output.Action)
	core.Print(nil, "message: %s", output.Message)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPipelineFixFormat(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/fix/format <pr-number> --repo=<repo> [--org=core] [--workspace=<name>|--repo-dir=<path>] [--commit] [--push] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineFixFormat", "repo and pull request number are required", nil), OK: false}
	}

	output, err := pipelineFixFormat(s, ctx, PipelineFixInput{
		Org:          org,
		Repo:         repo,
		Number:       number,
		WorkspaceDir: pipelineWorkspaceDir(options),
		Commit:       optionBoolValue(options, "commit"),
		Push:         optionBoolValue(options, "push"),
		DryRun:       optionBoolValue(options, "dry_run", "dry-run"),
	})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "pr:        %s/%s#%d", output.Org, output.Repo, output.Number)
	core.Print(nil, "action:    %s", output.Action)
	core.Print(nil, "files:     %d", output.Files)
	core.Print(nil, "committed: %v", output.Committed)
	core.Print(nil, "pushed:    %v", output.Pushed)
	if output.Message != "" {
		core.Print(nil, "message:   %s", output.Message)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPipelineFixThreads(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/fix/threads <pr-number> --repo=<repo> [--org=core] [--dry-run]")
		return core.Result{Value: core.E("agentic.cmdPipelineFixThreads", "repo and pull request number are required", nil), OK: false}
	}

	output, err := pipelineFixThreads(s, ctx, PipelineFixInput{Org: org, Repo: repo, Number: number, DryRun: optionBoolValue(options, "dry_run", "dry-run")})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "pr:      %s/%s#%d", output.Org, output.Repo, output.Number)
	core.Print(nil, "action:  %s", output.Action)
	core.Print(nil, "message: %s", output.Message)
	return core.Result{Value: output, OK: true}
}

var pipelineFixReviews = func(s *PrepSubsystem, ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	if input.Repo == "" || input.Number <= 0 {
		return PipelineFixOutput{}, core.E("pipelineFixReviews", "repo and pull request number are required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if !input.DryRun {
		s.commentOnIssue(ctx, input.Org, input.Repo, input.Number, "Can you fix the code reviews?")
	}
	return PipelineFixOutput{
		Success: true,
		Org:     input.Org,
		Repo:    input.Repo,
		Number:  input.Number,
		Action:  "comment",
		Message: "Can you fix the code reviews?",
	}, nil
}

var pipelineFixConflicts = func(s *PrepSubsystem, ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	if input.Repo == "" || input.Number <= 0 {
		return PipelineFixOutput{}, core.E("pipelineFixConflicts", "repo and pull request number are required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if !input.DryRun {
		s.commentOnIssue(ctx, input.Org, input.Repo, input.Number, "Can you fix the merge conflict?")
	}
	return PipelineFixOutput{
		Success: true,
		Org:     input.Org,
		Repo:    input.Repo,
		Number:  input.Number,
		Action:  "comment",
		Message: "Can you fix the merge conflict?",
	}, nil
}

var pipelineFixFormat = func(s *PrepSubsystem, ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	if input.Repo == "" || input.Number <= 0 {
		return PipelineFixOutput{}, core.E("pipelineFixFormat", "repo and pull request number are required", nil)
	}
	if input.WorkspaceDir == "" {
		return PipelineFixOutput{}, core.E("pipelineFixFormat", "workspace or repo_dir is required for formatting", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}

	process := s.Core().Process()
	fileList := process.RunIn(ctx, input.WorkspaceDir, "rg", "--files", "-g", "*.go")
	if !fileList.OK {
		fileList = process.RunIn(ctx, input.WorkspaceDir, "find", ".", "-name", "*.go", "-type", "f")
	}

	goFiles := []string{}
	if fileList.OK {
		for _, file := range core.Split(core.Trim(resultText(fileList)), "\n") {
			trimmed := core.Trim(file)
			if trimmed != "" {
				goFiles = append(goFiles, trimmed)
			}
		}
	}

	output := PipelineFixOutput{
		Success: true,
		Org:     input.Org,
		Repo:    input.Repo,
		Number:  input.Number,
		Action:  "format",
		Files:   len(goFiles),
		Message: "no Go files matched",
	}
	if len(goFiles) == 0 {
		return output, nil
	}

	output.Message = "formatted Go files"
	if input.DryRun {
		return output, nil
	}

	formatResult := process.RunIn(ctx, input.WorkspaceDir, "sh", "-lc", "files=$(rg --files -g '*.go' 2>/dev/null || find . -name '*.go' -type f); if [ -n \"$files\" ]; then gofmt -w $files; fi")
	if !formatResult.OK {
		return PipelineFixOutput{}, core.E("pipelineFixFormat", "gofmt failed", nil)
	}

	statusResult := process.RunIn(ctx, input.WorkspaceDir, "git", "status", "--porcelain")
	if !statusResult.OK {
		return output, nil
	}
	if core.Trim(resultText(statusResult)) == "" {
		output.Message = "repo already formatted"
		return output, nil
	}

	if input.Commit || input.Push {
		addResult := process.RunIn(ctx, input.WorkspaceDir, "git", "add", "-A")
		if !addResult.OK {
			return PipelineFixOutput{}, core.E("pipelineFixFormat", "git add failed", nil)
		}
		commitResult := process.RunIn(ctx, input.WorkspaceDir, "git", "commit", "-m", "chore: apply formatting")
		if !commitResult.OK {
			return PipelineFixOutput{}, core.E("pipelineFixFormat", "git commit failed", nil)
		}
		output.Committed = true
	}

	if input.Push {
		pushResult := process.RunIn(ctx, input.WorkspaceDir, "git", "push")
		if !pushResult.OK {
			return PipelineFixOutput{}, core.E("pipelineFixFormat", "git push failed", nil)
		}
		output.Pushed = true
	}

	return output, nil
}

var pipelineFixThreads = func(s *PrepSubsystem, ctx context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
	if input.Repo == "" || input.Number <= 0 {
		return PipelineFixOutput{}, core.E("pipelineFixThreads", "repo and pull request number are required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}

	reader := newPipelineForgeMetaReader(s, input.Org)
	meta, err := reader.GetPRMeta(ctx, input.Repo, input.Number)
	if err != nil {
		return PipelineFixOutput{}, err
	}
	unresolved := meta.ThreadsTotal - meta.ThreadsResolved
	if unresolved <= 0 {
		return PipelineFixOutput{
			Success: true,
			Org:     input.Org,
			Repo:    input.Repo,
			Number:  input.Number,
			Action:  "noop",
			Message: "no unresolved review threads remain",
		}, nil
	}

	message := core.Sprintf("Please resolve the %d remaining review thread(s) after the latest fix.", unresolved)
	if !input.DryRun {
		s.commentOnIssue(ctx, input.Org, input.Repo, input.Number, message)
	}

	return PipelineFixOutput{
		Success: true,
		Org:     input.Org,
		Repo:    input.Repo,
		Number:  input.Number,
		Action:  "comment",
		Message: message,
	}, nil
}
