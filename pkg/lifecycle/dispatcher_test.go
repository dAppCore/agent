package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDispatcher creates a Dispatcher with a memory registry, default router,
// and memory allowance store, pre-loaded with agents and allowances.
func setupDispatcher(t *testing.T, client *Client) (*Dispatcher, *MemoryRegistry, *MemoryStore) {
	t.Helper()

	reg := NewMemoryRegistry()
	router := NewDefaultRouter()
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	d := NewDispatcher(reg, router, svc, client)
	return d, reg, store
}

func registerAgent(t *testing.T, reg *MemoryRegistry, store *MemoryStore, id string, caps []string, maxLoad int) {
	t.Helper()
	_ = reg.Register(AgentInfo{
		ID:            id,
		Name:          id,
		Capabilities:  caps,
		Status:        AgentAvailable,
		LastHeartbeat: time.Now().UTC(),
		MaxLoad:       maxLoad,
	})
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         id,
		DailyTokenLimit: 100000,
		DailyJobLimit:   50,
		ConcurrentJobs:  5,
	})
}

// --- Dispatch tests ---

func TestDispatcher_Dispatch_Good_NilClient(t *testing.T) {
	d, reg, store := setupDispatcher(t, nil)
	registerAgent(t, reg, store, "agent-1", []string{"go"}, 5)

	task := &Task{ID: "task-1", Labels: []string{"go"}, Priority: PriorityMedium}
	agentID, err := d.Dispatch(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", agentID)

	// Verify usage was recorded.
	usage, _ := store.GetUsage("agent-1")
	assert.Equal(t, 1, usage.JobsStarted)
	assert.Equal(t, 1, usage.ActiveJobs)
}

func TestDispatcher_Dispatch_Good_WithHTTPClient(t *testing.T) {
	claimedTask := Task{ID: "task-1", Status: StatusInProgress, ClaimedBy: "agent-1"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/tasks/task-1/claim" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ClaimResponse{Task: &claimedTask})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	d, reg, store := setupDispatcher(t, client)
	registerAgent(t, reg, store, "agent-1", nil, 5)

	task := &Task{ID: "task-1", Priority: PriorityHigh}
	agentID, err := d.Dispatch(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", agentID)

	// Verify usage recorded.
	usage, _ := store.GetUsage("agent-1")
	assert.Equal(t, 1, usage.JobsStarted)
}

func TestDispatcher_Dispatch_Good_PicksBestAgent(t *testing.T) {
	d, reg, store := setupDispatcher(t, nil)
	registerAgent(t, reg, store, "heavy", []string{"go"}, 5)
	registerAgent(t, reg, store, "light", []string{"go"}, 5)

	// Give "heavy" some load.
	_ = reg.Register(AgentInfo{
		ID:            "heavy",
		Name:          "heavy",
		Capabilities:  []string{"go"},
		Status:        AgentAvailable,
		LastHeartbeat: time.Now().UTC(),
		CurrentLoad:   4,
		MaxLoad:       5,
	})

	task := &Task{ID: "task-1", Labels: []string{"go"}, Priority: PriorityMedium}
	agentID, err := d.Dispatch(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "light", agentID) // light has score 1.0, heavy has 0.2
}

func TestDispatcher_Dispatch_Bad_NoAgents(t *testing.T) {
	d, _, _ := setupDispatcher(t, nil)

	task := &Task{ID: "task-1", Priority: PriorityMedium}
	_, err := d.Dispatch(context.Background(), task)
	require.Error(t, err)
}

func TestDispatcher_Dispatch_Bad_AllowanceExceeded(t *testing.T) {
	d, reg, store := setupDispatcher(t, nil)
	registerAgent(t, reg, store, "agent-1", nil, 5)

	// Exhaust the agent's daily job limit.
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:       "agent-1",
		DailyJobLimit: 1,
	})
	_ = store.IncrementUsage("agent-1", 0, 1)

	task := &Task{ID: "task-1", Priority: PriorityMedium}
	_, err := d.Dispatch(context.Background(), task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestDispatcher_Dispatch_Bad_ClaimFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(APIError{Code: 409, Message: "already claimed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	d, reg, store := setupDispatcher(t, client)
	registerAgent(t, reg, store, "agent-1", nil, 5)

	task := &Task{ID: "task-1", Priority: PriorityMedium}
	_, err := d.Dispatch(context.Background(), task)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "claim task")

	// Verify usage was NOT recorded when claim fails.
	usage, _ := store.GetUsage("agent-1")
	assert.Equal(t, 0, usage.JobsStarted)
}

// --- DispatchLoop tests ---

func TestDispatcher_DispatchLoop_Good_Cancellation(t *testing.T) {
	d, _, _ := setupDispatcher(t, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := d.DispatchLoop(ctx, 100*time.Millisecond)
	require.ErrorIs(t, err, context.Canceled)
}

func TestDispatcher_DispatchLoop_Good_DispatchesPendingTasks(t *testing.T) {
	pendingTasks := []Task{
		{ID: "task-1", Status: StatusPending, Priority: PriorityMedium},
		{ID: "task-2", Status: StatusPending, Priority: PriorityHigh},
	}

	var mu sync.Mutex
	claimedIDs := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tasks":
			w.Header().Set("Content-Type", "application/json")
			mu.Lock()
			// Return only tasks not yet claimed.
			var remaining []Task
			for _, t := range pendingTasks {
				if !claimedIDs[t.ID] {
					remaining = append(remaining, t)
				}
			}
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(remaining)

		case r.Method == http.MethodPost:
			// Extract task ID from claim URL.
			w.Header().Set("Content-Type", "application/json")
			// Parse the task ID from the path.
			for _, t := range pendingTasks {
				if r.URL.Path == "/api/tasks/"+t.ID+"/claim" {
					mu.Lock()
					claimedIDs[t.ID] = true
					mu.Unlock()
					claimed := t
					claimed.Status = StatusInProgress
					_ = json.NewEncoder(w).Encode(ClaimResponse{Task: &claimed})
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	d, reg, store := setupDispatcher(t, client)
	registerAgent(t, reg, store, "agent-1", nil, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := d.DispatchLoop(ctx, 50*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify tasks were claimed.
	mu.Lock()
	defer mu.Unlock()
	assert.True(t, claimedIDs["task-1"])
	assert.True(t, claimedIDs["task-2"])
}

func TestDispatcher_DispatchLoop_Good_NilClientSkipsTick(t *testing.T) {
	d, _, _ := setupDispatcher(t, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err := d.DispatchLoop(ctx, 50*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// No panics — nil client is handled gracefully.
}

// --- Concurrent dispatch ---

func TestDispatcher_Dispatch_Good_Concurrent(t *testing.T) {
	d, reg, store := setupDispatcher(t, nil)
	registerAgent(t, reg, store, "agent-1", nil, 0)
	// Override allowance to truly unlimited (registerAgent hardcodes ConcurrentJobs: 5)
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "agent-1",
		DailyJobLimit:  100,
		ConcurrentJobs: 0, // 0 = unlimited
	})

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			task := &Task{ID: "task-" + string(rune('a'+n)), Priority: PriorityMedium}
			_, _ = d.Dispatch(context.Background(), task)
		}(i)
	}
	wg.Wait()

	// Verify usage was recorded for all dispatches.
	usage, _ := store.GetUsage("agent-1")
	assert.Equal(t, 10, usage.JobsStarted)
}

// --- Phase 7: Priority sorting tests ---

func TestSortTasksByPriority_Good(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tasks := []Task{
		{ID: "low-old", Priority: PriorityLow, CreatedAt: base},
		{ID: "critical-new", Priority: PriorityCritical, CreatedAt: base.Add(2 * time.Hour)},
		{ID: "medium-old", Priority: PriorityMedium, CreatedAt: base},
		{ID: "high-old", Priority: PriorityHigh, CreatedAt: base},
		{ID: "critical-old", Priority: PriorityCritical, CreatedAt: base},
	}

	sortTasksByPriority(tasks)

	// Critical tasks first, oldest critical before newer critical.
	assert.Equal(t, "critical-old", tasks[0].ID)
	assert.Equal(t, "critical-new", tasks[1].ID)
	// Then high.
	assert.Equal(t, "high-old", tasks[2].ID)
	// Then medium.
	assert.Equal(t, "medium-old", tasks[3].ID)
	// Then low.
	assert.Equal(t, "low-old", tasks[4].ID)
}

// --- Phase 7: Backoff duration tests ---

func TestBackoffDuration_Good(t *testing.T) {
	// retryCount=0 → 0 (no backoff).
	assert.Equal(t, time.Duration(0), backoffDuration(0))
	// retryCount=1 → 5s (base).
	assert.Equal(t, 5*time.Second, backoffDuration(1))
	// retryCount=2 → 10s.
	assert.Equal(t, 10*time.Second, backoffDuration(2))
	// retryCount=3 → 20s.
	assert.Equal(t, 20*time.Second, backoffDuration(3))
	// retryCount=4 → 40s.
	assert.Equal(t, 40*time.Second, backoffDuration(4))
}

// --- Phase 7: shouldSkipRetry tests ---

func TestShouldSkipRetry_Good(t *testing.T) {
	now := time.Now().UTC()
	recent := now.Add(-2 * time.Second) // 2s ago, backoff for retry 1 is 5s → skip.

	task := &Task{
		ID:          "task-1",
		RetryCount:  1,
		LastAttempt: &recent,
	}
	assert.True(t, shouldSkipRetry(task, now))

	// After backoff elapses, should NOT skip.
	old := now.Add(-10 * time.Second) // 10s ago, backoff for retry 1 is 5s → ready.
	task.LastAttempt = &old
	assert.False(t, shouldSkipRetry(task, now))
}

func TestShouldSkipRetry_Bad_NoRetry(t *testing.T) {
	now := time.Now().UTC()

	// RetryCount=0 → never skip.
	task := &Task{ID: "task-1", RetryCount: 0}
	assert.False(t, shouldSkipRetry(task, now))

	// RetryCount=0 even with a LastAttempt set → never skip.
	recent := now.Add(-1 * time.Second)
	task.LastAttempt = &recent
	assert.False(t, shouldSkipRetry(task, now))

	// RetryCount>0 but nil LastAttempt → never skip.
	task2 := &Task{ID: "task-2", RetryCount: 2, LastAttempt: nil}
	assert.False(t, shouldSkipRetry(task2, now))
}

// --- Phase 7: DispatchLoop priority order test ---

func TestDispatcher_DispatchLoop_Good_PriorityOrder(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	pendingTasks := []Task{
		{ID: "low-1", Status: StatusPending, Priority: PriorityLow, CreatedAt: base},
		{ID: "critical-1", Status: StatusPending, Priority: PriorityCritical, CreatedAt: base},
		{ID: "medium-1", Status: StatusPending, Priority: PriorityMedium, CreatedAt: base},
		{ID: "high-1", Status: StatusPending, Priority: PriorityHigh, CreatedAt: base},
		{ID: "critical-2", Status: StatusPending, Priority: PriorityCritical, CreatedAt: base.Add(time.Second)},
	}

	var mu sync.Mutex
	var claimOrder []string
	claimedIDs := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tasks":
			w.Header().Set("Content-Type", "application/json")
			mu.Lock()
			var remaining []Task
			for _, tk := range pendingTasks {
				if !claimedIDs[tk.ID] {
					remaining = append(remaining, tk)
				}
			}
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(remaining)

		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			for _, tk := range pendingTasks {
				if r.URL.Path == "/api/tasks/"+tk.ID+"/claim" {
					mu.Lock()
					claimedIDs[tk.ID] = true
					claimOrder = append(claimOrder, tk.ID)
					mu.Unlock()
					claimed := tk
					claimed.Status = StatusInProgress
					_ = json.NewEncoder(w).Encode(ClaimResponse{Task: &claimed})
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	d, reg, store := setupDispatcher(t, client)
	registerAgent(t, reg, store, "agent-1", nil, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := d.DispatchLoop(ctx, 50*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	mu.Lock()
	defer mu.Unlock()

	// All 5 tasks should have been claimed.
	require.Len(t, claimOrder, 5)
	// Critical tasks first (oldest before newest), then high, medium, low.
	assert.Equal(t, "critical-1", claimOrder[0])
	assert.Equal(t, "critical-2", claimOrder[1])
	assert.Equal(t, "high-1", claimOrder[2])
	assert.Equal(t, "medium-1", claimOrder[3])
	assert.Equal(t, "low-1", claimOrder[4])
}

// --- Phase 7: DispatchLoop retry backoff test ---

func TestDispatcher_DispatchLoop_Good_RetryBackoff(t *testing.T) {
	// A task with a recent LastAttempt and RetryCount=1 should be skipped
	// because the backoff period (5s) has not elapsed.
	recentAttempt := time.Now().UTC()
	pendingTasks := []Task{
		{
			ID:          "retrying-task",
			Status:      StatusPending,
			Priority:    PriorityHigh,
			CreatedAt:   time.Now().UTC().Add(-time.Hour),
			RetryCount:  1,
			LastAttempt: &recentAttempt,
		},
		{
			ID:        "fresh-task",
			Status:    StatusPending,
			Priority:  PriorityLow,
			CreatedAt: time.Now().UTC(),
		},
	}

	var mu sync.Mutex
	claimedIDs := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tasks":
			w.Header().Set("Content-Type", "application/json")
			mu.Lock()
			var remaining []Task
			for _, tk := range pendingTasks {
				if !claimedIDs[tk.ID] {
					remaining = append(remaining, tk)
				}
			}
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(remaining)

		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			for _, tk := range pendingTasks {
				if r.URL.Path == "/api/tasks/"+tk.ID+"/claim" {
					mu.Lock()
					claimedIDs[tk.ID] = true
					mu.Unlock()
					claimed := tk
					claimed.Status = StatusInProgress
					_ = json.NewEncoder(w).Encode(ClaimResponse{Task: &claimed})
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	d, reg, store := setupDispatcher(t, client)
	registerAgent(t, reg, store, "agent-1", nil, 10)

	// Run the loop for a short period — not long enough for the 5s backoff to elapse.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	err := d.DispatchLoop(ctx, 50*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	mu.Lock()
	defer mu.Unlock()

	// The fresh task should have been claimed.
	assert.True(t, claimedIDs["fresh-task"])
	// The retrying task should NOT have been claimed because backoff has not elapsed.
	assert.False(t, claimedIDs["retrying-task"])
}

// --- Phase 7: DispatchLoop dead-letter test ---

func TestDispatcher_DispatchLoop_Good_DeadLetter(t *testing.T) {
	// A task with RetryCount at MaxRetries-1 that fails dispatch should be dead-lettered.
	pendingTasks := []Task{
		{
			ID:         "doomed-task",
			Status:     StatusPending,
			Priority:   PriorityHigh,
			CreatedAt:  time.Now().UTC().Add(-time.Hour),
			MaxRetries: 1, // Will fail after 1 attempt.
			RetryCount: 0,
		},
	}

	var mu sync.Mutex
	var deadLettered bool
	var deadLetterNotes string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tasks":
			w.Header().Set("Content-Type", "application/json")
			mu.Lock()
			done := deadLettered
			mu.Unlock()
			if done {
				// Return empty list once dead-lettered.
				_ = json.NewEncoder(w).Encode([]Task{})
			} else {
				_ = json.NewEncoder(w).Encode(pendingTasks)
			}

		case r.Method == http.MethodPost && r.URL.Path == "/api/tasks/doomed-task/claim":
			// Claim always fails to trigger retry logic.
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(APIError{Code: 500, Message: "server error"})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/tasks/doomed-task":
			// This is the UpdateTask call for dead-lettering.
			var update TaskUpdate
			_ = json.NewDecoder(r.Body).Decode(&update)
			mu.Lock()
			deadLettered = true
			deadLetterNotes = update.Notes
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	d, reg, store := setupDispatcher(t, client)
	registerAgent(t, reg, store, "agent-1", nil, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := d.DispatchLoop(ctx, 50*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	mu.Lock()
	defer mu.Unlock()

	assert.True(t, deadLettered, "task should have been dead-lettered")
	assert.Equal(t, "max retries exceeded", deadLetterNotes)
}
