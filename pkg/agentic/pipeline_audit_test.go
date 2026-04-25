// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Len(t, output.Audits, 1)
	assert.Len(t, output.Created, 3)
	assert.Equal(t, []int{1}, output.Closed)
	assert.Equal(t, "closed", repo.Issues[1].State)
	assert.Len(t, repo.Comments[1], 1)
	assert.Contains(t, repo.Comments[1][0], "Implementation issues created")
}

func TestPipelineAudit_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineAudit(core.NewOptions())

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "repo is required")
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

	require.NoError(t, err)
	assert.Len(t, output.Existing, 1)
	assert.Len(t, output.Created, 1)
	assert.Equal(t, 2, output.Existing[0].Number)
}
