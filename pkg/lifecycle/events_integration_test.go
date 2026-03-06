package lifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Dispatcher event emission tests ---

func TestDispatcher_EmitsTaskDispatched(t *testing.T) {
	em := NewChannelEmitter(10)
	d, reg, store := setupDispatcher(t, nil)
	d.SetEventEmitter(em)
	registerAgent(t, reg, store, "agent-1", []string{"go"}, 5)

	task := &Task{ID: "t1", Labels: []string{"go"}}
	agentID, err := d.Dispatch(context.Background(), task)
	require.NoError(t, err)
	assert.Equal(t, "agent-1", agentID)

	// Should have received EventTaskDispatched.
	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventTaskDispatched, got[0].Type)
	assert.Equal(t, "t1", got[0].TaskID)
	assert.Equal(t, "agent-1", got[0].AgentID)
}

func TestDispatcher_EmitsDispatchFailedNoAgent(t *testing.T) {
	em := NewChannelEmitter(10)
	d, _, _ := setupDispatcher(t, nil)
	d.SetEventEmitter(em)
	// No agents registered.

	task := &Task{ID: "t2", Labels: []string{"go"}}
	_, err := d.Dispatch(context.Background(), task)
	require.Error(t, err)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventDispatchFailedNoAgent, got[0].Type)
	assert.Equal(t, "t2", got[0].TaskID)
}

func TestDispatcher_EmitsDispatchFailedQuota(t *testing.T) {
	em := NewChannelEmitter(10)
	d, reg, store := setupDispatcher(t, nil)
	d.SetEventEmitter(em)

	// Register agent with zero daily job limit (will be exceeded immediately).
	_ = reg.Register(AgentInfo{
		ID: "agent-q", Name: "agent-q", Capabilities: []string{"go"},
		Status: AgentAvailable, LastHeartbeat: time.Now().UTC(), MaxLoad: 5,
	})
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "agent-q",
		DailyJobLimit:  1,
		ConcurrentJobs: 5,
	})
	// Use up the single job.
	_ = store.IncrementUsage("agent-q", 0, 1)

	task := &Task{ID: "t3", Labels: []string{"go"}}
	_, err := d.Dispatch(context.Background(), task)
	require.Error(t, err)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventDispatchFailedQuota, got[0].Type)
	assert.Equal(t, "t3", got[0].TaskID)
	assert.Equal(t, "agent-q", got[0].AgentID)
}

func TestDispatcher_NoEventsWithoutEmitter(t *testing.T) {
	// Verify no panic when emitter is nil.
	d, reg, store := setupDispatcher(t, nil)
	registerAgent(t, reg, store, "agent-1", []string{"go"}, 5)

	task := &Task{ID: "t4", Labels: []string{"go"}}
	_, err := d.Dispatch(context.Background(), task)
	require.NoError(t, err)
	// No panic = pass.
}

// --- AllowanceService event emission tests ---

func TestAllowanceService_EmitsQuotaExceeded(t *testing.T) {
	em := NewChannelEmitter(10)
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	svc.SetEventEmitter(em)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100,
	})
	// Use all tokens.
	_ = store.IncrementUsage("agent-1", 100, 0)

	result, err := svc.Check("agent-1", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventQuotaExceeded, got[0].Type)
	assert.Equal(t, "agent-1", got[0].AgentID)
}

func TestAllowanceService_EmitsQuotaWarning(t *testing.T) {
	em := NewChannelEmitter(10)
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	svc.SetEventEmitter(em)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100,
	})
	// Use 85% of tokens — should trigger warning.
	_ = store.IncrementUsage("agent-1", 85, 0)

	result, err := svc.Check("agent-1", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, AllowanceWarning, result.Status)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventQuotaWarning, got[0].Type)
	assert.Equal(t, "agent-1", got[0].AgentID)
}

func TestAllowanceService_EmitsUsageRecorded(t *testing.T) {
	em := NewChannelEmitter(10)
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	svc.SetEventEmitter(em)

	_ = store.SetAllowance(&AgentAllowance{AgentID: "agent-1"})

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		JobID:     "job-1",
		Event:     QuotaEventJobStarted,
		Timestamp: time.Now().UTC(),
	})
	require.NoError(t, err)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventUsageRecorded, got[0].Type)
	assert.Equal(t, "agent-1", got[0].AgentID)
}

func TestAllowanceService_EmitsUsageRecordedOnCompletion(t *testing.T) {
	em := NewChannelEmitter(10)
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	svc.SetEventEmitter(em)

	_ = store.SetAllowance(&AgentAllowance{AgentID: "agent-1"})
	// Start a job first.
	_ = store.IncrementUsage("agent-1", 0, 1)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		JobID:     "job-1",
		Model:     "claude-sonnet",
		TokensIn:  500,
		TokensOut: 200,
		Event:     QuotaEventJobCompleted,
		Timestamp: time.Now().UTC(),
	})
	require.NoError(t, err)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventUsageRecorded, got[0].Type)
}

func TestAllowanceService_QuotaExceededOnJobLimit(t *testing.T) {
	em := NewChannelEmitter(10)
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	svc.SetEventEmitter(em)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:       "agent-1",
		DailyJobLimit: 2,
	})
	_ = store.IncrementUsage("agent-1", 0, 2)

	result, err := svc.Check("agent-1", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventQuotaExceeded, got[0].Type)
	assert.Contains(t, got[0].Payload, "daily job limit")
}

func TestAllowanceService_QuotaExceededOnConcurrent(t *testing.T) {
	em := NewChannelEmitter(10)
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	svc.SetEventEmitter(em)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "agent-1",
		ConcurrentJobs: 1,
	})
	_ = store.IncrementUsage("agent-1", 0, 1)

	result, err := svc.Check("agent-1", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventQuotaExceeded, got[0].Type)
	assert.Contains(t, got[0].Payload, "concurrent")
}

func TestAllowanceService_QuotaExceededOnModelAllowlist(t *testing.T) {
	em := NewChannelEmitter(10)
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	svc.SetEventEmitter(em)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "agent-1",
		ModelAllowlist: []string{"claude-sonnet"},
	})

	result, err := svc.Check("agent-1", "gpt-4")
	require.NoError(t, err)
	assert.False(t, result.Allowed)

	got := drainEvents(em, 1, time.Second)
	require.Len(t, got, 1)
	assert.Equal(t, EventQuotaExceeded, got[0].Type)
	assert.Contains(t, got[0].Payload, "allowlist")
}

func TestAllowanceService_NoEventsWithoutEmitter(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)
	// No emitter set.

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100,
	})
	_ = store.IncrementUsage("agent-1", 100, 0)

	result, err := svc.Check("agent-1", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	// No panic = pass.
}

// --- Helpers ---

// drainEvents reads up to n events from the emitter within the timeout.
func drainEvents(em *ChannelEmitter, n int, timeout time.Duration) []Event {
	var events []Event
	deadline := time.After(timeout)
	for range n {
		select {
		case e := <-em.Events():
			events = append(events, e)
		case <-deadline:
			return events
		}
	}
	return events
}
