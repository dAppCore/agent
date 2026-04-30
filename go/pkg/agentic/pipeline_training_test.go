// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestPipelineTraining_Good_RootHelp(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	output := captureStdout(t, func() {
		result := s.cmdPipelineTraining(core.NewOptions())
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "core-agent pipeline/training/capture")
	core.AssertContains(t, output, "core-agent pipeline/training/stats")
	core.AssertContains(t, output, "core-agent pipeline/training/export")
}

func TestPipelineTraining_Good_CaptureWritesJournal(t *testing.T) {
	srv := newTrainingTestServer(t, trainingTestServerConfig{
		Repo:                  "go-io",
		Number:                7,
		Merged:                true,
		State:                 "closed",
		HeadSHA:               "deadbeef",
		HeadRef:               "agent/fix-tests",
		BaseRef:               "dev",
		ReviewThreadsTotal:    2,
		ReviewThreadsResolved: 2,
		Diff:                  "diff --git a/main.go b/main.go\n+package main\n",
	})
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	output := captureStdout(t, func() {
		result := s.cmdPipelineTrainingCapture(core.NewOptions(
			core.Option{Key: "_arg", Value: "7"},
			core.Option{Key: "repo", Value: "go-io"},
		))
		core.RequireTrue(t, result.OK)
	})

	entries := readPipelineTrainingJournal(pipelineTrainingJournalPath())
	core.AssertLen(t, entries, 1)
	core.AssertContains(t, output, "go-io#7")
	core.AssertEqual(t, 2, entries[0].CodeRabbitFindings)
	core.AssertEqual(t, "forge.pull.diff", entries[0].DiffSource)
	core.AssertContains(t, entries[0].Diff, "diff --git")
}

func TestPipelineTraining_Good_StatsAggregatesJournal(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireNoError(t, appendJSONLRecord(pipelineTrainingJournalPath(), PipelineTrainingEntry{
		CapturedAt:         "2026-04-25T12:00:00Z",
		Repo:               "go-io",
		PRNumber:           7,
		CodeRabbitFindings: 0,
	}))
	core.RequireNoError(t, appendJSONLRecord(pipelineTrainingJournalPath(), PipelineTrainingEntry{
		CapturedAt:         "2026-04-25T12:10:00Z",
		Repo:               "go-log",
		PRNumber:           8,
		CodeRabbitFindings: 3,
	}))

	var stats PipelineTrainingStats
	output := captureStdout(t, func() {
		result := s.cmdPipelineTrainingStats(core.NewOptions())
		core.RequireTrue(t, result.OK)
		var ok bool
		stats, ok = result.Value.(PipelineTrainingStats)
		core.RequireTrue(t, ok)
	})

	core.AssertEqual(t, 2, stats.TotalPRs)
	core.AssertEqual(t, 1, stats.ZeroFindingPRs)
	core.AssertEqual(t, 1, stats.ByRepo["go-io"])
	core.AssertEqual(t, 1, stats.ByRepo["go-log"])
	core.AssertContains(t, output, "go-io")
	core.AssertContains(t, output, "go-log")
}

func TestPipelineTraining_Good_ExportWritesZeroFindingOnly(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireNoError(t, appendJSONLRecord(pipelineTrainingJournalPath(), PipelineTrainingEntry{
		CapturedAt:         "2026-04-25T12:00:00Z",
		Repo:               "go-io",
		PRNumber:           7,
		CodeRabbitFindings: 0,
		Diff:               "clean diff",
	}))
	core.RequireNoError(t, appendJSONLRecord(pipelineTrainingJournalPath(), PipelineTrainingEntry{
		CapturedAt:         "2026-04-25T12:10:00Z",
		Repo:               "go-log",
		PRNumber:           8,
		CodeRabbitFindings: 2,
		Diff:               "noisy diff",
	}))

	result := s.cmdPipelineTrainingExport(core.NewOptions())
	core.RequireTrue(t, result.OK)

	exported := readPipelineTrainingJournal(pipelineTrainingExportPath())
	core.AssertLen(t, exported, 1)
	core.AssertEqual(t, "go-io", exported[0].Repo)
	core.AssertEqual(t, 0, exported[0].CodeRabbitFindings)
}

func TestPipelineTraining_Bad_CaptureRequiresRepoAndNumber(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineTrainingCapture(core.NewOptions())

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "repo and pull request number are required")
}

func TestPipelineTraining_Bad_CaptureRejectsOpenPullRequest(t *testing.T) {
	srv := newTrainingTestServer(t, trainingTestServerConfig{
		Repo:    "go-io",
		Number:  7,
		Merged:  false,
		State:   "open",
		HeadSHA: "deadbeef",
		HeadRef: "agent/fix-tests",
		BaseRef: "dev",
		Diff:    "diff --git a/main.go b/main.go\n",
	})
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	result := s.cmdPipelineTrainingCapture(core.NewOptions(
		core.Option{Key: "_arg", Value: "7"},
		core.Option{Key: "repo", Value: "go-io"},
	))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "pull request is not merged")
}

func TestPipelineTraining_Ugly_CaptureFallsBackToGitShow(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := testCore
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		codePath:       t.TempDir(),
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	repoDir, headSHA := createTrainingGitRepo(t, c, s.codePath)
	core.AssertNotEmpty(t, repoDir)

	srv := newTrainingTestServer(t, trainingTestServerConfig{
		Repo:                  "go-io",
		Number:                9,
		Merged:                true,
		State:                 "closed",
		HeadSHA:               headSHA,
		HeadRef:               "agent/feature",
		BaseRef:               "dev",
		ReviewThreadsTotal:    0,
		ReviewThreadsResolved: 0,
		DiffStatus:            http.StatusNotFound,
	})
	t.Cleanup(srv.Close)
	s.forgeURL = srv.URL

	output, err := s.pipelineTrainingCapture(context.Background(), PipelineTrainingCaptureInput{
		Org:    "core",
		Repo:   "go-io",
		Number: 9,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "git.show", output.DiffSource)
}

func TestPipelineTraining_Ugly_StatsSkipsCorruptJournalRows(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.RequireNoError(t, ensureParentDir(pipelineTrainingJournalPath()))
	core.RequireTrue(t, fs.WriteAtomic(pipelineTrainingJournalPath(), core.Concat(
		"{not-json}\n",
		core.JSONMarshalString(PipelineTrainingEntry{CapturedAt: "2026-04-25T12:00:00Z", Repo: "go-io", PRNumber: 7, CodeRabbitFindings: 0}),
		"\n",
	)).OK)

	result := s.cmdPipelineTrainingStats(core.NewOptions())
	core.RequireTrue(t, result.OK)
	stats, ok := result.Value.(PipelineTrainingStats)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, stats.TotalPRs)
}

type trainingTestServerConfig struct {
	Repo                  string
	Number                int
	Merged                bool
	State                 string
	HeadSHA               string
	HeadRef               string
	BaseRef               string
	ReviewThreadsTotal    int
	ReviewThreadsResolved int
	Diff                  string
	DiffStatus            int
}

func newTrainingTestServer(t *testing.T, config trainingTestServerConfig) *httptest.Server {
	t.Helper()
	if config.DiffStatus == 0 {
		config.DiffStatus = http.StatusOK
	}
	if config.State == "" {
		config.State = "closed"
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case core.Sprintf("/api/v1/repos/core/%s/pulls/%d", config.Repo, config.Number):
			payload := map[string]any{
				"number":                  config.Number,
				"title":                   "Test PR",
				"state":                   config.State,
				"html_url":                core.Sprintf("https://forge.test/core/%s/pulls/%d", config.Repo, config.Number),
				"merged":                  config.Merged,
				"mergeable":               true,
				"mergeable_state":         "clean",
				"review_threads_total":    config.ReviewThreadsTotal,
				"review_threads_resolved": config.ReviewThreadsResolved,
				"review_comments":         config.ReviewThreadsTotal,
				"head": map[string]any{
					"ref": config.HeadRef,
					"sha": config.HeadSHA,
					"repo": map[string]any{
						"updated_at": "2026-04-25T12:00:00Z",
						"pushed_at":  "2026-04-25T12:00:00Z",
					},
				},
				"base": map[string]any{
					"ref": config.BaseRef,
				},
			}
			_, _ = w.Write([]byte(core.JSONMarshalString(payload)))
		case core.Sprintf("/api/v1/repos/core/%s/commits/%s/status", config.Repo, config.HeadSHA):
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"statuses": []map[string]any{
					{"context": "qa", "state": "success"},
				},
			})))
		case core.Sprintf("/api/v1/repos/core/%s/pulls/%d.diff", config.Repo, config.Number):
			w.WriteHeader(config.DiffStatus)
			if config.DiffStatus == http.StatusOK {
				_, _ = w.Write([]byte(config.Diff))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func createTrainingGitRepo(t *testing.T, c *core.Core, codePath string) (string, string) {
	t.Helper()
	repoDir := core.JoinPath(codePath, "core", "go-io")
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	core.RequireTrue(t, c.Process().Run(context.Background(), "git", "init", "-b", "dev", repoDir).OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@example.com").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(repoDir, "main.go"), "package main\n\nfunc main() {}\n").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "init").OK)

	head := c.Process().RunIn(context.Background(), repoDir, "git", "rev-parse", "HEAD")
	core.RequireTrue(t, head.OK)
	return repoDir, core.Trim(resultText(head))
}
