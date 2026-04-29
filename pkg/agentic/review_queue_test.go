// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- countFindings (extended beyond paths_test.go) ---

func TestReviewqueue_CountFindings_Good_BulletFindings(t *testing.T) {
	output := `Review:
- Missing error check in handler.go:42
- Unused import in config.go
* Race condition in worker pool`
	core.AssertEqual(t, 3, countFindings(output))
}

func TestReviewqueue_CountFindings_Good_IssueKeyword(t *testing.T) {
	output := `Line 10: Issue: variable shadowing
Line 25: Finding: unchecked return value`
	core.AssertEqual(t, 2, countFindings(output))
}

func TestReviewqueue_CountFindings_Good_DefaultOneIfNotClean(t *testing.T) {
	output := "Some output without markers but also not explicitly clean"
	count := countFindings(output)
	core.AssertEqual(t, 1, count)
	core.AssertNotContains(t, output, "No findings")
}

func TestReviewqueue_CountFindings_Good_MixedContent(t *testing.T) {
	output := `Summary of review:
The code is generally well structured.
- Missing nil check
Some commentary here
* Redundant allocation`
	core.AssertEqual(t, 2, countFindings(output))
}

// --- parseRetryAfter (extended) ---

func TestReviewqueue_ParseRetryAfter_Good_SingleMinuteAndSeconds(t *testing.T) {
	message := "try after 1 minute and 30 seconds"
	delay := parseRetryAfter(message)
	core.AssertEqual(t, 1*time.Minute+30*time.Second, delay)
	core.AssertTrue(t, delay > time.Minute)
}

func TestReviewqueue_ParseRetryAfter_Bad_EmptyMessage(t *testing.T) {
	message := ""
	delay := parseRetryAfter(message)
	core.AssertEqual(t, 5*time.Minute, delay)
	core.AssertTrue(t, delay > 0)
}

func TestReviewqueue_ReviewQueueReviewers_Good_Both(t *testing.T) {
	reviewers := reviewQueueReviewers("both")
	core.AssertLen(t, reviewers, 2)
	core.AssertEqual(t, "codex", reviewers[0])
	core.AssertEqual(t, "coderabbit", reviewers[1])
}

func TestReviewqueue_ReviewRepo_Good_CodexBypassesCodeRabbitRateLimit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	ratePath := core.JoinPath(home, ".core", "coderabbit-ratelimit.json")
	fs.EnsureDir(core.PathDir(ratePath))
	fs.Write(ratePath, core.JSONMarshalString(&RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(time.Hour),
		Message: "rate limited",
	}))

	binDir := t.TempDir()
	scriptPath := core.JoinPath(binDir, "codex")
	core.RequireTrue(t, core.WriteFile(scriptPath, []byte("#!/bin/sh\necho 'No findings'\n"), 0o755).OK)
	t.Setenv("PATH", binDir+string(core.PathListSeparator)+core.Getenv("PATH"))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.reviewRepo(context.Background(), t.TempDir(), "repo", "codex", true, true)

	core.AssertEqual(t, "clean", result.Verdict)
	core.AssertEqual(t, "skipped (dry run)", result.Action)
}
