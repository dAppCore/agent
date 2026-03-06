package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"forge.lthn.ai/core/agent/pkg/jobrunner"
)

func TestCompletion_Name_Good(t *testing.T) {
	h := NewCompletionHandler(nil)
	assert.Equal(t, "completion", h.Name())
}

func TestCompletion_Match_Good_AgentCompletion(t *testing.T) {
	h := NewCompletionHandler(nil)
	sig := &jobrunner.PipelineSignal{
		Type: "agent_completion",
	}
	assert.True(t, h.Match(sig))
}

func TestCompletion_Match_Bad_WrongType(t *testing.T) {
	h := NewCompletionHandler(nil)
	sig := &jobrunner.PipelineSignal{
		Type: "pr_update",
	}
	assert.False(t, h.Match(sig))
}

func TestCompletion_Match_Bad_EmptyType(t *testing.T) {
	h := NewCompletionHandler(nil)
	sig := &jobrunner.PipelineSignal{}
	assert.False(t, h.Match(sig))
}

func TestCompletion_Execute_Good_Success(t *testing.T) {
	var labelRemoved bool
	var labelAdded bool
	var commentPosted bool
	var commentBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// GetLabelByName (in-progress) — GET labels to find in-progress.
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-org/test-repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "in-progress", "color": "#1d76db"},
			})

		// RemoveIssueLabel (in-progress).
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/repos/test-org/test-repo/issues/5/labels/1":
			labelRemoved = true
			w.WriteHeader(http.StatusNoContent)

		// EnsureLabel (agent-completed) — POST to create.
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-org/test-repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "agent-completed", "color": "#0e8a16"})

		// AddIssueLabels.
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-org/test-repo/issues/5/labels":
			labelAdded = true
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 2, "name": "agent-completed"}})

		// CreateIssueComment.
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-org/test-repo/issues/5/comments":
			commentPosted = true
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			commentBody = body["body"]
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "body": body["body"]})

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
		RepoOwner:   "test-org",
		RepoName:    "test-repo",
		ChildNumber: 5,
		EpicNumber:  3,
		Success:     true,
		Message:     "Task completed successfully",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, "completion", result.Action)
	assert.Equal(t, "test-org", result.RepoOwner)
	assert.Equal(t, "test-repo", result.RepoName)
	assert.Equal(t, 3, result.EpicNumber)
	assert.Equal(t, 5, result.ChildNumber)
	assert.True(t, labelRemoved, "in-progress label should be removed")
	assert.True(t, labelAdded, "agent-completed label should be added")
	assert.True(t, commentPosted, "comment should be posted")
	assert.Contains(t, commentBody, "Task completed successfully")
}

func TestCompletion_Execute_Good_Failure(t *testing.T) {
	var labelAdded bool
	var commentBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/test-org/test-repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{})

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-org/test-repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "name": "agent-failed", "color": "#c0392b"})

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-org/test-repo/issues/5/labels":
			labelAdded = true
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 3, "name": "agent-failed"}})

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/test-org/test-repo/issues/5/comments":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			commentBody = body["body"]
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "body": body["body"]})

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
		RepoOwner:   "test-org",
		RepoName:    "test-repo",
		ChildNumber: 5,
		EpicNumber:  3,
		Success:     false,
		Error:       "tests failed",
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)

	assert.True(t, result.Success) // The handler itself succeeded
	assert.Equal(t, "completion", result.Action)
	assert.True(t, labelAdded, "agent-failed label should be added")
	assert.Contains(t, commentBody, "Agent reported failure")
	assert.Contains(t, commentBody, "tests failed")
}

func TestCompletion_Execute_Good_FailureNoError(t *testing.T) {
	var commentBody string

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "name": "agent-failed", "color": "#c0392b"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/issues/1/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/issues/1/comments":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			commentBody = body["body"]
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
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
		Error:       "", // No error message.
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, commentBody, "Agent reported failure")
	assert.NotContains(t, commentBody, "Error:") // No error detail.
}

func TestCompletion_Execute_Good_SuccessNoMessage(t *testing.T) {
	var commentPosted bool

	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "agent-completed", "color": "#0e8a16"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/issues/1/labels":
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/issues/1/comments":
			commentPosted = true
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
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
		Success:     true,
		Message:     "", // No message.
	}

	result, err := h.Execute(context.Background(), sig)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.False(t, commentPosted, "no comment should be posted when message is empty")
}

func TestCompletion_Execute_Bad_EnsureLabelFails(t *testing.T) {
	srv := httptest.NewServer(withVersion(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/org/repo/labels":
			// Return empty so EnsureLabel tries to create.
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/org/repo/labels":
			// Label creation fails.
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
		Success:     true,
	}

	_, err := h.Execute(context.Background(), sig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ensure label")
}
