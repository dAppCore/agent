// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
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

	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertLen(t, output.Audit.Created, 3)
	core.AssertLen(t, output.Runs, 1)
	core.AssertLen(t, output.Runs[0].Dispatched, 3)
	core.AssertEmpty(t, output.Direct)
}

func TestPipelineOnboard_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineOnboard(core.NewOptions())

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "repo is required")
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

	core.RequireNoError(t, err)
	core.AssertEmpty(t, output.Runs)
	core.AssertLen(t, output.Direct, 2)
}
