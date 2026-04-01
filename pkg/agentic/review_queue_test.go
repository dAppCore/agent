// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- countFindings (extended beyond paths_test.go) ---

func TestReviewqueue_CountFindings_Good_BulletFindings(t *testing.T) {
	output := `Review:
- Missing error check in handler.go:42
- Unused import in config.go
* Race condition in worker pool`
	assert.Equal(t, 3, countFindings(output))
}

func TestReviewqueue_CountFindings_Good_IssueKeyword(t *testing.T) {
	output := `Line 10: Issue: variable shadowing
Line 25: Finding: unchecked return value`
	assert.Equal(t, 2, countFindings(output))
}

func TestReviewqueue_CountFindings_Good_DefaultOneIfNotClean(t *testing.T) {
	output := "Some output without markers but also not explicitly clean"
	assert.Equal(t, 1, countFindings(output))
}

func TestReviewqueue_CountFindings_Good_MixedContent(t *testing.T) {
	output := `Summary of review:
The code is generally well structured.
- Missing nil check
Some commentary here
* Redundant allocation`
	assert.Equal(t, 2, countFindings(output))
}

// --- parseRetryAfter (extended) ---

func TestReviewqueue_ParseRetryAfter_Good_SingleMinuteAndSeconds(t *testing.T) {
	d := parseRetryAfter("try after 1 minute and 30 seconds")
	assert.Equal(t, 1*time.Minute+30*time.Second, d)
}

func TestReviewqueue_ParseRetryAfter_Bad_EmptyMessage(t *testing.T) {
	d := parseRetryAfter("")
	assert.Equal(t, 5*time.Minute, d)
}

func TestReviewqueue_ReviewQueueReviewers_Good_Both(t *testing.T) {
	assert.Equal(t, []string{"codex", "coderabbit"}, reviewQueueReviewers("both"))
}

func TestReviewqueue_ReviewRepo_Good_CodexBypassesCodeRabbitRateLimit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	ratePath := filepath.Join(home, ".core", "coderabbit-ratelimit.json")
	fs.EnsureDir(filepath.Dir(ratePath))
	fs.Write(ratePath, core.JSONMarshalString(&RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(time.Hour),
		Message: "rate limited",
	}))

	binDir := t.TempDir()
	scriptPath := filepath.Join(binDir, "codex")
	require.NoError(t, os.WriteFile(scriptPath, []byte("#!/bin/sh\necho 'No findings'\n"), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.reviewRepo(context.Background(), t.TempDir(), "repo", "codex", true, true)

	assert.Equal(t, "clean", result.Verdict)
	assert.Equal(t, "skipped (dry run)", result.Action)
}
