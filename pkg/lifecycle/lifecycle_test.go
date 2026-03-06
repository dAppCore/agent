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

// TestTaskLifecycle_ClaimProcessComplete tests the full task lifecycle:
// claim a pending task, check allowance, record usage events, complete the task.
func TestTaskLifecycle_ClaimProcessComplete(t *testing.T) {
	// Set up allowance infrastructure.
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "lifecycle-agent",
		DailyTokenLimit: 100000,
		DailyJobLimit:   10,
		ConcurrentJobs:  3,
	})

	// Phase 1: Pre-dispatch allowance check should pass.
	check, err := svc.Check("lifecycle-agent", "")
	require.NoError(t, err)
	assert.True(t, check.Allowed)
	assert.Equal(t, AllowanceOK, check.Status)

	// Phase 2: Simulate claiming a task via the HTTP client.
	pendingTask := Task{
		ID:       "lifecycle-001",
		Title:    "Full lifecycle test",
		Priority: PriorityHigh,
		Status:   StatusPending,
		Project:  "core",
	}

	claimedTask := pendingTask
	claimedTask.Status = StatusInProgress
	claimedTask.ClaimedBy = "lifecycle-agent"
	now := time.Now().UTC()
	claimedTask.ClaimedAt = &now

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/tasks/lifecycle-001/claim":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ClaimResponse{Task: &claimedTask})

		case r.Method == http.MethodPatch && r.URL.Path == "/api/tasks/lifecycle-001":
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPost && r.URL.Path == "/api/tasks/lifecycle-001/complete":
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	client.AgentID = "lifecycle-agent"

	// Claim the task.
	claimed, err := client.ClaimTask(context.Background(), "lifecycle-001")
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, claimed.Status)
	assert.Equal(t, "lifecycle-agent", claimed.ClaimedBy)

	// Phase 3: Record job start in the allowance system.
	err = svc.RecordUsage(UsageReport{
		AgentID: "lifecycle-agent",
		JobID:   "lifecycle-001",
		Event:   QuotaEventJobStarted,
	})
	require.NoError(t, err)

	usage, _ := store.GetUsage("lifecycle-agent")
	assert.Equal(t, 1, usage.ActiveJobs)
	assert.Equal(t, 1, usage.JobsStarted)

	// Phase 4: Update task progress.
	err = client.UpdateTask(context.Background(), "lifecycle-001", TaskUpdate{
		Status:   StatusInProgress,
		Progress: 50,
		Notes:    "Halfway through",
	})
	require.NoError(t, err)

	// Phase 5: Record job completion with token usage.
	err = svc.RecordUsage(UsageReport{
		AgentID:   "lifecycle-agent",
		JobID:     "lifecycle-001",
		Model:     "claude-sonnet",
		TokensIn:  5000,
		TokensOut: 3000,
		Event:     QuotaEventJobCompleted,
	})
	require.NoError(t, err)

	usage, _ = store.GetUsage("lifecycle-agent")
	assert.Equal(t, 0, usage.ActiveJobs)
	assert.Equal(t, int64(8000), usage.TokensUsed)

	// Phase 6: Complete the task via the API.
	err = client.CompleteTask(context.Background(), "lifecycle-001", TaskResult{
		Success:   true,
		Output:    "Task completed successfully",
		Artifacts: []string{"output.go"},
	})
	require.NoError(t, err)

	// Phase 7: Verify allowance is still within limits.
	check, err = svc.Check("lifecycle-agent", "")
	require.NoError(t, err)
	assert.True(t, check.Allowed)
	assert.Equal(t, AllowanceOK, check.Status)
	assert.Equal(t, int64(92000), check.RemainingTokens)
	assert.Equal(t, 9, check.RemainingJobs)
}

// TestTaskLifecycle_ClaimProcessFail tests the lifecycle when a job fails
// and verifies that 50% of tokens are returned.
func TestTaskLifecycle_ClaimProcessFail(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "fail-agent",
		DailyTokenLimit: 50000,
		DailyJobLimit:   5,
		ConcurrentJobs:  2,
	})

	// Start job.
	err := svc.RecordUsage(UsageReport{
		AgentID: "fail-agent",
		JobID:   "fail-001",
		Event:   QuotaEventJobStarted,
	})
	require.NoError(t, err)

	// Job fails with 10000 tokens consumed.
	err = svc.RecordUsage(UsageReport{
		AgentID:   "fail-agent",
		JobID:     "fail-001",
		Model:     "claude-sonnet",
		TokensIn:  6000,
		TokensOut: 4000,
		Event:     QuotaEventJobFailed,
	})
	require.NoError(t, err)

	// Verify 50% returned: 10000 charged, 5000 returned = 5000 net.
	usage, _ := store.GetUsage("fail-agent")
	assert.Equal(t, int64(5000), usage.TokensUsed)
	assert.Equal(t, 0, usage.ActiveJobs)

	// Verify model usage is net: 10000 - 5000 = 5000.
	modelUsage, _ := store.GetModelUsage("claude-sonnet")
	assert.Equal(t, int64(5000), modelUsage)

	// Check allowance - should still have room.
	check, err := svc.Check("fail-agent", "")
	require.NoError(t, err)
	assert.True(t, check.Allowed)
	assert.Equal(t, int64(45000), check.RemainingTokens)
}

// TestTaskLifecycle_ClaimProcessCancel tests the lifecycle when a job is
// cancelled and verifies that 100% of tokens are returned.
func TestTaskLifecycle_ClaimProcessCancel(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "cancel-agent",
		DailyTokenLimit: 50000,
		DailyJobLimit:   5,
		ConcurrentJobs:  2,
	})

	// Start job.
	err := svc.RecordUsage(UsageReport{
		AgentID: "cancel-agent",
		JobID:   "cancel-001",
		Event:   QuotaEventJobStarted,
	})
	require.NoError(t, err)

	// Job cancelled with 8000 tokens consumed.
	err = svc.RecordUsage(UsageReport{
		AgentID:   "cancel-agent",
		JobID:     "cancel-001",
		TokensIn:  5000,
		TokensOut: 3000,
		Event:     QuotaEventJobCancelled,
	})
	require.NoError(t, err)

	// Verify 100% returned: tokens should be 0 (only job start had 0 tokens).
	usage, _ := store.GetUsage("cancel-agent")
	assert.Equal(t, int64(0), usage.TokensUsed)
	assert.Equal(t, 0, usage.ActiveJobs)

	// Model usage should be zero for cancelled jobs.
	modelUsage, _ := store.GetModelUsage("claude-sonnet")
	assert.Equal(t, int64(0), modelUsage)
}

// TestTaskLifecycle_MultipleAgentsConcurrent verifies that multiple agents
// can operate on the same store concurrently without data races.
func TestTaskLifecycle_MultipleAgentsConcurrent(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	agents := []string{"agent-a", "agent-b", "agent-c"}
	for _, agentID := range agents {
		_ = store.SetAllowance(&AgentAllowance{
			AgentID:         agentID,
			DailyTokenLimit: 100000,
			DailyJobLimit:   50,
			ConcurrentJobs:  5,
		})
	}

	var wg sync.WaitGroup
	for _, agentID := range agents {
		wg.Add(1)
		go func(aid string) {
			defer wg.Done()
			for range 10 {
				// Check allowance.
				result, err := svc.Check(aid, "")
				assert.NoError(t, err)
				assert.True(t, result.Allowed)

				// Start job.
				_ = svc.RecordUsage(UsageReport{
					AgentID: aid,
					JobID:   aid + "-job",
					Event:   QuotaEventJobStarted,
				})

				// Complete job.
				_ = svc.RecordUsage(UsageReport{
					AgentID:   aid,
					JobID:     aid + "-job",
					Model:     "claude-sonnet",
					TokensIn:  100,
					TokensOut: 50,
					Event:     QuotaEventJobCompleted,
				})
			}
		}(agentID)
	}
	wg.Wait()

	// Verify each agent has consistent usage.
	for _, agentID := range agents {
		usage, err := store.GetUsage(agentID)
		require.NoError(t, err)
		assert.Equal(t, int64(1500), usage.TokensUsed) // 10 jobs x 150 tokens
		assert.Equal(t, 10, usage.JobsStarted)         // 10 starts
		assert.Equal(t, 0, usage.ActiveJobs)           // all completed
	}
}

// TestTaskLifecycle_ClaimedByFilter verifies that ListTasks with ClaimedBy
// filter sends the correct query parameter.
func TestTaskLifecycle_ClaimedByFilter(t *testing.T) {
	claimedTask := Task{
		ID:        "claimed-task-1",
		Title:     "Agent's task",
		Status:    StatusInProgress,
		ClaimedBy: "agent-x",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "agent-x", r.URL.Query().Get("claimed_by"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Task{claimedTask})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	tasks, err := client.ListTasks(context.Background(), ListOptions{
		ClaimedBy: "agent-x",
	})

	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "agent-x", tasks[0].ClaimedBy)
}
