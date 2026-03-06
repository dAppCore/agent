package forgejo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"forge.lthn.ai/core/agent/pkg/jobrunner"
)

// ---------------------------------------------------------------------------
// Supplementary Forgejo signal source tests — extends Phase 3 coverage
// ---------------------------------------------------------------------------

func TestForgejoSource_Poll_Good_MultipleEpicsMultipleChildren(t *testing.T) {
	// Two epics, each with multiple unchecked children that have linked PRs.
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 10,
					"body":   "## Sprint\n- [ ] #11\n- [ ] #12\n- [x] #13\n",
					"labels": []map[string]string{{"name": "epic"}},
					"state":  "open",
				},
				{
					"number": 20,
					"body":   "## Sprint 2\n- [ ] #21\n",
					"labels": []map[string]string{{"name": "epic"}},
					"state":  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			prs := []map[string]any{
				{
					"number": 30, "body": "Fixes #11", "state": "open",
					"mergeable": true, "merged": false,
					"head": map[string]string{"sha": "aaa111", "ref": "fix-11", "label": "fix-11"},
				},
				{
					"number": 31, "body": "Fixes #12", "state": "open",
					"mergeable": false, "merged": false,
					"head": map[string]string{"sha": "bbb222", "ref": "fix-12", "label": "fix-12"},
				},
				{
					"number": 32, "body": "Resolves #21", "state": "open",
					"mergeable": true, "merged": false,
					"head": map[string]string{"sha": "ccc333", "ref": "fix-21", "label": "fix-21"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		case strings.Contains(path, "/status"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state": "success", "total_count": 1, "statuses": []any{},
			})

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)

	// Epic 10 has #11 and #12 unchecked; epic 20 has #21 unchecked. Total 3 signals.
	require.Len(t, signals, 3, "expected three signals from two epics")

	childNumbers := map[int]bool{}
	for _, sig := range signals {
		childNumbers[sig.ChildNumber] = true
	}
	assert.True(t, childNumbers[11])
	assert.True(t, childNumbers[12])
	assert.True(t, childNumbers[21])
}

func TestForgejoSource_Poll_Good_CombinedStatusFetchErrorFallsToPending(t *testing.T) {
	// When combined status fetch fails, check status should default to PENDING.
	var statusFetched atomic.Bool

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 1, "body": "- [ ] #2\n",
					"labels": []map[string]string{{"name": "epic"}}, "state": "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			prs := []map[string]any{
				{
					"number": 10, "body": "Fixes #2", "state": "open",
					"mergeable": true, "merged": false,
					"head": map[string]string{"sha": "sha123", "ref": "fix", "label": "fix"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		case strings.Contains(path, "/status"):
			statusFetched.Store(true)
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)

	require.Len(t, signals, 1)
	assert.True(t, statusFetched.Load(), "status endpoint should have been called")
	assert.Equal(t, "PENDING", signals[0].CheckStatus, "failed status fetch should default to PENDING")
}

func TestForgejoSource_Poll_Good_MixedReposFirstFailsSecondSucceeds(t *testing.T) {
	// First repo fails (issues endpoint 500), second repo succeeds.
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/repos/bad-org/bad-repo/issues"):
			w.WriteHeader(http.StatusInternalServerError)

		case strings.Contains(path, "/repos/good-org/good-repo/issues"):
			issues := []map[string]any{
				{
					"number": 1, "body": "- [ ] #2\n",
					"labels": []map[string]string{{"name": "epic"}}, "state": "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/repos/good-org/good-repo/pulls"):
			prs := []map[string]any{
				{
					"number": 10, "body": "Fixes #2", "state": "open",
					"mergeable": true, "merged": false,
					"head": map[string]string{"sha": "abc", "ref": "fix", "label": "fix"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		case strings.Contains(path, "/status"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state": "success", "total_count": 1, "statuses": []any{},
			})

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"bad-org/bad-repo", "good-org/good-repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	require.Len(t, signals, 1, "only the good repo should produce signals")
	assert.Equal(t, "good-org", signals[0].RepoOwner)
	assert.Equal(t, "good-repo", signals[0].RepoName)
}

func TestForgejoSource_Report_Good_CommentBodyTable(t *testing.T) {
	tests := []struct {
		name         string
		result       *jobrunner.ActionResult
		wantContains []string
	}{
		{
			name: "successful action",
			result: &jobrunner.ActionResult{
				Action: "enable_auto_merge", RepoOwner: "org", RepoName: "repo",
				EpicNumber: 10, ChildNumber: 11, PRNumber: 20, Success: true,
			},
			wantContains: []string{"enable_auto_merge", "succeeded", "#11", "PR #20"},
		},
		{
			name: "failed action with error",
			result: &jobrunner.ActionResult{
				Action: "tick_parent", RepoOwner: "org", RepoName: "repo",
				EpicNumber: 10, ChildNumber: 11, PRNumber: 20,
				Success: false, Error: "rate limit exceeded",
			},
			wantContains: []string{"tick_parent", "failed", "#11", "PR #20", "rate limit exceeded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody string

			srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				capturedBody = body["body"]
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			})))
			defer srv.Close()

			client := newTestClient(t, srv.URL)
			s := New(Config{}, client)

			err := s.Report(context.Background(), tt.result)
			require.NoError(t, err)

			for _, want := range tt.wantContains {
				assert.Contains(t, capturedBody, want)
			}
		})
	}
}

func TestForgejoSource_Report_Good_PostsToCorrectEpicIssue(t *testing.T) {
	var capturedPath string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			capturedPath = r.URL.Path
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{}, client)

	result := &jobrunner.ActionResult{
		Action: "merge", RepoOwner: "test-org", RepoName: "test-repo",
		EpicNumber: 42, ChildNumber: 7, PRNumber: 99, Success: true,
	}

	err := s.Report(context.Background(), result)
	require.NoError(t, err)

	expected := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/comments", result.RepoOwner, result.RepoName, result.EpicNumber)
	assert.Equal(t, expected, capturedPath, "comment should be posted on the epic issue")
}

func TestForgejoSource_Poll_Good_SignalFieldCompleteness(t *testing.T) {
	// Verify that all expected signal fields are populated correctly.
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 100, "body": "## Work\n- [ ] #101\n- [x] #102\n",
					"labels": []map[string]string{{"name": "epic"}}, "state": "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			prs := []map[string]any{
				{
					"number": 200, "body": "Closes #101", "state": "open",
					"mergeable": true, "merged": false,
					"head": map[string]string{"sha": "deadbeef", "ref": "feature", "label": "feature"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		case strings.Contains(path, "/status"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state": "success", "total_count": 2,
				"statuses": []map[string]any{{"status": "success"}, {"status": "success"}},
			})

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"acme/widgets"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)

	require.Len(t, signals, 1)
	sig := signals[0]

	assert.Equal(t, 100, sig.EpicNumber)
	assert.Equal(t, 101, sig.ChildNumber)
	assert.Equal(t, 200, sig.PRNumber)
	assert.Equal(t, "acme", sig.RepoOwner)
	assert.Equal(t, "widgets", sig.RepoName)
	assert.Equal(t, "OPEN", sig.PRState)
	assert.Equal(t, "MERGEABLE", sig.Mergeable)
	assert.Equal(t, "SUCCESS", sig.CheckStatus)
	assert.Equal(t, "deadbeef", sig.LastCommitSHA)
	assert.False(t, sig.NeedsCoding)
	assert.Equal(t, "acme/widgets", sig.RepoFullName())
}

func TestForgejoSource_Poll_Good_AllChildrenCheckedNoSignals(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 1, "body": "- [x] #2\n- [x] #3\n",
					"labels": []map[string]string{{"name": "epic"}}, "state": "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			_ = json.NewEncoder(w).Encode([]any{})

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, signals, "all children checked means no work to do")
}

func TestForgejoSource_Poll_Good_NeedsCodingSignalFields(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues/7"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 7, "title": "Implement authentication",
				"body": "Add OAuth2 support.", "state": "open",
				"assignees": []map[string]any{{"login": "agent-bot", "username": "agent-bot"}},
			})

		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 1, "body": "- [ ] #7\n",
					"labels": []map[string]string{{"name": "epic"}}, "state": "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			_ = json.NewEncoder(w).Encode([]any{})

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)

	require.Len(t, signals, 1)
	sig := signals[0]
	assert.True(t, sig.NeedsCoding)
	assert.Equal(t, "agent-bot", sig.Assignee)
	assert.Equal(t, "Implement authentication", sig.IssueTitle)
	assert.Contains(t, sig.IssueBody, "OAuth2 support")
	assert.Equal(t, 0, sig.PRNumber, "PRNumber should be zero for NeedsCoding signals")
}
