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

// mockScanServer creates a server that handles repo listing and issue listing.
func mockScanServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// List org repos
	mux.HandleFunc("/api/v1/orgs/core/repos", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{"name": "go-io", "full_name": "core/go-io"},
			{"name": "go-log", "full_name": "core/go-log"},
			{"name": "agent", "full_name": "core/agent"},
		})))
	})

	// List issues for repos
	mux.HandleFunc("/api/v1/repos/core/go-io/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{
				"number":   10,
				"title":    "Replace fmt.Errorf with E()",
				"labels":   []map[string]any{{"name": "agentic"}},
				"assignee": nil,
				"html_url": "https://forge.lthn.ai/core/go-io/issues/10",
			},
			{
				"number":   11,
				"title":    "Add missing tests",
				"labels":   []map[string]any{{"name": "agentic"}, {"name": "help-wanted"}},
				"assignee": map[string]any{"login": "virgil"},
				"html_url": "https://forge.lthn.ai/core/go-io/issues/11",
			},
		})))
	})

	mux.HandleFunc("/api/v1/repos/core/go-log/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{
				"number":   5,
				"title":    "Fix log rotation",
				"labels":   []map[string]any{{"name": "bug"}},
				"assignee": nil,
				"html_url": "https://forge.lthn.ai/core/go-log/issues/5",
			},
		})))
	})

	mux.HandleFunc("/api/v1/repos/core/agent/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// --- scan ---

func TestScan_Scan_Good_Case(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{Org: "core"})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertGreater(t, out.Count, 0)
	// Verify issues contain repos from mock server
	repos := make(map[string]bool)
	for _, iss := range out.Issues {
		repos[iss.Repo] = true
	}
	core.AssertTrue(t, repos["go-io"] || repos["go-log"], "should contain issues from mock repos")
}

func TestScan_AllRepos_Good_Case(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertGreater(t, out.Count, 0)
}

func TestScan_WithLimit_Good_Case(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{Limit: 1})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertLessOrEqual(t, out.Count, 1)
}

func TestScan_DefaultLabels_Good_Case(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Default labels: agentic, help-wanted, bug
	_, out, err := s.scan(context.Background(), nil, ScanInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
}

func TestScan_CustomLabels_Good_Case(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{
		Labels: []string{"bug"},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
}

func TestScan_Deduplicates_Good_Case(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Two labels that return the same issues — should be deduped
	_, out, err := s.scan(context.Background(), nil, ScanInput{
		Labels: []string{"agentic", "help-wanted"},
		Limit:  50,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)

	// Check no duplicates (same repo+number)
	seen := make(map[string]bool)
	for _, issue := range out.Issues {
		key := issue.Repo + "#" + itoa(issue.Number)
		core.AssertFalse(t, seen[key], "duplicate issue: %s", key)
		seen[key] = true
	}
}

func TestScan_NoToken_Bad_Case(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.scan(context.Background(), nil, ScanInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no Forge token")
}

// --- listRepoIssues ---

func TestScan_ListRepoIssues_Good_ReturnsIssues(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	core.RequireNoError(t, err)
	core.AssertLen(t, issues, 2)
	core.AssertEqual(t, "go-io", issues[0].Repo)
	core.AssertEqual(t, 10, issues[0].Number)
	core.AssertContains(t, issues[0].Labels, "agentic")
}

func TestScan_ListRepoIssues_Good_EmptyResult(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "agent", "agentic")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, issues)
}

func TestScan_ListRepoIssues_Good_AssigneeExtracted(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	core.RequireNoError(t, err)
	core.AssertLen(t, issues, 2)
	core.AssertEqual(t, "", issues[0].Assignee)
	core.AssertEqual(t, "virgil", issues[1].Assignee)
}

func TestScan_ListRepoIssues_Good_EncodesLabelQuery(t *testing.T) {
	expectedLabel := "bug+urgent & review"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/api/v1/repos/core/go-io/issues", r.URL.Path)
		core.AssertEqual(t, expectedLabel, r.URL.Query().Get("labels"))
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", expectedLabel)
	core.RequireNoError(t, err)
	core.AssertEmpty(t, issues)
}

func TestScan_ListRepoIssues_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	core.AssertError(t, err)
}

func TestScan_ListRepoIssues_Bad_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "parse issues response")
}

// --- scan Bad/Ugly ---

func TestScan_Scan_Bad_Case(t *testing.T) {
	// Forge returns error for org repos
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.scan(context.Background(), nil, ScanInput{})
	core.AssertError(t, err)
}

func TestScan_Scan_Ugly_Case(t *testing.T) {
	// Org with no repos
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if core.Contains(r.URL.Path, "/orgs/") {
			w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 0, out.Count)
}

// --- listOrgRepos Good/Bad/Ugly ---

func TestScan_ListOrgRepos_Good_Case(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	repos, err := s.listOrgRepos(context.Background(), "core")
	core.RequireNoError(t, err)
	core.AssertLen(t, repos, 3)
	core.AssertContains(t, repos, "go-io")
	core.AssertContains(t, repos, "go-log")
	core.AssertContains(t, repos, "agent")
}

func TestScan_ListOrgRepos_Bad_Case(t *testing.T) {
	// Forge returns error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, err := s.listOrgRepos(context.Background(), "core")
	core.AssertError(t, err)
}

func TestScan_ListOrgRepos_Ugly_Case(t *testing.T) {
	// Empty org name
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	repos, err := s.listOrgRepos(context.Background(), "")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, repos)
}

// --- listRepoIssues Ugly ---

func TestScan_ListRepoIssues_Ugly_Case(t *testing.T) {
	// Issues with very long titles
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		longTitle := repeatString("Very Long Issue Title ", 50)
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{
				"number":   1,
				"title":    longTitle,
				"labels":   []map[string]any{{"name": "agentic"}},
				"assignee": nil,
				"html_url": "https://forge.lthn.ai/core/go-io/issues/1",
			},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	core.RequireNoError(t, err)
	core.AssertLen(t, issues, 1)
	core.AssertTrue(t, len(issues[0].Title) > 100)
}

func TestScan_ListRepoIssues_Good_URLRewrite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{
				"number":   1,
				"title":    "Test",
				"labels":   []map[string]any{},
				"assignee": nil,
				"html_url": "https://forge.lthn.ai/core/go-io/issues/1",
			},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "")
	core.RequireNoError(t, err)
	core.AssertLen(t, issues, 1)
	// URL should be rewritten to use the mock server URL
	core.AssertContains(t, issues[0].URL, srv.URL)
}
