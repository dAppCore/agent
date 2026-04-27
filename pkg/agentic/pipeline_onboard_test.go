// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineOnboard_Good_ChainsAuditEpicAndDispatch(t *testing.T) {
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
	output, err := s.pipelineOnboard(context.Background(), PipelineOnboardInput{
		Org:            "core",
		Repo:           "go-io",
		DispatchDryRun: true,
	})

	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Len(t, output.Audit.Created, 3)
	require.Len(t, output.Runs, 1)
	assert.Len(t, output.Runs[0].Dispatched, 3)
	assert.Empty(t, output.Direct)
}

func TestPipelineOnboard_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineOnboard(core.NewOptions())

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestPipelineOnboard_Ugly_DirectDispatchWhenEpicNotCreated(t *testing.T) {
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
	output, err := s.pipelineOnboard(context.Background(), PipelineOnboardInput{
		Org:            "core",
		Repo:           "go-io",
		DispatchDryRun: true,
	})

	require.NoError(t, err)
	assert.Empty(t, output.Runs)
	assert.Len(t, output.Direct, 2)
}
