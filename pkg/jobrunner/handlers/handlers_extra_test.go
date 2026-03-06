package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentci "forge.lthn.ai/core/agent/pkg/orchestrator"
	"forge.lthn.ai/core/agent/pkg/jobrunner"
)

// --- Name tests for all handlers ---

func TestEnableAutoMerge_Name_Good(t *testing.T) {
	h := NewEnableAutoMergeHandler(nil)
	assert.Equal(t, "enable_auto_merge", h.Name())
}

func TestPublishDraft_Name_Good(t *testing.T) {
	h := NewPublishDraftHandler(nil)
	assert.Equal(t, "publish_draft", h.Name())
}

func TestDismissReviews_Name_Good(t *testing.T) {
	h := NewDismissReviewsHandler(nil)
	assert.Equal(t, "dismiss_reviews", h.Name())
}

func TestSendFixCommand_Name_Good(t *testing.T) {
	h := NewSendFixCommandHandler(nil)
	assert.Equal(t, "send_fix_command", h.Name())
}

func TestTickParent_Name_Good(t *testing.T) {
	h := NewTickParentHandler(nil)
	assert.Equal(t, "tick_parent", h.Name())
}

// --- Additional Match tests ---

func TestEnableAutoMerge_Match_Bad_Closed(t *testing.T) {
	h := NewEnableAutoMergeHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:     "CLOSED",
		Mergeable:   "MERGEABLE",
		CheckStatus: "SUCCESS",
	}
	assert.False(t, h.Match(sig))
}

func TestEnableAutoMerge_Match_Bad_ChecksFailing(t *testing.T) {
	h := NewEnableAutoMergeHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:     "OPEN",
		Mergeable:   "MERGEABLE",
		CheckStatus: "FAILURE",
	}
	assert.False(t, h.Match(sig))
}

func TestEnableAutoMerge_Match_Bad_Conflicting(t *testing.T) {
	h := NewEnableAutoMergeHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:     "OPEN",
		Mergeable:   "CONFLICTING",
		CheckStatus: "SUCCESS",
	}
	assert.False(t, h.Match(sig))
}

func TestPublishDraft_Match_Bad_Closed(t *testing.T) {
	h := NewPublishDraftHandler(nil)
	sig := &jobrunner.PipelineSignal{
		IsDraft:     true,
		PRState:     "CLOSED",
		CheckStatus: "SUCCESS",
	}
	assert.False(t, h.Match(sig))
}

func TestDismissReviews_Match_Bad_Closed(t *testing.T) {
	h := NewDismissReviewsHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:         "CLOSED",
		ThreadsTotal:    3,
		ThreadsResolved: 1,
	}
	assert.False(t, h.Match(sig))
}

func TestDismissReviews_Match_Bad_NoThreads(t *testing.T) {
	h := NewDismissReviewsHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:         "OPEN",
		ThreadsTotal:    0,
		ThreadsResolved: 0,
	}
	assert.False(t, h.Match(sig))
}

func TestSendFixCommand_Match_Bad_Closed(t *testing.T) {
	h := NewSendFixCommandHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:   "CLOSED",
		Mergeable: "CONFLICTING",
	}
	assert.False(t, h.Match(sig))
}

func TestSendFixCommand_Match_Bad_NoIssues(t *testing.T) {
	h := NewSendFixCommandHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:     "OPEN",
		Mergeable:   "MERGEABLE",
		CheckStatus: "SUCCESS",
	}
	assert.False(t, h.Match(sig))
}

func TestSendFixCommand_Match_Good_ThreadsFailure(t *testing.T) {
	h := NewSendFixCommandHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:         "OPEN",
		Mergeable:       "MERGEABLE",
		CheckStatus:     "FAILURE",
		ThreadsTotal:    2,
		ThreadsResolved: 0,
	}
	assert.True(t, h.Match(sig))
}

func TestTickParent_Match_Bad_Closed(t *testing.T) {
	h := NewTickParentHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState: "CLOSED",
	}
	assert.False(t, h.Match(sig))
}

// --- Additional Execute tests ---

func TestPublishDraft_Execute_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewPublishDraftHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner: "org",
		RepoName:  "repo",
		PRNumber:  1,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "publish draft failed")
}

func TestSendFixCommand_Execute_Good_Reviews(t *testing.T) {
	var capturedBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			b, _ := io.ReadAll(r.Body)
			capturedBody = string(b)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewSendFixCommandHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner:       "org",
		RepoName:        "repo",
		PRNumber:        5,
		PRState:         "OPEN",
		Mergeable:       "MERGEABLE",
		CheckStatus:     "FAILURE",
		ThreadsTotal:    2,
		ThreadsResolved: 0,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, capturedBody, "fix the code reviews")
}

func TestSendFixCommand_Execute_Bad_CommentFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewSendFixCommandHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner: "org",
		RepoName:  "repo",
		PRNumber:  1,
		Mergeable: "CONFLICTING",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "post comment failed")
}

func TestTickParent_Execute_Good_AlreadyTicked(t *testing.T) {
	epicBody := "## Tasks\n- [x] #7\n- [ ] #8\n"

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/42") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 42,
				"body":   epicBody,
				"title":  "Epic",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewTickParentHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner:   "org",
		RepoName:    "repo",
		EpicNumber:  42,
		ChildNumber: 7,
		PRNumber:    99,
		PRState:     "MERGED",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "tick_parent", result.Action)
}

func TestTickParent_Execute_Bad_FetchEpicFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewTickParentHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner:   "org",
		RepoName:    "repo",
		EpicNumber:  999,
		ChildNumber: 1,
		PRState:     "MERGED",
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetch epic")
}

func TestTickParent_Execute_Bad_EditEpicFails(t *testing.T) {
	epicBody := "## Tasks\n- [ ] #7\n"

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/42"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 42,
				"body":   epicBody,
				"title":  "Epic",
			})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/issues/42"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewTickParentHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner:   "org",
		RepoName:    "repo",
		EpicNumber:  42,
		ChildNumber: 7,
		PRNumber:    99,
		PRState:     "MERGED",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "edit epic failed")
}

func TestTickParent_Execute_Bad_CloseChildFails(t *testing.T) {
	epicBody := "## Tasks\n- [ ] #7\n"

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/42"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 42,
				"body":   epicBody,
				"title":  "Epic",
			})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/issues/42"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 42,
				"body":   strings.Replace(epicBody, "- [ ] #7", "- [x] #7", 1),
				"title":  "Epic",
			})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/issues/7"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewTickParentHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner:   "org",
		RepoName:    "repo",
		EpicNumber:  42,
		ChildNumber: 7,
		PRNumber:    99,
		PRState:     "MERGED",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "close child issue failed")
}

func TestDismissReviews_Execute_Bad_ListFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewDismissReviewsHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner: "org",
		RepoName:  "repo",
		PRNumber:  1,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list reviews")
}

func TestDismissReviews_Execute_Good_NothingToDismiss(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			// All reviews are either approved or already dismissed.
			reviews := []map[string]any{
				{
					"id": 1, "state": "APPROVED", "dismissed": false, "stale": false,
					"body": "lgtm", "commit_id": "abc123",
				},
				{
					"id": 2, "state": "REQUEST_CHANGES", "dismissed": true, "stale": true,
					"body": "already dismissed", "commit_id": "abc123",
				},
				{
					"id": 3, "state": "REQUEST_CHANGES", "dismissed": false, "stale": false,
					"body": "not stale", "commit_id": "abc123",
				},
			}
			_ = json.NewEncoder(w).Encode(reviews)
			return
		}
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewDismissReviewsHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner: "org",
		RepoName:  "repo",
		PRNumber:  1,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success, "nothing to dismiss should be success")
}

func TestDismissReviews_Execute_Bad_DismissFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			reviews := []map[string]any{
				{
					"id": 1, "state": "REQUEST_CHANGES", "dismissed": false, "stale": true,
					"body": "fix it", "commit_id": "abc123",
				},
			}
			_ = json.NewEncoder(w).Encode(reviews)
			return
		}

		// Dismiss fails.
		w.WriteHeader(http.StatusForbidden)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewDismissReviewsHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner: "org",
		RepoName:  "repo",
		PRNumber:  1,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "failed to dismiss")
}

// --- Dispatch Execute edge cases ---

func TestDispatch_Execute_Good_AlreadyInProgress(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "in-progress", "color": "#1d76db"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "in-progress"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			// Issue already has in-progress label.
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     5,
				"number": 5,
				"labels": []map[string]any{{"name": "in-progress", "id": 1}},
				"title":  "Test",
			})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)

	spinner := newTestSpinner(map[string]agentci.AgentConfig{
		"darbs-claude": {Host: "localhost", QueueDir: "/tmp/queue", Active: true},
	})
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	sig := &jobrunner.PipelineSignal{
		NeedsCoding: true,
		Assignee:    "darbs-claude",
		RepoOwner:   "org",
		RepoName:    "repo",
		ChildNumber: 5,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success, "already in-progress should be a no-op success")
}

func TestDispatch_Execute_Good_AlreadyCompleted(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 2, "name": "agent-completed", "color": "#0e8a16"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "in-progress"})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     5,
				"number": 5,
				"labels": []map[string]any{{"name": "agent-completed", "id": 2}},
				"title":  "Done",
			})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	spinner := newTestSpinner(map[string]agentci.AgentConfig{
		"darbs-claude": {Host: "localhost", QueueDir: "/tmp/queue", Active: true},
	})
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	sig := &jobrunner.PipelineSignal{
		NeedsCoding: true,
		Assignee:    "darbs-claude",
		RepoOwner:   "org",
		RepoName:    "repo",
		ChildNumber: 5,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestDispatch_Execute_Bad_InvalidRepoOwner(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	spinner := newTestSpinner(map[string]agentci.AgentConfig{
		"darbs-claude": {Host: "localhost", QueueDir: "/tmp/queue", Active: true},
	})
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	sig := &jobrunner.PipelineSignal{
		NeedsCoding: true,
		Assignee:    "darbs-claude",
		RepoOwner:   "org$bad",
		RepoName:    "repo",
		ChildNumber: 1,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo owner")
}
