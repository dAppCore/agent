// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- countFindings (extended beyond paths_test.go) ---

func TestReviewQueue_CountFindings_Good_BulletFindings(t *testing.T) {
	output := `Review:
- Missing error check in handler.go:42
- Unused import in config.go
* Race condition in worker pool`
	assert.Equal(t, 3, countFindings(output))
}

func TestReviewQueue_CountFindings_Good_IssueKeyword(t *testing.T) {
	output := `Line 10: Issue: variable shadowing
Line 25: Finding: unchecked return value`
	assert.Equal(t, 2, countFindings(output))
}

func TestReviewQueue_CountFindings_Good_DefaultOneIfNotClean(t *testing.T) {
	output := "Some output without markers but also not explicitly clean"
	assert.Equal(t, 1, countFindings(output))
}

func TestReviewQueue_CountFindings_Good_MixedContent(t *testing.T) {
	output := `Summary of review:
The code is generally well structured.
- Missing nil check
Some commentary here
* Redundant allocation`
	assert.Equal(t, 2, countFindings(output))
}

// --- parseRetryAfter (extended) ---

func TestReviewQueue_ParseRetryAfter_Good_SingleMinuteAndSeconds(t *testing.T) {
	d := parseRetryAfter("try after 1 minute and 30 seconds")
	assert.Equal(t, 1*time.Minute+30*time.Second, d)
}

func TestReviewQueue_ParseRetryAfter_Bad_EmptyMessage(t *testing.T) {
	d := parseRetryAfter("")
	assert.Equal(t, 5*time.Minute, d)
}
