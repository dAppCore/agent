// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipelineAudit_pipelineIssueIsAudit_GoodBad — the [Audit]/audit: title
// markers flag an audit issue; an ordinary title does not.
func TestPipelineAudit_pipelineIssueIsAudit_GoodBad(t *testing.T) {
	core.AssertTrue(t, pipelineIssueIsAudit(pipelineIssueRecord{Title: "[Audit] flaky tests"}))
	core.AssertTrue(t, pipelineIssueIsAudit(pipelineIssueRecord{Title: "audit: sweep deps"}))
	core.AssertFalse(t, pipelineIssueIsAudit(pipelineIssueRecord{Title: "fix the parser"}))
}

// TestPipelineAudit_pipelineIssueIsEpic_Bad_PlainIssue — an unlabelled issue is
// not an epic (the signal is structural, not title-based).
func TestPipelineAudit_pipelineIssueIsEpic_Bad_PlainIssue(t *testing.T) {
	core.AssertFalse(t, pipelineIssueIsEpic(pipelineIssueRecord{Title: "just an issue"}))
}
