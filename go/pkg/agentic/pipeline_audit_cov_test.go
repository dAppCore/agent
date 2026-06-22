// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipelineAuditCov_Findings_Good_BulletAndNumberedLines — bullet (-,*) and
// numbered (1.) list lines are each extracted as a finding; heading lines (#)
// and blank lines are skipped.
func TestPipelineAuditCov_Findings_Good_BulletAndNumberedLines(t *testing.T) {
	issue := pipelineIssueRecord{
		Title: "[Audit] Security",
		Body:  "# Heading\n\n- First finding\n* Second finding\n1. Third finding\n\n",
	}

	findings := pipelineAuditFindings(issue)

	core.AssertEqual(t, []string{"First finding", "Second finding", "Third finding"}, findings)
}

// TestPipelineAuditCov_Findings_Ugly_ParagraphFallback — when the body carries
// no list markers, the first non-title paragraph becomes the single finding
// (the paragraph-fallback branch).
func TestPipelineAuditCov_Findings_Ugly_ParagraphFallback(t *testing.T) {
	issue := pipelineIssueRecord{
		Title: "Token handling is unsafe",
		Body:  "Token handling is unsafe\n\nThe parser trusts the caller-supplied length without bounds checking.",
	}

	findings := pipelineAuditFindings(issue)

	core.AssertLen(t, findings, 1)
	core.AssertEqual(t, "The parser trusts the caller-supplied length without bounds checking.", findings[0])
}

// TestPipelineAuditCov_Findings_Bad_EmptyBody — an empty body yields no
// findings at all (neither list nor paragraph branch matches).
func TestPipelineAuditCov_Findings_Bad_EmptyBody(t *testing.T) {
	core.AssertEmpty(t, pipelineAuditFindings(pipelineIssueRecord{Title: "Nothing", Body: ""}))
}

// TestPipelineAuditCov_FindingSummary_Good_StripsBackticksAndCollapsesSpace —
// backticks are removed and runs of whitespace collapse to a single space.
func TestPipelineAuditCov_FindingSummary_Good_StripsBackticksAndCollapsesSpace(t *testing.T) {
	core.AssertEqual(t, "use the Fs primitive", pipelineFindingSummary("  use   the `Fs`\tprimitive  "))
}

// TestPipelineAuditCov_FindingSummary_Bad_Empty — a whitespace-only value
// summarises to the empty string.
func TestPipelineAuditCov_FindingSummary_Bad_Empty(t *testing.T) {
	core.AssertEqual(t, "", pipelineFindingSummary("   \t  "))
}

// TestPipelineAuditCov_FindingSummary_Ugly_TruncatesLongValue — a value longer
// than 96 runes is truncated to 93 chars plus an ellipsis.
func TestPipelineAuditCov_FindingSummary_Ugly_TruncatesLongValue(t *testing.T) {
	long := repeatString("a", 200)

	summary := pipelineFindingSummary(long)

	core.AssertLen(t, summary, 96)
	core.AssertEqual(t, repeatString("a", 93)+"...", summary)
}

// TestPipelineAuditCov_CmdAudit_Good_PrintsSummaryAndCreatedIssues — the audit
// command wrapper prints the repo/created summary and returns the typed output.
// HTTP-only path (no subprocess), so captureStdout is safe.
func TestPipelineAuditCov_CmdAudit_Good_PrintsSummaryAndCreatedIssues(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "[Audit] Security",
		Body:   "- Validate tokens\n- Sanitize input",
		State:  "open",
		Labels: []string{"audit", "security"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineAudit(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineAuditOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, typed.Created, 2)
	core.AssertContains(t, output, "repo:     core/go-io")
	core.AssertContains(t, output, "created:  2")
	core.AssertContains(t, output, "created:  #")
}

// TestPipelineAuditCov_CmdAudit_Good_DryRunNoCreatedFooter — a dry-run over a
// repo with no audit issues prints the "no audit issues" footer.
func TestPipelineAuditCov_CmdAudit_Good_DryRunNoCreatedFooter(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineAudit(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "dry-run", Value: "true"},
		))
	})

	core.RequireTrue(t, result.OK)
	core.AssertContains(t, output, "no audit issues")
}
