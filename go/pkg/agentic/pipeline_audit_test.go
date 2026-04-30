// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestPipelineAudit_Good_CreatesImplementationIssuesAndClosesAudit(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "[Audit] Security",
		Body:   "- Validate tokens\n- Sanitize input\n- Add rate limiting",
		State:  "open",
		Labels: []string{"audit", "security"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineAudit(context.Background(), PipelineAuditInput{Org: "core", Repo: "go-io"})

	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertLen(t, output.Audits, 1)
	core.AssertLen(t, output.Created, 3)
	core.AssertEqual(t, []int{1}, output.Closed)
	core.AssertEqual(t, "closed", repo.Issues[1].State)
	core.AssertLen(t, repo.Comments[1], 1)
	core.AssertContains(t, repo.Comments[1][0], "Implementation issues created")
}

func TestPipelineAudit_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineAudit(core.NewOptions())

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "repo is required")
}

func TestPipelineAudit_Ugly_DeduplicatesExistingImplementationIssue(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "[Audit] Security",
		Body:   "- Validate tokens\n- Sanitize input",
		State:  "open",
		Labels: []string{"audit", "security"},
	}
	repo.Issues[2] = &pipelineTestIssue{
		Number: 2,
		Title:  "security(go-io): Validate tokens",
		Body:   "Parent audit: #1",
		State:  "open",
		Labels: []string{"agentic", "security"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineAudit(context.Background(), PipelineAuditInput{Org: "core", Repo: "go-io"})

	core.RequireNoError(t, err)
	core.AssertLen(t, output.Existing, 1)
	core.AssertLen(t, output.Created, 1)
	core.AssertEqual(t, 2, output.Existing[0].Number)
}
