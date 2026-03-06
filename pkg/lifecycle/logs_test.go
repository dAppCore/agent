package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamLogs_Good_CompletedTask(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tasks/task-1", r.URL.Path)
		n := calls.Add(1)

		task := Task{ID: "task-1"}
		switch {
		case n <= 2:
			task.Status = StatusInProgress
		default:
			task.Status = StatusCompleted
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	var buf bytes.Buffer

	err := StreamLogs(context.Background(), client, "task-1", 10*time.Millisecond, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Status: in_progress")
	assert.Contains(t, output, "Status: completed")
	assert.GreaterOrEqual(t, int(calls.Load()), 3)
}

func TestStreamLogs_Good_BlockedTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		task := Task{ID: "task-2", Status: StatusBlocked}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	var buf bytes.Buffer

	err := StreamLogs(context.Background(), client, "task-2", 10*time.Millisecond, &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Status: blocked")
}

func TestStreamLogs_Good_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		task := Task{ID: "task-3", Status: StatusInProgress}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	var buf bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	err := StreamLogs(ctx, client, "task-3", 20*time.Millisecond, &buf)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, buf.String(), "Status: in_progress")
}

func TestStreamLogs_Good_TransientErrorContinues(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)

		if n == 1 {
			// First call: server error.
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(APIError{Message: "transient"})
			return
		}
		// Second call: completed.
		task := Task{ID: "task-4", Status: StatusCompleted}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	var buf bytes.Buffer

	err := StreamLogs(context.Background(), client, "task-4", 10*time.Millisecond, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Error:")
	assert.Contains(t, output, "Status: completed")
}

func TestStreamLogs_Bad_EmptyTaskID(t *testing.T) {
	client := NewClient("https://api.example.com", "test-token")
	var buf bytes.Buffer

	err := StreamLogs(context.Background(), client, "", 10*time.Millisecond, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task ID is required")
}

func TestStreamLogs_Good_ImmediateCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		task := Task{ID: "task-5", Status: StatusInProgress}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	var buf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := StreamLogs(ctx, client, "task-5", 10*time.Millisecond, &buf)
	require.ErrorIs(t, err, context.Canceled)
}
