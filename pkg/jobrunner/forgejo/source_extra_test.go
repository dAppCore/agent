package forgejo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"forge.lthn.ai/core/agent/pkg/jobrunner"
)

func TestForgejoSource_Poll_Good_InvalidRepo(t *testing.T) {
	// Invalid repo format should be logged and skipped, not error.
	s := New(Config{Repos: []string{"invalid-no-slash"}}, nil)
	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, signals)
}

func TestForgejoSource_Poll_Good_MultipleRepos(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			// Return one epic per repo.
			issues := []map[string]any{
				{
					"number": 1,
					"body":   "- [ ] #2\n",
					"labels": []map[string]string{{"name": "epic"}},
					"state":  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			prs := []map[string]any{
				{
					"number":    10,
					"body":      "Fixes #2",
					"state":     "open",
					"mergeable": true,
					"merged":    false,
					"head":      map[string]string{"sha": "abc", "ref": "fix", "label": "fix"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		case strings.Contains(path, "/status"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state":       "success",
				"total_count": 1,
				"statuses":    []any{},
			})

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org-a/repo-1", "org-b/repo-2"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	assert.Len(t, signals, 2)
}

func TestForgejoSource_Poll_Good_NeedsCoding(t *testing.T) {
	// When a child issue has no linked PR but is assigned, NeedsCoding should be true.
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues/5"):
			// Child issue with assignee.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number":    5,
				"title":     "Implement feature",
				"body":      "Please implement this.",
				"state":     "open",
				"assignees": []map[string]any{{"login": "darbs-claude", "username": "darbs-claude"}},
			})

		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 1,
					"body":   "- [ ] #5\n",
					"labels": []map[string]string{{"name": "epic"}},
					"state":  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			// No PRs linked.
			_ = json.NewEncoder(w).Encode([]any{})

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"test-org/test-repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)

	require.Len(t, signals, 1)
	sig := signals[0]
	assert.True(t, sig.NeedsCoding)
	assert.Equal(t, "darbs-claude", sig.Assignee)
	assert.Equal(t, "Implement feature", sig.IssueTitle)
	assert.Equal(t, "Please implement this.", sig.IssueBody)
	assert.Equal(t, 5, sig.ChildNumber)
}

func TestForgejoSource_Poll_Good_MergedPR(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 1,
					"body":   "- [ ] #3\n",
					"labels": []map[string]string{{"name": "epic"}},
					"state":  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			prs := []map[string]any{
				{
					"number":    20,
					"body":      "Fixes #3",
					"state":     "closed",
					"mergeable": false,
					"merged":    true,
					"head":      map[string]string{"sha": "merged123", "ref": "fix", "label": "fix"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		case strings.Contains(path, "/status"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state":       "success",
				"total_count": 1,
				"statuses":    []any{},
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

	require.Len(t, signals, 1)
	assert.Equal(t, "MERGED", signals[0].PRState)
	assert.Equal(t, "UNKNOWN", signals[0].Mergeable)
}

func TestForgejoSource_Poll_Good_NoHeadSHA(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 1,
					"body":   "- [ ] #3\n",
					"labels": []map[string]string{{"name": "epic"}},
					"state":  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			prs := []map[string]any{
				{
					"number":    20,
					"body":      "Fixes #3",
					"state":     "open",
					"mergeable": true,
					"merged":    false,
					// No head field.
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

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
	// Without head SHA, check status stays PENDING.
	assert.Equal(t, "PENDING", signals[0].CheckStatus)
}

func TestForgejoSource_Report_Good_Nil(t *testing.T) {
	s := New(Config{}, nil)
	err := s.Report(context.Background(), nil)
	assert.NoError(t, err)
}

func TestForgejoSource_Report_Good_Failed(t *testing.T) {
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

	result := &jobrunner.ActionResult{
		Action:      "dispatch",
		RepoOwner:   "org",
		RepoName:    "repo",
		EpicNumber:  1,
		ChildNumber: 2,
		PRNumber:    3,
		Success:     false,
		Error:       "transfer failed",
	}

	err := s.Report(context.Background(), result)
	require.NoError(t, err)
	assert.Contains(t, capturedBody, "failed")
	assert.Contains(t, capturedBody, "transfer failed")
}

func TestForgejoSource_Poll_Good_APIErrors(t *testing.T) {
	// When the issues API fails, poll should continue with other repos.
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, signals)
}

func TestForgejoSource_Poll_Good_EmptyRepos(t *testing.T) {
	s := New(Config{Repos: []string{}}, nil)
	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, signals)
}

func TestForgejoSource_Poll_Good_NonEpicIssues(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			// Issues without the "epic" label.
			issues := []map[string]any{
				{
					"number": 1,
					"body":   "- [ ] #2\n",
					"labels": []map[string]string{{"name": "bug"}},
					"state":  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, signals, "non-epic issues should not generate signals")
}
