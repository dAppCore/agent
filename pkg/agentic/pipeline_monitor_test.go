// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineMonitor_Good_AutoIntervenesAndMerges(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Pulls[1] = &pipelineTestPR{
		Number:         1,
		Title:          "Fix conflicts",
		State:          "open",
		Mergeable:      boolPtr(false),
		HeadRef:        "agent/conflicts",
		HeadSHA:        "sha-conflicts",
		BaseRef:        "dev",
		MergeableState: "dirty",
	}
	repo.Pulls[2] = &pipelineTestPR{
		Number:                  2,
		Title:                   "Fix reviews",
		State:                   "open",
		Mergeable:               boolPtr(true),
		HeadRef:                 "agent/reviews",
		HeadSHA:                 "sha-reviews",
		BaseRef:                 "dev",
		ReviewThreadsTotal:      3,
		ReviewThreadsResolved:   1,
		ReviewThreadsUnresolved: 2,
		Statuses: []map[string]any{
			{"context": "qa", "status": "success"},
		},
	}
	repo.Pulls[3] = &pipelineTestPR{
		Number:                3,
		Title:                 "Ready to merge",
		State:                 "open",
		Mergeable:             boolPtr(true),
		HeadRef:               "agent/merge",
		HeadSHA:               "sha-merge",
		BaseRef:               "dev",
		ReviewThreadsTotal:    1,
		ReviewThreadsResolved: 1,
		Statuses: []map[string]any{
			{"context": "qa", "status": "success"},
			{"context": "build", "status": "success"},
		},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineMonitorWithReader(context.Background(), PipelineMonitorInput{
		Org:  "core",
		Repo: "go-io",
	}, &pipelineForgeMetaReader{subsystem: s, org: "core"})

	require.NoError(t, err)
	assert.Len(t, output.Actions, 3)
	assert.Contains(t, repo.Comments[1][0], "Can you fix the merge conflict?")
	assert.Contains(t, repo.Comments[2][0], "Can you fix the code reviews?")
	assert.Contains(t, repo.Merged, 3)
}

func TestPipelineMonitor_Bad_NoToken(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	s.forgeToken = ""

	_, err := s.pipelineMonitorWithReader(context.Background(), PipelineMonitorInput{Org: "core", Repo: "go-io"}, &pipelineForgeMetaReader{subsystem: s, org: "core"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Forge token configured")
}

func TestPipelineMonitor_Ugly_NoActionWhenChecksPending(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Pulls[4] = &pipelineTestPR{
		Number:    4,
		Title:     "Waiting for CI",
		State:     "open",
		Mergeable: boolPtr(true),
		HeadRef:   "agent/pending",
		HeadSHA:   "sha-pending",
		BaseRef:   "dev",
		Statuses:  []map[string]any{{"context": "qa", "status": "pending"}},
		Reactions: map[string]int{},
		Comments:  0,
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	s, _ := testPrepWithCore(t, srv)
	output, err := s.pipelineMonitorWithReader(context.Background(), PipelineMonitorInput{
		Org:  "core",
		Repo: "go-io",
	}, &pipelineForgeMetaReader{subsystem: s, org: "core"})

	require.NoError(t, err)
	assert.Empty(t, output.Actions)
}
