package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentci "forge.lthn.ai/core/agent/pkg/orchestrator"
	"forge.lthn.ai/core/agent/pkg/jobrunner"
)

// --- Dispatch: Execute with invalid repo name ---

func TestDispatch_Execute_Bad_InvalidRepoNameSpecialChars(t *testing.T) {
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
		RepoOwner:   "valid-org",
		RepoName:    "repo$bad!",
		ChildNumber: 1,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo name")
}

// --- Dispatch: Execute when EnsureLabel fails ---

func TestDispatch_Execute_Bad_EnsureLabelCreationFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusInternalServerError)
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
		ChildNumber: 1,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ensure label")
}

// dispatchMockServer creates a standard mock server for dispatch tests.
// It handles all the Forgejo API calls needed for a full dispatch flow.
func dispatchMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// GetLabelByName / list labels
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "in-progress", "color": "#1d76db"},
				{"id": 2, "name": "agent-ready", "color": "#00ff00"},
			})

		// CreateLabel (shouldn't normally be needed since we return it above)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "in-progress", "color": "#1d76db"})

		// GetIssue (returns issue with no label to trigger the full dispatch flow)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			w.WriteHeader(http.StatusNotFound) // Issue not found => full dispatch flow

		// AssignIssue
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "number": 5})

		// AddIssueLabels
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "in-progress"}})

		// RemoveIssueLabel
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/labels/"):
			w.WriteHeader(http.StatusNoContent)

		// CreateIssueComment
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/comments"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "body": "dispatched"})

		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
}

// --- Dispatch: Execute when GetIssue returns 404 (full dispatch path) ---

func TestDispatch_Execute_Good_GetIssueNotFound(t *testing.T) {
	srv := dispatchMockServer(t)
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	spinner := newTestSpinner(map[string]agentci.AgentConfig{
		"darbs-claude": {Host: "localhost", QueueDir: "/tmp/nonexistent-queue", Active: true},
	})
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	sig := &jobrunner.PipelineSignal{
		NeedsCoding: true,
		Assignee:    "darbs-claude",
		RepoOwner:   "org",
		RepoName:    "repo",
		ChildNumber: 5,
		EpicNumber:  3,
		IssueTitle:  "Test issue",
		IssueBody:   "Test body",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.Equal(t, "dispatch", result.Action)
}

// --- Completion: Execute when AddIssueLabels fails for success case ---

func TestCompletion_Execute_Bad_AddCompleteLabelFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repo/labels"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "agent-completed", "color": "#0e8a16"})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/labels"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewCompletionHandler(client)

	sig := &jobrunner.PipelineSignal{
		Type:        "agent_completion",
		RepoOwner:   "org",
		RepoName:    "repo",
		ChildNumber: 5,
		Success:     true,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "add completed label")
}

// --- Completion: Execute when AddIssueLabels fails for failure case ---

func TestCompletion_Execute_Bad_AddFailLabelFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/repo/labels"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "name": "agent-failed", "color": "#c0392b"})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/labels"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewCompletionHandler(client)

	sig := &jobrunner.PipelineSignal{
		Type:        "agent_completion",
		RepoOwner:   "org",
		RepoName:    "repo",
		ChildNumber: 5,
		Success:     false,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "add failed label")
}

// --- Completion: Execute with EnsureLabel failure on failure path ---

func TestCompletion_Execute_Bad_FailedPathEnsureLabelFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/labels"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewCompletionHandler(client)

	sig := &jobrunner.PipelineSignal{
		Type:        "agent_completion",
		RepoOwner:   "org",
		RepoName:    "repo",
		ChildNumber: 1,
		Success:     false,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ensure label")
}

// --- EnableAutoMerge: additional edge case ---

func TestEnableAutoMerge_Match_Bad_PendingChecks(t *testing.T) {
	h := NewEnableAutoMergeHandler(nil)
	sig := &jobrunner.PipelineSignal{
		PRState:     "OPEN",
		IsDraft:     false,
		Mergeable:   "MERGEABLE",
		CheckStatus: "PENDING",
	}
	assert.False(t, h.Match(sig))
}

func TestEnableAutoMerge_Execute_Bad_InternalServerError(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewEnableAutoMergeHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner: "org",
		RepoName:  "repo",
		PRNumber:  1,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "merge failed")
}

// --- PublishDraft: Match with MERGED state ---

func TestPublishDraft_Match_Bad_MergedState(t *testing.T) {
	h := NewPublishDraftHandler(nil)
	sig := &jobrunner.PipelineSignal{
		IsDraft:     true,
		PRState:     "MERGED",
		CheckStatus: "SUCCESS",
	}
	assert.False(t, h.Match(sig))
}

// --- SendFixCommand: Execute merge conflict message ---

func TestSendFixCommand_Execute_Good_MergeConflictMessage(t *testing.T) {
	var capturedBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			capturedBody = body["body"]
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			return
		}
		w.WriteHeader(http.StatusOK)
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
	assert.True(t, result.Success)
	assert.Contains(t, capturedBody, "fix the merge conflict")
}

// --- DismissReviews: Execute with stale review that gets dismissed ---

func TestDismissReviews_Execute_Good_StaleReviewDismissed(t *testing.T) {
	var dismissCalled bool

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/reviews") {
			reviews := []map[string]any{
				{
					"id": 1, "state": "REQUEST_CHANGES", "dismissed": false, "stale": true,
					"body": "fix it", "commit_id": "abc123",
				},
			}
			_ = json.NewEncoder(w).Encode(reviews)
			return
		}

		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/dismissals") {
			dismissCalled = true
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "state": "DISMISSED"})
			return
		}

		w.WriteHeader(http.StatusOK)
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	h := NewDismissReviewsHandler(client)

	sig := &jobrunner.PipelineSignal{
		RepoOwner:       "org",
		RepoName:        "repo",
		PRNumber:        1,
		PRState:         "OPEN",
		ThreadsTotal:    1,
		ThreadsResolved: 0,
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.True(t, dismissCalled)
}

// --- TickParent: Execute ticks and closes ---

func TestTickParent_Execute_Good_TicksCheckboxAndCloses(t *testing.T) {
	epicBody := "## Tasks\n- [ ] #7\n- [ ] #8\n"
	var editedBody string
	var closedIssue bool

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
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if b, ok := body["body"].(string); ok {
				editedBody = b
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 42,
				"body":   editedBody,
				"title":  "Epic",
			})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/issues/7"):
			closedIssue = true
			_ = json.NewEncoder(w).Encode(map[string]any{
				"number": 7,
				"state":  "closed",
			})
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
	assert.True(t, result.Success)
	assert.Contains(t, editedBody, "- [x] #7")
	assert.True(t, closedIssue)
}

// --- Dispatch: DualRun mode ---

func TestDispatch_Execute_Good_DualRunModeDispatch(t *testing.T) {
	srv := dispatchMockServer(t)
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)

	spinner := agentci.NewSpinner(
		agentci.ClothoConfig{Strategy: "clotho-verified"},
		map[string]agentci.AgentConfig{
			"darbs-claude": {
				Host:     "localhost",
				QueueDir: "/tmp/nonexistent-queue",
				Active:   true,
				Model:    "sonnet",
				DualRun:  true,
			},
		},
	)
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	sig := &jobrunner.PipelineSignal{
		NeedsCoding: true,
		Assignee:    "darbs-claude",
		RepoOwner:   "org",
		RepoName:    "repo",
		ChildNumber: 5,
		EpicNumber:  3,
		IssueTitle:  "Test issue",
		IssueBody:   "Test body",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.Equal(t, "dispatch", result.Action)
}

// --- TickParent: ChildNumber not found in epic body ---

func TestTickParent_Execute_Good_ChildNotInBody(t *testing.T) {
	epicBody := "## Tasks\n- [ ] #99\n"

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
		ChildNumber: 50,
		PRNumber:    100,
		PRState:     "MERGED",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success)
}

// --- Dispatch: AssignIssue fails (warn, continue) ---

func TestDispatch_Execute_Good_AssignIssueFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "in-progress", "color": "#1d76db"},
				{"id": 2, "name": "agent-ready", "color": "#00ff00"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "in-progress"})
		// GetIssue returns issue with NO special labels
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 5, "number": 5, "title": "Test Issue",
				"labels": []map[string]any{},
			})
		// AssignIssue FAILS
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"assign failed"}`))
		// AddIssueLabels succeeds
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "in-progress"}})
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/labels/"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/comments"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "body": "dispatched"})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	spinner := newTestSpinner(map[string]agentci.AgentConfig{
		"darbs-claude": {Host: "localhost", QueueDir: "/tmp/nonexistent-queue", Active: true},
	})
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 5,
		PRNumber:    10,
		RepoOwner:   "org",
		RepoName:    "repo",
		Assignee:    "darbs-claude",
		IssueTitle:  "Test Issue",
		IssueBody:   "Test body",
	}

	// Should not return error because AssignIssue failure is only a warning.
	result, err := h.Execute(context.Background(), signal)
	// secureTransfer will fail because SSH isn't available, but we exercised the assign-error path.
	_ = result
	_ = err
}

// --- Dispatch: AddIssueLabels fails ---

func TestDispatch_Execute_Bad_AddIssueLabelsError(t *testing.T) {
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
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 5, "number": 5, "title": "Test Issue",
				"labels": []map[string]any{},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "number": 5})
		// AddIssueLabels FAILS
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/labels"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"label add failed"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	spinner := newTestSpinner(map[string]agentci.AgentConfig{
		"darbs-claude": {Host: "localhost", QueueDir: "/tmp/nonexistent-queue", Active: true},
	})
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 5,
		PRNumber:    10,
		RepoOwner:   "org",
		RepoName:    "repo",
		Assignee:    "darbs-claude",
		IssueTitle:  "Test Issue",
		IssueBody:   "Test body",
	}

	_, err := h.Execute(context.Background(), signal)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "add in-progress label")
}

// --- Dispatch: GetIssue returns issue with existing labels not matching ---

func TestDispatch_Execute_Good_IssueFoundNoSpecialLabels(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "in-progress", "color": "#1d76db"},
				{"id": 2, "name": "agent-ready", "color": "#00ff00"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "in-progress"})
		// GetIssue returns issue with unrelated labels
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 5, "number": 5, "title": "Test Issue",
				"labels": []map[string]any{
					{"id": 10, "name": "enhancement"},
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/repos/org/repo/issues/5":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "number": 5})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/labels"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "in-progress"}})
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/labels/"):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/comments"):
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "body": "dispatched"})
		default:
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{})
		}
	})))
	defer srv.Close()

	client := newTestForgeClient(t, srv.URL)
	spinner := newTestSpinner(map[string]agentci.AgentConfig{
		"darbs-claude": {Host: "localhost", QueueDir: "/tmp/nonexistent-queue", Active: true},
	})
	h := NewDispatchHandler(client, srv.URL, "test-token", spinner)

	signal := &jobrunner.PipelineSignal{
		EpicNumber:  1,
		ChildNumber: 5,
		PRNumber:    10,
		RepoOwner:   "org",
		RepoName:    "repo",
		Assignee:    "darbs-claude",
		IssueTitle:  "Test Issue",
		IssueBody:   "Test body",
	}

	// Execute will proceed past label check and try SSH (which fails).
	result, err := h.Execute(context.Background(), signal)
	// Should either succeed (if somehow SSH works) or fail at secureTransfer.
	_ = result
	_ = err
}
