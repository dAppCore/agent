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

	forgejosdk "codeberg.org/mvdkleijn/forgejo-sdk/forgejo/v2"

	"forge.lthn.ai/core/go-scm/forge"
	"forge.lthn.ai/core/agent/pkg/jobrunner"
)

// --- Signal parsing and filtering tests ---

func TestParseEpicChildren_Good_EmptyBody(t *testing.T) {
	unchecked, checked := parseEpicChildren("")
	assert.Nil(t, unchecked)
	assert.Nil(t, checked)
}

func TestParseEpicChildren_Good_MixedContent(t *testing.T) {
	// Checkboxes mixed with regular markdown content.
	body := `## Epic: Refactor Auth

Some description of the epic.

### Tasks
- [x] #10 — Migrate session store
- [ ] #11 — Update OAuth flow
- [x] #12 — Fix token refresh
- [ ] #13 — Add 2FA support

### Notes
This is a note, not a checkbox.
- Regular list item
- Another item
`
	unchecked, checked := parseEpicChildren(body)
	assert.Equal(t, []int{11, 13}, unchecked)
	assert.Equal(t, []int{10, 12}, checked)
}

func TestParseEpicChildren_Good_LargeIssueNumbers(t *testing.T) {
	body := "- [ ] #9999\n- [x] #10000\n"
	unchecked, checked := parseEpicChildren(body)
	assert.Equal(t, []int{9999}, unchecked)
	assert.Equal(t, []int{10000}, checked)
}

func TestParseEpicChildren_Good_ConsecutiveCheckboxes(t *testing.T) {
	body := "- [ ] #1\n- [ ] #2\n- [ ] #3\n- [ ] #4\n- [ ] #5\n"
	unchecked, checked := parseEpicChildren(body)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, unchecked)
	assert.Nil(t, checked)
}

// --- findLinkedPR tests ---

func TestFindLinkedPR_Good_MultipleReferencesInBody(t *testing.T) {
	prs := []*forgejosdk.PullRequest{
		{Index: 10, Body: "Fixes #5 and relates to #7"},
		{Index: 11, Body: "Closes #8"},
	}

	// Should find PR #10 because it references #7.
	pr := findLinkedPR(prs, 7)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(10), pr.Index)

	// Should find PR #10 because it references #5.
	pr = findLinkedPR(prs, 5)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(10), pr.Index)
}

func TestFindLinkedPR_Good_EmptyBodyPR(t *testing.T) {
	prs := []*forgejosdk.PullRequest{
		{Index: 10, Body: ""},
		{Index: 11, Body: "Fixes #7"},
	}

	pr := findLinkedPR(prs, 7)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(11), pr.Index)
}

func TestFindLinkedPR_Good_FirstMatchWins(t *testing.T) {
	// Both PRs reference #7, first one should win.
	prs := []*forgejosdk.PullRequest{
		{Index: 10, Body: "Fixes #7"},
		{Index: 11, Body: "Also fixes #7"},
	}

	pr := findLinkedPR(prs, 7)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(10), pr.Index)
}

func TestFindLinkedPR_Good_EmptySlice(t *testing.T) {
	prs := []*forgejosdk.PullRequest{}
	pr := findLinkedPR(prs, 1)
	assert.Nil(t, pr)
}

// --- mapPRState edge case ---

func TestMapPRState_Good_MergedOverridesState(t *testing.T) {
	// HasMerged=true should return MERGED regardless of State.
	pr := &forgejosdk.PullRequest{State: forgejosdk.StateOpen, HasMerged: true}
	assert.Equal(t, "MERGED", mapPRState(pr))
}

// --- mapCombinedStatus edge cases ---

func TestMapCombinedStatus_Good_WarningState(t *testing.T) {
	// Unknown/warning state should default to PENDING.
	cs := &forgejosdk.CombinedStatus{
		State:      forgejosdk.StatusWarning,
		TotalCount: 1,
	}
	assert.Equal(t, "PENDING", mapCombinedStatus(cs))
}

// --- buildSignal edge cases ---

func TestBuildSignal_Good_ClosedPR(t *testing.T) {
	pr := &forgejosdk.PullRequest{
		Index:     5,
		State:     forgejosdk.StateClosed,
		Mergeable: false,
		HasMerged: false,
		Head:      &forgejosdk.PRBranchInfo{Sha: "abc"},
	}

	sig := buildSignal("org", "repo", 1, 2, pr, "FAILURE")
	assert.Equal(t, "CLOSED", sig.PRState)
	assert.Equal(t, "CONFLICTING", sig.Mergeable)
	assert.Equal(t, "FAILURE", sig.CheckStatus)
	assert.Equal(t, "abc", sig.LastCommitSHA)
}

func TestBuildSignal_Good_MergedPR(t *testing.T) {
	pr := &forgejosdk.PullRequest{
		Index:     99,
		State:     forgejosdk.StateClosed,
		Mergeable: false,
		HasMerged: true,
		Head:      &forgejosdk.PRBranchInfo{Sha: "merged123"},
	}

	sig := buildSignal("owner", "repo", 10, 5, pr, "SUCCESS")
	assert.Equal(t, "MERGED", sig.PRState)
	assert.Equal(t, "UNKNOWN", sig.Mergeable)
	assert.Equal(t, 99, sig.PRNumber)
	assert.Equal(t, "merged123", sig.LastCommitSHA)
}

// --- splitRepo edge cases ---

func TestSplitRepo_Bad_OnlySlash(t *testing.T) {
	_, _, err := splitRepo("/")
	assert.Error(t, err)
}

func TestSplitRepo_Bad_MultipleSlashes(t *testing.T) {
	// Should take only the first part as owner, rest as repo.
	owner, repo, err := splitRepo("a/b/c")
	require.NoError(t, err)
	assert.Equal(t, "a", owner)
	assert.Equal(t, "b/c", repo)
}

// --- Poll with combined status failure ---

func TestForgejoSource_Poll_Good_CombinedStatusFailure(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
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
					"head":      map[string]string{"sha": "fail123", "ref": "feature", "label": "feature"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		case strings.Contains(path, "/status"):
			status := map[string]any{
				"state":       "failure",
				"total_count": 2,
				"statuses":    []map[string]any{{"status": "failure", "context": "ci"}},
			}
			_ = json.NewEncoder(w).Encode(status)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)

	require.Len(t, signals, 1)
	assert.Equal(t, "FAILURE", signals[0].CheckStatus)
	assert.Equal(t, "OPEN", signals[0].PRState)
	assert.Equal(t, "MERGEABLE", signals[0].Mergeable)
}

// --- Poll with combined status error ---

func TestForgejoSource_Poll_Good_CombinedStatusError(t *testing.T) {
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
					"mergeable": false,
					"merged":    false,
					"head":      map[string]string{"sha": "err123", "ref": "fix", "label": "fix"},
				},
			}
			_ = json.NewEncoder(w).Encode(prs)

		// Combined status endpoint returns 500 — should fall back to PENDING.
		case strings.Contains(path, "/status"):
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	defer srv.Close()

	client := newTestClient(t, srv.URL)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)

	require.Len(t, signals, 1)
	// Combined status API error -> falls back to PENDING.
	assert.Equal(t, "PENDING", signals[0].CheckStatus)
	assert.Equal(t, "CONFLICTING", signals[0].Mergeable)
}

// --- Poll with child that has no assignee (NeedsCoding path, no assignee) ---

func TestForgejoSource_Poll_Good_ChildNoAssignee(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues/5"):
			// Child issue with no assignee.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number":    5,
				"title":     "Unassigned task",
				"body":      "No one is working on this.",
				"state":     "open",
				"assignees": []map[string]any{},
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

	// No signal should be emitted when child has no assignee and no PR.
	assert.Empty(t, signals)
}

// --- Poll with child issue fetch failure ---

func TestForgejoSource_Poll_Good_ChildFetchFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues/5"):
			// Child issue fetch fails.
			w.WriteHeader(http.StatusInternalServerError)

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

	// Child fetch error should be logged and skipped, not returned as error.
	assert.Empty(t, signals)
}

// --- Poll with multiple epics ---

func TestForgejoSource_Poll_Good_MultipleEpics(t *testing.T) {
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
				{
					"number": 2,
					"body":   "- [ ] #4\n",
					"labels": []map[string]string{{"name": "epic"}},
					"state":  "open",
				},
			}
			_ = json.NewEncoder(w).Encode(issues)

		case strings.Contains(path, "/pulls"):
			prs := []map[string]any{
				{
					"number":    10,
					"body":      "Fixes #3",
					"state":     "open",
					"mergeable": true,
					"merged":    false,
					"head":      map[string]string{"sha": "aaa", "ref": "f1", "label": "f1"},
				},
				{
					"number":    11,
					"body":      "Fixes #4",
					"state":     "open",
					"mergeable": true,
					"merged":    false,
					"head":      map[string]string{"sha": "bbb", "ref": "f2", "label": "f2"},
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

	require.Len(t, signals, 2)
	assert.Equal(t, 1, signals[0].EpicNumber)
	assert.Equal(t, 3, signals[0].ChildNumber)
	assert.Equal(t, 10, signals[0].PRNumber)

	assert.Equal(t, 2, signals[1].EpicNumber)
	assert.Equal(t, 4, signals[1].ChildNumber)
	assert.Equal(t, 11, signals[1].PRNumber)
}

// --- Report with nil result ---

func TestForgejoSource_Report_Good_NilResult(t *testing.T) {
	s := New(Config{}, nil)
	err := s.Report(context.Background(), nil)
	assert.NoError(t, err)
}

// --- Report constructs correct comment body ---

func TestForgejoSource_Report_Good_SuccessFormat(t *testing.T) {
	var capturedPath string
	var capturedBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
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
		Action:      "tick_parent",
		RepoOwner:   "core",
		RepoName:    "go-scm",
		EpicNumber:  5,
		ChildNumber: 10,
		PRNumber:    20,
		Success:     true,
	}

	err := s.Report(context.Background(), result)
	require.NoError(t, err)

	// Comment should be on the epic issue.
	assert.Contains(t, capturedPath, "/issues/5/comments")
	assert.Contains(t, capturedBody, "tick_parent")
	assert.Contains(t, capturedBody, "succeeded")
	assert.Contains(t, capturedBody, "#10")
	assert.Contains(t, capturedBody, "PR #20")
}

func TestForgejoSource_Report_Good_FailureWithError(t *testing.T) {
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
		Action:      "enable_auto_merge",
		RepoOwner:   "org",
		RepoName:    "repo",
		EpicNumber:  1,
		ChildNumber: 2,
		PRNumber:    3,
		Success:     false,
		Error:       "merge conflict detected",
	}

	err := s.Report(context.Background(), result)
	require.NoError(t, err)

	assert.Contains(t, capturedBody, "failed")
	assert.Contains(t, capturedBody, "merge conflict detected")
}

// --- Poll filters only epic-labelled issues ---

func TestForgejoSource_Poll_Good_MixedLabels(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(path, "/issues"):
			issues := []map[string]any{
				{
					"number": 1,
					"body":   "- [ ] #2\n",
					"labels": []map[string]string{{"name": "epic"}, {"name": "priority-high"}},
					"state":  "open",
				},
				{
					"number": 3,
					"body":   "- [ ] #4\n",
					"labels": []map[string]string{{"name": "bug"}},
					"state":  "open",
				},
				{
					"number": 5,
					"body":   "- [ ] #6\n",
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
					"head":      map[string]string{"sha": "sha1", "ref": "f1", "label": "f1"},
				},
				{
					"number":    11,
					"body":      "Fixes #4",
					"state":     "open",
					"mergeable": true,
					"merged":    false,
					"head":      map[string]string{"sha": "sha2", "ref": "f2", "label": "f2"},
				},
				{
					"number":    12,
					"body":      "Fixes #6",
					"state":     "open",
					"mergeable": true,
					"merged":    false,
					"head":      map[string]string{"sha": "sha3", "ref": "f3", "label": "f3"},
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

	// Only issues #1 and #5 have the "epic" label.
	require.Len(t, signals, 2)
	assert.Equal(t, 1, signals[0].EpicNumber)
	assert.Equal(t, 2, signals[0].ChildNumber)
	assert.Equal(t, 5, signals[1].EpicNumber)
	assert.Equal(t, 6, signals[1].ChildNumber)
}

// --- Poll with PRs error after issues succeed ---

func TestForgejoSource_Poll_Good_PRsAPIError(t *testing.T) {
	callCount := 0

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		callCount++

		switch {
		case strings.Contains(path, "/issues"):
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
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client, err := forge.New(srv.URL, "test-token")
	require.NoError(t, err)
	s := New(Config{Repos: []string{"org/repo"}}, client)

	signals, err := s.Poll(context.Background())
	require.NoError(t, err)
	// PR API failure -> repo is skipped, no signals.
	assert.Empty(t, signals)
}

// --- New creates source correctly ---

func TestForgejoSource_New_Good(t *testing.T) {
	s := New(Config{Repos: []string{"a/b", "c/d"}}, nil)
	assert.Equal(t, "forgejo", s.Name())
	assert.Equal(t, []string{"a/b", "c/d"}, s.repos)
}
