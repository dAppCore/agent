// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipelineClassifyIssueStructural_Good_StructuralSignals verifies the
// classifier reads epic / audit / PR signals from typed API fields — labels,
// native sub-issue links, and the pull_request field — for the representative
// issue shapes the audit path encounters.
func TestPipelineClassifyIssueStructural_Good_StructuralSignals(t *testing.T) {
	auditByLabel := pipelineClassifyIssueStructural(pipelineIssueRecord{
		Number: 1,
		Title:  "Security review",
		Labels: []pipelineLabelRecord{{Name: "audit"}, {Name: "security"}},
	})
	core.AssertTrue(t, auditByLabel.IsAudit)
	core.AssertFalse(t, auditByLabel.IsEpic)
	core.AssertFalse(t, auditByLabel.IsPR)
	core.AssertEqual(t, []string{"audit", "security"}, auditByLabel.Labels)

	epicByLabel := pipelineClassifyIssueStructural(pipelineIssueRecord{
		Number: 2,
		Title:  "Epic: harden auth",
		Labels: []pipelineLabelRecord{{Name: "agentic"}, {Name: "epic"}},
	})
	core.AssertTrue(t, epicByLabel.IsEpic)
	core.AssertFalse(t, epicByLabel.IsAudit)

	epicByChildren := pipelineClassifyIssueStructural(pipelineIssueRecord{
		Number:    3,
		Title:     "Tracking issue",
		SubIssues: []pipelineSubIssueRecord{{IssueID: 11, State: "open"}, {Number: 12, State: "closed"}},
	})
	core.AssertTrue(t, epicByChildren.IsEpic)

	pullRequest := pipelineClassifyIssueStructural(pipelineIssueRecord{
		Number:      4,
		Title:       "feat: add thing",
		PullRequest: map[string]any{"merged": false},
	})
	core.AssertTrue(t, pullRequest.IsPR)
	core.AssertFalse(t, pullRequest.IsEpic)
}

// TestPipelineClassifyIssueStructural_Bad_BodyChecklistIsNotAnEpic confirms the
// classifier no longer treats a markdown checklist body as an epic signal. An
// issue carrying a `- [ ] #N` checklist but no `epic` label and no structural
// sub-issue links is plain — parity with PHP, which never parses body prose for
// children.
func TestPipelineClassifyIssueStructural_Bad_BodyChecklistIsNotAnEpic(t *testing.T) {
	signal := pipelineClassifyIssueStructural(pipelineIssueRecord{
		Number: 5,
		Title:  "Loose tracking notes",
		Body:   "Plan:\n- [ ] #21 do the first thing\n- [x] #22 did the second thing",
	})

	core.AssertFalse(t, signal.IsEpic)
	core.AssertFalse(t, signal.IsAudit)
	core.AssertFalse(t, signal.IsPR)
}

// TestPipelineClassifyIssueStructural_Ugly_EmptyAndMalformedRecords verifies the
// classifier is total over degenerate inputs: an empty record, blank label
// names, and sub-issue records with no usable identifier all classify cleanly
// without panicking, yielding a non-nil (possibly empty) label slice.
func TestPipelineClassifyIssueStructural_Ugly_EmptyAndMalformedRecords(t *testing.T) {
	empty := pipelineClassifyIssueStructural(pipelineIssueRecord{})
	core.AssertFalse(t, empty.IsAudit)
	core.AssertFalse(t, empty.IsEpic)
	core.AssertFalse(t, empty.IsPR)
	core.AssertEqual(t, 0, len(empty.Labels))

	blankLabels := pipelineClassifyIssueStructural(pipelineIssueRecord{
		Number: 6,
		Labels: []pipelineLabelRecord{{Name: ""}, {Name: "audit"}},
	})
	core.AssertTrue(t, blankLabels.IsAudit)
	core.AssertEqual(t, []string{"audit"}, blankLabels.Labels)

	unusableChildren := pipelineClassifyIssueStructural(pipelineIssueRecord{
		Number:    7,
		SubTasks:  []pipelineSubIssueRecord{{IssueID: 0, Number: 0}},
		SubIssues: []pipelineSubIssueRecord{{IssueID: 0, Number: 0}},
	})
	core.AssertFalse(t, unusableChildren.IsEpic)
}

// TestPipelineIssueStructuralChildren_Good_SubTasksPreferredOverSubIssues mirrors
// PHP ForgejoMetaReader::extractEpicChildren, which reads `subtasks` first and
// falls back to `sub_issues`. The numeric identifier falls back from issue_id to
// number when issue_id is absent.
func TestPipelineIssueStructuralChildren_Good_SubTasksPreferredOverSubIssues(t *testing.T) {
	both := pipelineIssueStructuralChildren(pipelineIssueRecord{
		SubTasks:  []pipelineSubIssueRecord{{IssueID: 31}, {Number: 32}},
		SubIssues: []pipelineSubIssueRecord{{IssueID: 99}},
	})
	core.AssertEqual(t, []int{31, 32}, both)

	subIssuesOnly := pipelineIssueStructuralChildren(pipelineIssueRecord{
		SubIssues: []pipelineSubIssueRecord{{Number: 41}, {IssueID: 42}},
	})
	core.AssertEqual(t, []int{41, 42}, subIssuesOnly)

	none := pipelineIssueStructuralChildren(pipelineIssueRecord{Number: 8})
	core.AssertEqual(t, 0, len(none))
}

// TestPipelineAuditWithReader_Good_StructuralEpicSkippedAndAuditConverted drives
// the audit path through an injected structural reader: an epic issue (epic
// label) is skipped, while an audit issue (audit label) is converted into
// implementation issues and closed — proving the audit loop classifies via the
// MetaReader, not the body.
func TestPipelineAuditWithReader_Good_StructuralEpicSkippedAndAuditConverted(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "Epic: security hardening",
		Body:   "- [ ] #2 something",
		State:  "open",
		Labels: []string{"agentic", "epic"},
	}
	repo.Issues[2] = &pipelineTestIssue{
		Number: 2,
		Title:  "[Audit] Security",
		Body:   "- Validate tokens\n- Sanitize input",
		State:  "open",
		Labels: []string{"audit", "security"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := pipelineAuditWithReader(s, s.commandContext(), PipelineAuditInput{Org: "core", Repo: "go-io"}, newPipelineForgeMetaReader(s, "core"))

	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertLen(t, output.Audits, 1)
	core.AssertEqual(t, 2, output.Audits[0].Number)
	core.AssertLen(t, output.Created, 2)
	core.AssertEqual(t, []int{2}, output.Closed)
	core.AssertEqual(t, "open", repo.Issues[1].State)
}
