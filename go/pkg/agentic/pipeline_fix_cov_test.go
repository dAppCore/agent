// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestPipelineFixCov_CmdReviews_Good_PrintsCommentAction — the fix/reviews
// command wrapper posts the review comment and prints the pr/action/message
// summary, returning the typed output. HTTP-only seam.
func TestPipelineFixCov_CmdReviews_Good_PrintsCommentAction(t *testing.T) {
	repo := newPipelineTestRepo()
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixReviews(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "7"},
		))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineFixOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "comment", typed.Action)
	core.AssertContains(t, output, "pr:      core/go-io#7")
	core.AssertContains(t, output, "action:  comment")
	core.AssertContains(t, repo.Comments[7][0], "Can you fix the code reviews?")
}

// TestPipelineFixCov_CmdReviews_Bad_MissingNumber — the wrapper prints usage and
// fails when the pull request number is absent.
func TestPipelineFixCov_CmdReviews_Bad_MissingNumber(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixReviews(core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	})

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "repo and pull request number are required")
	core.AssertContains(t, output, "usage: core-agent pipeline/fix/reviews")
}

// TestPipelineFixCov_CmdConflicts_Good_PrintsCommentAction — the fix/conflicts
// command wrapper posts the conflict comment and prints its summary.
func TestPipelineFixCov_CmdConflicts_Good_PrintsCommentAction(t *testing.T) {
	repo := newPipelineTestRepo()
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixConflicts(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "8"},
		))
	})

	core.RequireTrue(t, result.OK)
	core.AssertContains(t, output, "pr:      core/go-io#8")
	core.AssertContains(t, repo.Comments[8][0], "Can you fix the merge conflict?")
}

// TestPipelineFixCov_CmdConflicts_Ugly_SeamErrorPrints — a seam error is
// printed and surfaced as a failed result. The conflicts seam never errors for
// valid wrapper input (it only posts a comment), so it is stubbed to exercise
// the wrapper's error-print arm.
func TestPipelineFixCov_CmdConflicts_Ugly_SeamErrorPrints(t *testing.T) {
	original := pipelineFixConflicts
	t.Cleanup(func() { pipelineFixConflicts = original })
	pipelineFixConflicts = func(_ *PrepSubsystem, _ context.Context, _ PipelineFixInput) (PipelineFixOutput, error) {
		return PipelineFixOutput{}, core.E("pipelineFixConflicts", "forge unreachable", nil)
	}

	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixConflicts(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "8"},
		))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
	core.AssertContains(t, output, "forge unreachable")
}

// TestPipelineFixCov_CmdThreads_Ugly_UnknownPRPrintsError — the fix/threads
// wrapper surfaces the GetPRMeta read error for a non-existent pull request
// (the wrapper's error-print arm, reached through real HTTP 404).
func TestPipelineFixCov_CmdThreads_Ugly_UnknownPRPrintsError(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixThreads(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "999"},
		))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
	core.AssertContains(t, output, "failed to read PR")
}

// TestPipelineFixCov_CmdThreads_Good_PrintsCommentForUnresolved — the
// fix/threads command wrapper reads the PR meta, comments on the unresolved
// threads, and prints the action summary.
func TestPipelineFixCov_CmdThreads_Good_PrintsCommentForUnresolved(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Pulls[5] = &pipelineTestPR{
		Number:                5,
		Title:                 "Needs follow-up",
		State:                 "open",
		Mergeable:             boolPtr(true),
		HeadRef:               "agent/threads",
		HeadSHA:               "sha-threads",
		BaseRef:               "dev",
		ReviewThreadsTotal:    3,
		ReviewThreadsResolved: 1,
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixThreads(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "5"},
		))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineFixOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "comment", typed.Action)
	core.AssertContains(t, output, "action:  comment")
	core.AssertContains(t, repo.Comments[5][0], "2 remaining review thread")
}

// TestPipelineFixCov_CmdFormat_Good_PrintsSummary — the fix/format command
// wrapper prints the files/committed/pushed summary. The pipelineFixFormat seam
// is stubbed so no real gofmt/git subprocess runs inside captureStdout (the
// real seam shells out before the dry-run check, which would corrupt the
// captured pipe).
func TestPipelineFixCov_CmdFormat_Good_PrintsSummary(t *testing.T) {
	original := pipelineFixFormat
	t.Cleanup(func() { pipelineFixFormat = original })
	pipelineFixFormat = func(_ *PrepSubsystem, _ context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
		return PipelineFixOutput{
			Success:   true,
			Org:       input.Org,
			Repo:      input.Repo,
			Number:    input.Number,
			Action:    "format",
			Files:     4,
			Committed: true,
			Pushed:    false,
			Message:   "formatted Go files",
		}, nil
	}

	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixFormat(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "12"},
			core.Option{Key: "repo-dir", Value: "/tmp/whatever"},
		))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineFixOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 4, typed.Files)
	core.AssertContains(t, output, "pr:        core/go-io#12")
	core.AssertContains(t, output, "files:     4")
	core.AssertContains(t, output, "committed: true")
	core.AssertContains(t, output, "message:   formatted Go files")
}

// TestPipelineFixCov_CmdFormat_Bad_MissingNumber — the wrapper prints usage and
// fails before any seam call when the pull request number is absent.
func TestPipelineFixCov_CmdFormat_Bad_MissingNumber(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixFormat(core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	})

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "repo and pull request number are required")
	core.AssertContains(t, output, "usage: core-agent pipeline/fix/format")
}

// TestPipelineFixCov_CmdFormat_Ugly_SeamErrorPrints — a seam error is printed
// and surfaced as a failed result.
func TestPipelineFixCov_CmdFormat_Ugly_SeamErrorPrints(t *testing.T) {
	original := pipelineFixFormat
	t.Cleanup(func() { pipelineFixFormat = original })
	pipelineFixFormat = func(_ *PrepSubsystem, _ context.Context, _ PipelineFixInput) (PipelineFixOutput, error) {
		return PipelineFixOutput{}, core.E("pipelineFixFormat", "gofmt failed", nil)
	}

	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineFixFormat(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "12"},
			core.Option{Key: "repo-dir", Value: "/tmp/whatever"},
		))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
	core.AssertContains(t, output, "gofmt failed")
}
