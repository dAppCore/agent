// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// writeReviewScript drops a fake reviewer binary on PATH so reviewRepo's
// process.RunIn call returns controlled output without invoking the real
// coderabbit/codex CLI. exitCode lets a test drive the non-OK arm.
func writeReviewScript(t *testing.T, name, stdout string, exitCode int) {
	t.Helper()
	binDir := t.TempDir()
	scriptPath := core.JoinPath(binDir, name)
	body := core.Concat("#!/bin/sh\ncat <<'REVIEW_EOF'\n", stdout, "\nREVIEW_EOF\nexit ", core.Itoa(exitCode), "\n")
	core.RequireTrue(t, core.WriteFile(scriptPath, []byte(body), 0o755).OK)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))
}

// --- reviewQueueReviewers (codex + default arms) ---

func TestReviewqueue_ReviewQueueReviewers_Good_Codex(t *testing.T) {
	reviewers := reviewQueueReviewers("codex")
	core.AssertLen(t, reviewers, 1)
	core.AssertEqual(t, "codex", reviewers[0])
}

func TestReviewqueue_ReviewQueueReviewers_Good_Default(t *testing.T) {
	// Unknown reviewer name falls back to coderabbit only.
	reviewers := reviewQueueReviewers("unknown-reviewer")
	core.AssertLen(t, reviewers, 1)
	core.AssertEqual(t, "coderabbit", reviewers[0])
}

func TestReviewqueue_ReviewQueueReviewers_Ugly_Empty(t *testing.T) {
	// Empty + whitespace both resolve to the default coderabbit.
	core.AssertEqual(t, []string{"coderabbit"}, reviewQueueReviewers(""))
	core.AssertEqual(t, []string{"coderabbit"}, reviewQueueReviewers("  "))
}

// --- parseRetryAfter (no-match falls back to default) ---

func TestReviewqueue_ParseRetryAfter_Ugly_NoMatchDefaults(t *testing.T) {
	// A message with no "N minutes" shape returns the 5-minute default.
	core.AssertEqual(t, 5*time.Minute, parseRetryAfter("please slow down"))
	core.AssertEqual(t, 5*time.Minute, parseRetryAfter("rate limited, try later"))
}

// --- compileRetryAfterPattern ---

func TestReviewqueue_CompileRetryAfterPattern_Good_Case(t *testing.T) {
	// The package-level pattern compiled successfully and matches the message
	// shape parseRetryAfter relies on.
	core.AssertNotNil(t, retryAfterPattern)
	core.AssertTrue(t, retryAfterPattern.MatchString("retry after 2 minutes and 5 seconds"))
}

// --- reviewRepo: clean → merged (happy path through pushAndMerge) ---

func TestReviewqueue_ReviewRepo_Good_CleanMerged(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "No findings — LGTM", 0)

	origMerge := pushAndMerge
	t.Cleanup(func() { pushAndMerge = origMerge })
	merged := false
	pushAndMerge = func(_ *PrepSubsystem, _ context.Context, _, repo string) error {
		merged = true
		core.AssertEqual(t, "go-io", repo)
		return nil
	}

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", false, false)

	core.AssertEqual(t, "clean", result.Verdict)
	core.AssertEqual(t, 0, result.Findings)
	core.AssertEqual(t, "merged", result.Action)
	core.AssertTrue(t, merged)
}

// --- reviewRepo: clean but push/merge fails ---

func TestReviewqueue_ReviewRepo_Bad_CleanPushFailed(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "no issues found", 0)

	origMerge := pushAndMerge
	t.Cleanup(func() { pushAndMerge = origMerge })
	pushAndMerge = func(_ *PrepSubsystem, _ context.Context, _, _ string) error {
		return core.E("pushAndMerge", "push failed: remote rejected", nil)
	}

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", false, false)

	core.AssertEqual(t, "clean", result.Verdict)
	core.AssertContains(t, result.Action, "push failed")
}

// --- reviewRepo: clean + dry run skips the merge ---

func TestReviewqueue_ReviewRepo_Good_CleanDryRun(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "No findings", 0)

	origMerge := pushAndMerge
	t.Cleanup(func() { pushAndMerge = origMerge })
	pushAndMerge = func(_ *PrepSubsystem, _ context.Context, _, _ string) error {
		t.Fatal("dry run must not push/merge")
		return nil
	}

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", true, false)

	core.AssertEqual(t, "clean", result.Verdict)
	core.AssertEqual(t, "skipped (dry run)", result.Action)
}

// --- reviewRepo: clean + local-only stops before push ---

func TestReviewqueue_ReviewRepo_Good_CleanLocalOnly(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "LGTM", 0)

	origMerge := pushAndMerge
	t.Cleanup(func() { pushAndMerge = origMerge })
	pushAndMerge = func(_ *PrepSubsystem, _ context.Context, _, _ string) error {
		t.Fatal("local-only must not push/merge")
		return nil
	}

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", false, true)

	core.AssertEqual(t, "clean", result.Verdict)
	core.AssertEqual(t, "clean (local only)", result.Action)
}

// --- reviewRepo: findings → fix dispatched ---

func TestReviewqueue_ReviewRepo_Good_FindingsDispatched(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "- Missing nil check in handler.go:42\n- Unused import", 0)

	origDispatch := dispatchFixFromQueue
	t.Cleanup(func() { dispatchFixFromQueue = origDispatch })
	var dispatchedRepo, dispatchedTask string
	dispatchFixFromQueue = func(_ *PrepSubsystem, _ context.Context, repo, task string) error {
		dispatchedRepo = repo
		dispatchedTask = task
		return nil
	}

	repoDir := t.TempDir()
	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), repoDir, "go-io", "coderabbit", false, false)

	core.AssertEqual(t, "findings", result.Verdict)
	core.AssertEqual(t, 2, result.Findings)
	core.AssertEqual(t, "fix_dispatched", result.Action)
	core.AssertEqual(t, "go-io", dispatchedRepo)
	core.AssertContains(t, dispatchedTask, "coderabbit-findings.txt")

	// The findings file is written into the repo's .core dir for the fix agent.
	findingsFile := core.JoinPath(repoDir, ".core", "coderabbit-findings.txt")
	core.AssertTrue(t, fs.IsFile(findingsFile))
}

// --- reviewRepo: findings but fix dispatch fails ---

func TestReviewqueue_ReviewRepo_Bad_FindingsDispatchFailed(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "- A real finding here", 0)

	origDispatch := dispatchFixFromQueue
	t.Cleanup(func() { dispatchFixFromQueue = origDispatch })
	dispatchFixFromQueue = func(_ *PrepSubsystem, _ context.Context, _, _ string) error {
		return core.E("dispatchFixFromQueue", "dispatch failed for go-io", nil)
	}

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", false, false)

	core.AssertEqual(t, "findings", result.Verdict)
	core.AssertEqual(t, "fix_dispatch_failed", result.Action)
	core.AssertContains(t, result.Detail, "dispatch failed")
}

// --- reviewRepo: findings + dry run skips the fix dispatch ---

func TestReviewqueue_ReviewRepo_Good_FindingsDryRun(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "- Finding one\n- Finding two", 0)

	origDispatch := dispatchFixFromQueue
	t.Cleanup(func() { dispatchFixFromQueue = origDispatch })
	dispatchFixFromQueue = func(_ *PrepSubsystem, _ context.Context, _, _ string) error {
		t.Fatal("dry run must not dispatch a fix agent")
		return nil
	}

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", true, false)

	core.AssertEqual(t, "findings", result.Verdict)
	core.AssertEqual(t, "skipped (dry run)", result.Action)
}

// --- reviewRepo: rate limit detected from the reviewer output ---

func TestReviewqueue_ReviewRepo_Ugly_RateLimitFromOutput(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	// The real coderabbit exits 0 and prints the rate-limit notice to stdout;
	// the process action only surfaces stdout when the command succeeds.
	writeReviewScript(t, "coderabbit", "Rate limit exceeded — please try after 3 minutes", 0)

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", false, false)

	core.AssertEqual(t, "rate_limited", result.Verdict)
	core.AssertContains(t, result.Detail, "Rate limit exceeded")
}

// --- reviewRepo: rate-limit state on disk short-circuits coderabbit ---

func TestReviewqueue_ReviewRepo_Ugly_RateLimitFromState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	// Persist an active rate-limit window; coderabbit (unlike codex) honours it.
	ratePath := core.JoinPath(home, ".core", "coderabbit-ratelimit.json")
	core.RequireTrue(t, fs.EnsureDir(core.PathDir(ratePath)).OK)
	core.RequireTrue(t, fs.Write(ratePath, core.JSONMarshalString(&RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(time.Hour),
		Message: "still cooling down",
	})).OK)

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", false, false)

	core.AssertEqual(t, "rate_limited", result.Verdict)
	core.AssertContains(t, result.Detail, "retry after")
}

// --- reviewRepo: reviewer command errors with no clean marker ---

func TestReviewqueue_ReviewRepo_Bad_CommandError(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	// Non-zero exit with no clean marker → error verdict. The process action
	// returns the error (not stdout) in the Result on failure, so reviewRepo's
	// `output, _ := r.Value.(string)` is empty and Detail comes through empty.
	writeReviewScript(t, "coderabbit", "fatal: could not read CLAUDE.md", 1)

	s := newPrepWithProcess()
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "coderabbit", false, false)

	core.AssertEqual(t, "error", result.Verdict)
	core.AssertEmpty(t, result.Detail)
}

// --- reviewRepo: empty reviewer defaults to coderabbit ---

func TestReviewqueue_ReviewRepo_Ugly_EmptyReviewerDefaultsCoderabbit(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	writeReviewScript(t, "coderabbit", "No findings", 0)

	origMerge := pushAndMerge
	t.Cleanup(func() { pushAndMerge = origMerge })
	pushAndMerge = func(_ *PrepSubsystem, _ context.Context, _, _ string) error { return nil }

	s := newPrepWithProcess()
	// Empty reviewer string still has the rate-limit guard run (reviewer != "codex").
	result := s.reviewRepo(context.Background(), t.TempDir(), "go-io", "", true, false)

	core.AssertEqual(t, "clean", result.Verdict)
	core.AssertEqual(t, "skipped (dry run)", result.Action)
}

// --- runPRManageLoop: context cancellation exits the loop ---

func TestReviewqueue_RunPRManageLoop_Good_CancelExits(t *testing.T) {
	s := newPrepWithProcess()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		s.runPRManageLoop(ctx, time.Hour)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runPRManageLoop did not return after context cancellation")
	}
}

func TestReviewqueue_RunPRManageLoop_Bad_GuardsInvalidArgs(t *testing.T) {
	s := newPrepWithProcess()
	// Nil context and non-positive interval both return immediately.
	core.AssertNotPanics(t, func() {
		s.runPRManageLoop(nil, time.Hour)
		s.runPRManageLoop(context.Background(), 0)
	})
}

// --- cmdReviewQueue: prints rate-limit, processed, and skipped lines ---

func TestReviewqueue_CmdReviewQueue_Good_PrintsAllSections(t *testing.T) {
	s := newPrepWithProcess()

	orig := reviewQueue
	t.Cleanup(func() { reviewQueue = orig })
	reviewQueue = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, input ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
		core.AssertEqual(t, 3, input.Limit)
		core.AssertTrue(t, input.DryRun)
		return nil, ReviewQueueOutput{
			Success:   true,
			Processed: []ReviewResult{{Repo: "go-io", Verdict: "clean", Action: "merged"}},
			Skipped:   []string{"go-scm (limit reached)"},
			RateLimit: &RateLimitInfo{Limited: true, Message: "retry after 5 minutes"},
		}, nil
	}

	var out string
	captureOK := false
	captured := captureStdout(t, func() {
		result := s.cmdReviewQueue(core.NewOptions(
			core.Option{Key: "limit", Value: 3},
			core.Option{Key: "dry-run", Value: true},
		))
		captureOK = result.OK
	})
	out = captured

	core.AssertTrue(t, captureOK)
	core.AssertContains(t, out, "rate limit: retry after 5 minutes")
	core.AssertContains(t, out, "go-io: clean (merged)")
	core.AssertContains(t, out, "skipped: go-scm (limit reached)")
}

func TestReviewqueue_CmdReviewQueue_Bad_PropagatesError(t *testing.T) {
	s := newPrepWithProcess()

	orig := reviewQueue
	t.Cleanup(func() { reviewQueue = orig })
	reviewQueue = func(_ *PrepSubsystem, _ context.Context, _ *mcp.CallToolRequest, _ ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
		return nil, ReviewQueueOutput{}, core.E("agentic.review-queue", "queue exploded", nil)
	}

	r := s.cmdReviewQueue(core.NewOptions())
	core.AssertFalse(t, r.OK)
	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "queue exploded")
}

// --- storeReviewOutput: findings verdict recorded in the journal ---

func TestReviewqueue_StoreReviewOutput_Good_FindingsVerdict(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	s := newPrepWithProcess()
	// Output without a clean marker is recorded with verdict "findings".
	s.storeReviewOutput(t.TempDir(), "go-io", "coderabbit", "- A finding that needs fixing")

	jsonlPath := core.JoinPath(home, ".core", "training", "reviews", "reviews.jsonl")
	core.RequireTrue(t, fs.IsFile(jsonlPath))
	readResult := fs.Read(jsonlPath)
	core.RequireTrue(t, readResult.OK)
	core.AssertContains(t, readResult.Value.(string), "\"verdict\":\"findings\"")
	core.AssertContains(t, readResult.Value.(string), "\"repo\":\"go-io\"")
}

func TestReviewqueue_StoreReviewOutput_Good_CleanVerdict(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	s := newPrepWithProcess()
	s.storeReviewOutput(t.TempDir(), "go-io", "coderabbit", "No findings — all good")

	jsonlPath := core.JoinPath(home, ".core", "training", "reviews", "reviews.jsonl")
	core.RequireTrue(t, fs.IsFile(jsonlPath))
	readResult := fs.Read(jsonlPath)
	core.RequireTrue(t, readResult.OK)
	core.AssertContains(t, readResult.Value.(string), "\"verdict\":\"clean\"")
}
