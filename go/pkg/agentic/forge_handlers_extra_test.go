// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

func newForgeMockSubsystem(t *testing.T, hits *int) *PrepSubsystem {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*hits++
		_, _ = w.Write([]byte("[]"))
	}))
	t.Cleanup(srv.Close)
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

// TestForge_ListCommands_ReachForge — the issue/PR list commands build their
// request and call the forge (mock records the hits).
func TestForge_ListCommands_ReachForge(t *testing.T) {
	hits := 0
	s := newForgeMockSubsystem(t, &hits)
	repo := func(k, v string) core.Options {
		return core.NewOptions(core.Option{Key: "_arg", Value: "test-repo"}, core.Option{Key: "org", Value: "core"})
	}
	captureStdout(t, func() {
		s.cmdIssueList(repo("", ""))
		s.cmdPRList(repo("", ""))
	})
	core.AssertTrue(t, hits > 0)
}

// TestForge_GetMergeCommands_Exercised — issue/PR get + create + merge + close
// commands run their request-building paths against the mock forge.
func TestForge_GetMergeCommands_Exercised(t *testing.T) {
	hits := 0
	s := newForgeMockSubsystem(t, &hits)
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "test-repo"},
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "number", Value: "12"},
		core.Option{Key: "issue", Value: "12"},
		core.Option{Key: "title", Value: "x"},
		core.Option{Key: "branch", Value: "agent/x"},
	)
	captureStdout(t, func() {
		s.cmdIssueGet(opts)
		s.cmdIssueCreate(opts)
		s.cmdPRGet(opts)
		s.cmdPRMerge(opts)
		s.cmdPRClose(opts)
	})
	core.AssertTrue(t, hits > 0)
}

// TestForge_BranchDelete_Exercised — branch delete reaches the forge with a
// valid repo + branch.
func TestForge_BranchDelete_Exercised(t *testing.T) {
	hits := 0
	s := newForgeMockSubsystem(t, &hits)
	captureStdout(t, func() {
		s.cmdBranchDelete(core.NewOptions(
			core.Option{Key: "_arg", Value: "test-repo"},
			core.Option{Key: "org", Value: "core"},
			core.Option{Key: "branch", Value: "agent/x"},
		))
	})
	core.AssertTrue(t, hits > 0)
}
