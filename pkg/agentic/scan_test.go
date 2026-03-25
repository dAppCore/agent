// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockScanServer creates a server that handles repo listing and issue listing.
func mockScanServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// List org repos
	mux.HandleFunc("/api/v1/orgs/core/repos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "go-io", "full_name": "core/go-io"},
			{"name": "go-log", "full_name": "core/go-log"},
			{"name": "agent", "full_name": "core/agent"},
		})
	})

	// List issues for repos
	mux.HandleFunc("/api/v1/repos/core/go-io/issues", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
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
		})
	})

	mux.HandleFunc("/api/v1/repos/core/go-log/issues", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"number":   5,
				"title":    "Fix log rotation",
				"labels":   []map[string]any{{"name": "bug"}},
				"assignee": nil,
				"html_url": "https://forge.lthn.ai/core/go-log/issues/5",
			},
		})
	})

	mux.HandleFunc("/api/v1/repos/core/agent/issues", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// --- scan ---

func TestScan_Good_AllRepos(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Greater(t, out.Count, 0)
}

func TestScan_Good_WithLimit(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{Limit: 1})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.LessOrEqual(t, out.Count, 1)
}

func TestScan_Good_DefaultLabels(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Default labels: agentic, help-wanted, bug
	_, out, err := s.scan(context.Background(), nil, ScanInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
}

func TestScan_Good_CustomLabels(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{
		Labels: []string{"bug"},
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
}

func TestScan_Good_Deduplicates(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Two labels that return the same issues — should be deduped
	_, out, err := s.scan(context.Background(), nil, ScanInput{
		Labels: []string{"agentic", "help-wanted"},
		Limit:  50,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)

	// Check no duplicates (same repo+number)
	seen := make(map[string]bool)
	for _, issue := range out.Issues {
		key := issue.Repo + "#" + itoa(issue.Number)
		assert.False(t, seen[key], "duplicate issue: %s", key)
		seen[key] = true
	}
}

func TestScan_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		forgeToken: "",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.scan(context.Background(), nil, ScanInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Forge token")
}

// --- listRepoIssues ---

func TestScan_ListRepoIssues_Good_ReturnsIssues(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, "go-io", issues[0].Repo)
	assert.Equal(t, 10, issues[0].Number)
	assert.Contains(t, issues[0].Labels, "agentic")
}

func TestScan_ListRepoIssues_Good_EmptyResult(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "agent", "agentic")
	require.NoError(t, err)
	assert.Empty(t, issues)
}

func TestScan_ListRepoIssues_Good_AssigneeExtracted(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	require.NoError(t, err)
	require.Len(t, issues, 2)
	assert.Equal(t, "", issues[0].Assignee)
	assert.Equal(t, "virgil", issues[1].Assignee)
}

func TestScan_ListRepoIssues_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	assert.Error(t, err)
}

// --- scan Bad/Ugly ---

func TestScan_Scan_Bad(t *testing.T) {
	// Forge returns error for org repos
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.scan(context.Background(), nil, ScanInput{})
	assert.Error(t, err)
}

func TestScan_Scan_Ugly(t *testing.T) {
	// Org with no repos
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/orgs/") {
			json.NewEncoder(w).Encode([]map[string]any{})
			return
		}
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.scan(context.Background(), nil, ScanInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 0, out.Count)
}

// --- listOrgRepos Good/Bad/Ugly ---

func TestScan_ListOrgRepos_Good(t *testing.T) {
	srv := mockScanServer(t)
	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	repos, err := s.listOrgRepos(context.Background(), "core")
	require.NoError(t, err)
	assert.Len(t, repos, 3)
	assert.Contains(t, repos, "go-io")
	assert.Contains(t, repos, "go-log")
	assert.Contains(t, repos, "agent")
}

func TestScan_ListOrgRepos_Bad(t *testing.T) {
	// Forge returns error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, err := s.listOrgRepos(context.Background(), "core")
	assert.Error(t, err)
}

func TestScan_ListOrgRepos_Ugly(t *testing.T) {
	// Empty org name
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	repos, err := s.listOrgRepos(context.Background(), "")
	require.NoError(t, err)
	assert.Empty(t, repos)
}

// --- listRepoIssues Ugly ---

func TestScan_ListRepoIssues_Ugly(t *testing.T) {
	// Issues with very long titles
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		longTitle := strings.Repeat("Very Long Issue Title ", 50)
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"number":   1,
				"title":    longTitle,
				"labels":   []map[string]any{{"name": "agentic"}},
				"assignee": nil,
				"html_url": "https://forge.lthn.ai/core/go-io/issues/1",
			},
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "agentic")
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.True(t, len(issues[0].Title) > 100)
}

func TestScan_ListRepoIssues_Good_URLRewrite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"number":   1,
				"title":    "Test",
				"labels":   []map[string]any{},
				"assignee": nil,
				"html_url": "https://forge.lthn.ai/core/go-io/issues/1",
			},
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	issues, err := s.listRepoIssues(context.Background(), "core", "go-io", "")
	require.NoError(t, err)
	require.Len(t, issues, 1)
	// URL should be rewritten to use the mock server URL
	assert.Contains(t, issues[0].URL, srv.URL)
}
