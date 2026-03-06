package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Client.CreateTask tests ---

func TestClient_CreateTask_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/tasks", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		var task Task
		err := json.NewDecoder(r.Body).Decode(&task)
		require.NoError(t, err)
		assert.Equal(t, "New feature", task.Title)
		assert.Equal(t, PriorityHigh, task.Priority)

		// Return the task with an assigned ID.
		task.ID = "task-new-1"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	task := Task{
		Title:       "New feature",
		Description: "Build something great",
		Priority:    PriorityHigh,
		Labels:      []string{"feature"},
		Status:      StatusPending,
	}

	created, err := client.CreateTask(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "task-new-1", created.ID)
	assert.Equal(t, "New feature", created.Title)
	assert.Equal(t, PriorityHigh, created.Priority)
}

func TestClient_CreateTask_Bad_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(APIError{Message: "validation failed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	task := Task{Title: "Bad task"}

	created, err := client.CreateTask(context.Background(), task)
	assert.Error(t, err)
	assert.Nil(t, created)
	assert.Contains(t, err.Error(), "validation failed")
}

// --- SubmitTask tests ---

func TestSubmitTask_Good_AllFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var task Task
		err := json.NewDecoder(r.Body).Decode(&task)
		require.NoError(t, err)
		assert.Equal(t, "Implement login", task.Title)
		assert.Equal(t, "OAuth2 login flow", task.Description)
		assert.Equal(t, []string{"auth", "frontend"}, task.Labels)
		assert.Equal(t, PriorityHigh, task.Priority)
		assert.Equal(t, StatusPending, task.Status)
		assert.False(t, task.CreatedAt.IsZero())

		task.ID = "task-submit-1"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	created, err := SubmitTask(context.Background(), client, "Implement login", "OAuth2 login flow", []string{"auth", "frontend"}, PriorityHigh)
	require.NoError(t, err)
	assert.Equal(t, "task-submit-1", created.ID)
	assert.Equal(t, "Implement login", created.Title)
}

func TestSubmitTask_Good_MinimalFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var task Task
		_ = json.NewDecoder(r.Body).Decode(&task)
		task.ID = "task-minimal"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	created, err := SubmitTask(context.Background(), client, "Simple task", "", nil, PriorityLow)
	require.NoError(t, err)
	assert.Equal(t, "task-minimal", created.ID)
}

func TestSubmitTask_Bad_EmptyTitle(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	created, err := SubmitTask(context.Background(), client, "", "description", nil, PriorityMedium)
	assert.Error(t, err)
	assert.Nil(t, created)
	assert.Contains(t, err.Error(), "title is required")
}

func TestSubmitTask_Bad_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(APIError{Message: "internal error"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	created, err := SubmitTask(context.Background(), client, "Good title", "", nil, PriorityMedium)
	assert.Error(t, err)
	assert.Nil(t, created)
	assert.Contains(t, err.Error(), "create task")
}
