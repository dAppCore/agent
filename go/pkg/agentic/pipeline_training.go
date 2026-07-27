// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
)

// input := PipelineTrainingCaptureInput{Org: "core", Repo: "go-io", Number: 42}
type PipelineTrainingCaptureInput struct {
	Org    string `json:"org,omitempty"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
}

// out := PipelineTrainingCaptureOutput{Success: true, Repo: "go-io", PRNumber: 42}
type PipelineTrainingCaptureOutput struct {
	Success            bool   `json:"success"`
	Org                string `json:"org,omitempty"`
	Repo               string `json:"repo"`
	PRNumber           int    `json:"pr_number"`
	CodeRabbitFindings int    `json:"coderabbit_findings"`
	DiffSource         string `json:"diff_source,omitempty"`
	JournalPath        string `json:"journal_path,omitempty"`
}

// out := PipelineTrainingExportOutput{Success: true, Exported: 3, Path: "/tmp/.core/training/export.jsonl"}
type PipelineTrainingExportOutput struct {
	Success  bool   `json:"success"`
	Exported int    `json:"exported"`
	Path     string "json:\"path\""
}

// checks, passing, failing := pipelineTrainingCheckCounts(meta.Checks)
func pipelineTrainingCheckCounts(checks []PipelineCheckMeta) (int, int, int) {
	total := len(checks)
	passing := 0
	failing := 0
	for _, check := range checks {
		switch core.Lower(firstNonEmpty(check.Conclusion, check.Status)) {
		case "success", "completed":
			passing++
		case "failure", "error", "cancelled", "timed_out":
			failing++
		}
	}
	return total, passing, failing
}

func (s *PrepSubsystem) cmdPipelineTrainingCapture(options core.Options) core.Result {
	ctx := s.commandContext()
	org, repo, number := pipelineRepoAndNumberValue(options)
	if repo == "" || number <= 0 {
		core.Print(nil, "usage: core-agent pipeline/training/capture <pr-number> --repo=<repo> [--org=core]")
		return core.Result{Value: core.E("agentic.cmdPipelineTrainingCapture", "repo and pull request number are required", nil), OK: false}
	}

	output, err := pipelineTrainingCapture(s, ctx, PipelineTrainingCaptureInput{Org: org, Repo: repo, Number: number})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "pr:        %s/%s#%d", output.Org, output.Repo, output.PRNumber)
	core.Print(nil, "findings:  %d", output.CodeRabbitFindings)
	core.Print(nil, "diff:      %s", output.DiffSource)
	core.Print(nil, "journal:   %s", output.JournalPath)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdPipelineTrainingStats(_ core.Options) core.Result {
	entries := readPipelineTrainingJournal(pipelineTrainingJournalPath())
	stats := summarisePipelineTraining(entries)

	core.Print(nil, "journal:            %s", pipelineTrainingJournalPath())
	core.Print(nil, "total_prs:          %d", stats.TotalPRs)
	core.Print(nil, "zero_finding_prs:   %d", stats.ZeroFindingPRs)
	for _, repo := range sortedTrainingRepos(stats.ByRepo) {
		core.Print(nil, "  %-16s total=%d zero=%d", repo, stats.ByRepo[repo], stats.ByRepoZeroClean[repo])
	}

	return core.Result{Value: stats, OK: true}
}

func (s *PrepSubsystem) cmdPipelineTrainingExport(_ core.Options) core.Result {
	entries := filterZeroFindingTrainingEntries(readPipelineTrainingJournal(pipelineTrainingJournalPath()))
	path := pipelineTrainingExportPath()
	if err := writePipelineTrainingExport(path, entries); err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	output := PipelineTrainingExportOutput{
		Success:  true,
		Exported: len(entries),
		Path:     path,
	}
	core.Print(nil, "exported: %d", output.Exported)
	core.Print(nil, "path:     %s", output.Path)
	return core.Result{Value: output, OK: true}
}

var pipelineTrainingCapture = func(s *PrepSubsystem, ctx context.Context, input PipelineTrainingCaptureInput) (PipelineTrainingCaptureOutput, error) {
	if input.Repo == "" || input.Number <= 0 {
		return PipelineTrainingCaptureOutput{}, core.E("pipelineTrainingCapture", "repo and pull request number are required", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if s.forgeToken == "" || s.forgeURL == "" {
		return PipelineTrainingCaptureOutput{}, core.E("pipelineTrainingCapture", "Forge access is required to capture PR training data", nil)
	}

	reader := newPipelineForgeMetaReader(s, input.Org)
	meta, err := reader.GetPRMeta(ctx, input.Repo, input.Number)
	if err != nil {
		return PipelineTrainingCaptureOutput{}, err
	}
	if meta.State != "merged" {
		return PipelineTrainingCaptureOutput{}, core.E("pipelineTrainingCapture", core.Concat("pull request is not merged: ", core.Sprint(input.Number)), nil)
	}

	diff, diffSource, err := pipelineTrainingReadDiff(s, ctx, input.Org, input.Repo, input.Number, meta)
	if err != nil {
		return PipelineTrainingCaptureOutput{}, err
	}

	checksTotal, checksPassing, checksFailing := pipelineTrainingCheckCounts(meta.Checks)
	entry := PipelineTrainingEntry{
		CapturedAt:            time.Now().UTC().Format(time.RFC3339),
		Org:                   input.Org,
		Repo:                  input.Repo,
		PRNumber:              meta.Number,
		PRURL:                 meta.URL,
		State:                 meta.State,
		Mergeable:             meta.Mergeable,
		BaseBranch:            meta.BaseBranch,
		HeadBranch:            meta.HeadBranch,
		HeadSHA:               meta.HeadSHA,
		ChecksTotal:           checksTotal,
		ChecksPassing:         checksPassing,
		ChecksFailing:         checksFailing,
		ReviewThreadsTotal:    meta.ThreadsTotal,
		ReviewThreadsResolved: meta.ThreadsResolved,
		CodeRabbitFindings:    meta.ThreadsTotal,
		FindingSource:         "review_threads_total",
		Diff:                  diff,
		DiffSource:            diffSource,
	}
	if err := appendJSONLRecord(pipelineTrainingJournalPath(), entry); err != nil {
		return PipelineTrainingCaptureOutput{}, err
	}

	return PipelineTrainingCaptureOutput{
		Success:            true,
		Org:                input.Org,
		Repo:               input.Repo,
		PRNumber:           meta.Number,
		CodeRabbitFindings: entry.CodeRabbitFindings,
		DiffSource:         diffSource,
		JournalPath:        pipelineTrainingJournalPath(),
	}, nil
}

// diff, source, err := s.pipelineTrainingReadDiff(ctx, "core", "go-io", 42, meta)
var pipelineTrainingReadDiff = func(s *PrepSubsystem, ctx context.Context, org, repo string, number int, meta PipelinePRMeta) (string, string, error) {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d.diff", s.forgeURL, org, repo, number)
	result := HTTPGet(ctx, url, s.forgeToken, "token")
	if result.OK && core.Trim(resultText(result)) != "" {
		return resultText(result), "forge.pull.diff", nil
	}
	return pipelineTrainingReadGitDiff(s, ctx, org, repo, meta)
}

// diff, source, err := s.pipelineTrainingReadGitDiff(ctx, "core", "go-io", meta)
var pipelineTrainingReadGitDiff = func(s *PrepSubsystem, ctx context.Context, org, repo string, meta PipelinePRMeta) (string, string, error) {
	repoDir := s.localRepoDir(org, repo)
	if repoDir == "" || !fs.Exists(repoDir).OK || fs.IsFile(repoDir).OK {
		return "", "", core.E("pipelineTrainingReadGitDiff", "no local repo checkout and Forge diff endpoint was unavailable", nil)
	}

	process := s.Core().Process()
	if meta.HeadSHA != "" {
		showResult := process.RunIn(ctx, repoDir, "git", "show", "--format=", meta.HeadSHA)
		if showResult.OK && core.Trim(resultText(showResult)) != "" {
			return resultText(showResult), "git.show", nil
		}
	}

	if meta.BaseBranch != "" && meta.HeadBranch != "" {
		if fetchBaseResult := process.RunIn(ctx, repoDir, "git", "fetch", "origin", meta.BaseBranch); !fetchBaseResult.OK {
			core.Warn("pipelineTrainingReadGitDiff: failed to fetch base branch", "repo", repo, "branch", meta.BaseBranch, "reason", fetchBaseResult.Value)
		}
		if fetchHeadResult := process.RunIn(ctx, repoDir, "git", "fetch", "origin", meta.HeadBranch); !fetchHeadResult.OK {
			core.Warn("pipelineTrainingReadGitDiff: failed to fetch head branch", "repo", repo, "branch", meta.HeadBranch, "reason", fetchHeadResult.Value)
		}
		diffResult := process.RunIn(ctx, repoDir, "git", "diff", core.Concat("origin/", meta.BaseBranch), core.Concat("origin/", meta.HeadBranch))
		if diffResult.OK && core.Trim(resultText(diffResult)) != "" {
			return resultText(diffResult), "git.diff", nil
		}
	}

	return "", "", core.E("pipelineTrainingReadGitDiff", "unable to resolve pull request diff", nil)
}
