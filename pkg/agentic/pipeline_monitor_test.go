// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"net/http"
	"net/http/httptest"
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

	core.RequireNoError(t, err)
	core.AssertLen(t, output.Actions, 3)
	core.AssertContains(t, repo.Comments[1][0], "Can you fix the merge conflict?")
	core.AssertContains(t, repo.Comments[2][0], "Can you fix the code reviews?")
	core.AssertContains(t, repo.Merged, 3)
}

func TestPipelineMonitor_Bad_NoToken(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	s.forgeToken = ""

	_, err := s.pipelineMonitorWithReader(context.Background(), PipelineMonitorInput{Org: "core", Repo: "go-io"}, &pipelineForgeMetaReader{subsystem: s, org: "core"})

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no Forge token configured")
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

	core.RequireNoError(t, err)
	core.AssertEmpty(t, output.Actions)
}

func TestPipelineMonitor_ForgeMetaReader_GetPRMeta_Good(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Pulls[12] = &pipelineTestPR{
		Number:                12,
		Title:                 "Stabilise pipeline",
		State:                 "open",
		Mergeable:             boolPtr(true),
		MergeableState:        "clean",
		HeadRef:               "agent/stabilise-pipeline",
		HeadSHA:               "sha-12",
		BaseRef:               "dev",
		ReviewThreadsTotal:    3,
		ReviewThreadsResolved: 2,
		Statuses: []map[string]any{
			{"context": "qa", "status": "success"},
			{"name": "build", "conclusion": "failure"},
		},
		Reactions: map[string]int{"eyes": 1},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	meta, err := reader.GetPRMeta(context.Background(), "go-io", 12)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 12, meta.Number)
	core.AssertEqual(t, "mergeable", meta.Mergeable)
	core.AssertEqual(t, "agent/stabilise-pipeline", meta.HeadBranch)
	core.AssertLen(t, meta.Checks, 2)
	core.AssertEqual(t, 3, meta.ThreadsTotal)
	core.AssertEqual(t, 2, meta.ThreadsResolved)
	core.AssertTrue(t, meta.HasEyesReaction)
}

func TestPipelineMonitor_ForgeMetaReader_GetPRMeta_Bad(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetPRMeta(context.Background(), "go-io", 44)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read PR")
}

func TestPipelineMonitor_ForgeMetaReader_GetPRMeta_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/core/go-io/pulls/12":
			_, _ = w.Write([]byte(`{"number":12,"state":"open","head":{"ref":"agent/fix","sha":"sha-12","repo":{"updated_at":"2026-04-25T12:00:00Z","pushed_at":"2026-04-25T12:00:00Z"}},"base":{"ref":"dev"},"reactions":{"eyes":0}}`))
		case "/api/v1/repos/core/go-io/commits/sha-12/status":
			_, _ = w.Write([]byte(`not-json`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	meta, err := reader.GetPRMeta(context.Background(), "go-io", 12)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 12, meta.Number)
	core.AssertLen(t, meta.Checks, 0)
}

func TestPipelineMonitor_ForgeMetaReader_GetEpicMeta_Good(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "Epic",
		State:  "open",
		Body:   "Epic branch: `agent/epic`\n- [x] #2 Done child\n- [ ] #3 Open child",
	}
	repo.Issues[2] = &pipelineTestIssue{Number: 2, Title: "Done child", State: "closed"}
	repo.Issues[3] = &pipelineTestIssue{Number: 3, Title: "Open child", State: "open"}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	meta, err := reader.GetEpicMeta(context.Background(), "go-io", 1)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, meta.Number)
	core.AssertEqual(t, "agent/epic", meta.Branch)
	core.AssertLen(t, meta.Children, 2)
	core.AssertTrue(t, meta.Children[0].Checked)
	core.AssertEqual(t, "closed", meta.Children[0].State)
	core.AssertEqual(t, "open", meta.Children[1].State)
}

func TestPipelineMonitor_ForgeMetaReader_GetEpicMeta_Bad(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetEpicMeta(context.Background(), "go-io", 1)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "issue")
}

func TestPipelineMonitor_ForgeMetaReader_GetEpicMeta_Ugly(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{Number: 1, Title: "Epic", State: "open", Body: "plain body"}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	meta, err := reader.GetEpicMeta(context.Background(), "go-io", 1)

	core.RequireNoError(t, err)
	core.AssertEmpty(t, meta.Branch)
	core.AssertLen(t, meta.Children, 0)
}

func TestPipelineMonitor_ForgeMetaReader_GetIssueState_Good(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[7] = &pipelineTestIssue{
		Number: 7,
		Title:  "Fix the flaky tests",
		Body:   "Body",
		State:  "closed",
		Labels: []string{"bug", "agentic"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	state, err := reader.GetIssueState(context.Background(), "go-io", 7)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 7, state.Number)
	core.AssertEqual(t, "closed", state.State)
	core.AssertEqual(t, "Fix the flaky tests", state.Title)
	core.AssertContains(t, state.Labels, "bug")
}

func TestPipelineMonitor_ForgeMetaReader_GetIssueState_Bad(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetIssueState(context.Background(), "go-io", 77)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "issue")
}

func TestPipelineMonitor_ForgeMetaReader_GetIssueState_Ugly(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[8] = &pipelineTestIssue{Number: 8, Title: "Untitled", State: ""}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	state, err := reader.GetIssueState(context.Background(), "go-io", 8)

	core.RequireNoError(t, err)
	core.AssertEqual(t, "open", state.State)
	core.AssertEqual(t, "Untitled", state.Title)
}

func TestPipelineMonitor_ForgeMetaReader_GetCommentReactions_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/api/v1/repos/core/go-io/issues/comments/55/reactions", r.URL.Path)
		core.AssertEqual(t, "token test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`[{"content":"eyes"},{"content":"eyes"},{"content":"rocket"}]`))
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	reactions, err := reader.GetCommentReactions(context.Background(), "go-io", 55)

	core.RequireNoError(t, err)
	core.AssertLen(t, reactions, 2)

	counts := map[string]int{}
	for _, reaction := range reactions {
		counts[reaction.Content] = reaction.Count
	}
	core.AssertEqual(t, 2, counts["eyes"])
	core.AssertEqual(t, 1, counts["rocket"])
}

func TestPipelineMonitor_ForgeMetaReader_GetCommentReactions_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetCommentReactions(context.Background(), "go-io", 55)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read comment reactions")
}

func TestPipelineMonitor_ForgeMetaReader_GetCommentReactions_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetCommentReactions(context.Background(), "go-io", 55)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to decode reactions")
}
