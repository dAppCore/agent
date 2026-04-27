// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineEpic_Good_CreateGroupsIssuesIntoEpic(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[10] = &pipelineTestIssue{Number: 10, Title: "security(go-io): Validate tokens", State: "open", Labels: []string{"agentic", "security"}}
	repo.Issues[11] = &pipelineTestIssue{Number: 11, Title: "security(go-io): Sanitize input", State: "open", Labels: []string{"agentic", "security"}}
	repo.Issues[12] = &pipelineTestIssue{Number: 12, Title: "security(go-io): Add rate limiting", State: "open", Labels: []string{"agentic", "security"}}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineEpicCreate(context.Background(), PipelineEpicCreateInput{Org: "core", Repo: "go-io"})

	require.NoError(t, err)
	require.Len(t, output.Epics, 1)
	assert.Equal(t, 13, output.Epics[0].Number)
	assert.Equal(t, "epic/13-security", output.Epics[0].Branch)
	assert.Contains(t, repo.Issues[13].Body, "- [ ] #10 security(go-io): Validate tokens")
	assert.Contains(t, repo.Comments[10][0], "Parent: #13")
	assert.Equal(t, "deadbeef", repo.Branches["epic/13-security"])
}

func TestPipelineEpic_Bad_RunRequiresRepoAndNumber(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineEpicRun(core.NewOptions())

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "repo and epic number are required")
}

func TestPipelineEpic_Ugly_SyncMarksClosedChildrenChecked(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[10] = &pipelineTestIssue{Number: 10, Title: "security(go-io): Validate tokens", State: "closed", Labels: []string{"agentic", "security"}}
	repo.Issues[11] = &pipelineTestIssue{Number: 11, Title: "security(go-io): Sanitize input", State: "open", Labels: []string{"agentic", "security"}}
	repo.Issues[20] = &pipelineTestIssue{
		Number: 20,
		Title:  "epic(go-io): security pipeline",
		State:  "open",
		Labels: []string{"agentic", "epic", "security"},
		Body:   "## Overview\n\nEpic branch: `epic/20-security`\n\n## Child Issues\n\n- [ ] #10 security(go-io): Validate tokens\n- [ ] #11 security(go-io): Sanitize input\n",
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineEpicSync(context.Background(), "core", "go-io", 20, false)

	require.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Equal(t, 1, output.Checked)
	assert.Contains(t, repo.Issues[20].Body, "- [x] #10 security(go-io): Validate tokens")
	assert.Contains(t, repo.Issues[20].Body, "- [ ] #11 security(go-io): Sanitize input")
}

func TestPipelineEpic_Run_Good_DryRunDispatchesUncheckedChildren(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[10] = &pipelineTestIssue{Number: 10, Title: "security(go-io): Validate tokens", State: "open", Labels: []string{"agentic", "security"}}
	repo.Issues[11] = &pipelineTestIssue{Number: 11, Title: "security(go-io): Sanitize input", State: "open", Labels: []string{"agentic", "security"}}
	repo.Issues[12] = &pipelineTestIssue{Number: 12, Title: "security(go-io): Add rate limiting", State: "open", Labels: []string{"agentic", "security"}}
	repo.Issues[20] = &pipelineTestIssue{
		Number: 20,
		Title:  "epic(go-io): security pipeline",
		State:  "open",
		Labels: []string{"agentic", "epic", "security"},
		Body:   "## Overview\n\nEpic branch: `epic/20-security`\n\n## Child Issues\n\n- [ ] #10 security(go-io): Validate tokens\n- [ ] #11 security(go-io): Sanitize input\n- [ ] #12 security(go-io): Add rate limiting\n",
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineEpicRun(context.Background(), PipelineEpicRunInput{
		Org:        "core",
		Repo:       "go-io",
		EpicNumber: 20,
		DryRun:     true,
	})

	require.NoError(t, err)
	assert.Len(t, output.Dispatched, 3)
	assert.Equal(t, "epic/20-security", output.Branch)
}
