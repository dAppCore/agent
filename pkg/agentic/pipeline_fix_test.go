// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineFix_Good_ReviewsPostsComment(t *testing.T) {
	repo := newPipelineTestRepo()
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineFixReviews(context.Background(), PipelineFixInput{Org: "core", Repo: "go-io", Number: 7})

	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Contains(t, repo.Comments[7][0], "Can you fix the code reviews?")
}

func TestPipelineFix_Bad_FormatRequiresWorkspace(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, err := s.pipelineFixFormat(context.Background(), PipelineFixInput{Org: "core", Repo: "go-io", Number: 9})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace or repo_dir is required")
}

func TestPipelineFix_Ugly_ThreadsNoopWhenAlreadyResolved(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Pulls[5] = &pipelineTestPR{
		Number:                5,
		Title:                 "Already clean",
		State:                 "open",
		Mergeable:             boolPtr(true),
		HeadRef:               "agent/clean",
		HeadSHA:               "sha-clean",
		BaseRef:               "dev",
		ReviewThreadsTotal:    2,
		ReviewThreadsResolved: 2,
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineFixThreads(context.Background(), PipelineFixInput{Org: "core", Repo: "go-io", Number: 5})

	require.NoError(t, err)
	assert.Equal(t, "noop", output.Action)
	assert.Equal(t, "no unresolved review threads remain", output.Message)
}

func TestPipelineFix_Format_Good_DryRunCountsFiles(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "one.go"), "package main\nfunc main( ){}\n").OK)
	require.True(t, fs.Write(core.JoinPath(dir, "two.go"), "package main\nfunc helper( ){}\n").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	output, err := s.pipelineFixFormat(context.Background(), PipelineFixInput{
		Org:          "core",
		Repo:         "go-io",
		Number:       12,
		WorkspaceDir: dir,
		DryRun:       true,
	})

	require.NoError(t, err)
	assert.Equal(t, "format", output.Action)
	assert.Equal(t, 2, output.Files)
}
