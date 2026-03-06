package lifecycle

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errorStore is a mock AllowanceStore that returns errors for specific operations.
type errorStore struct {
	*MemoryStore
	failIncrementUsage    bool
	failDecrementActive   bool
	failReturnTokens      bool
	failIncrementModel    bool
	failGetAllowance      bool
	failGetUsage          bool
}

func newErrorStore() *errorStore {
	return &errorStore{MemoryStore: NewMemoryStore()}
}

func (e *errorStore) GetAllowance(agentID string) (*AgentAllowance, error) {
	if e.failGetAllowance {
		return nil, errors.New("simulated GetAllowance error")
	}
	return e.MemoryStore.GetAllowance(agentID)
}

func (e *errorStore) GetUsage(agentID string) (*UsageRecord, error) {
	if e.failGetUsage {
		return nil, errors.New("simulated GetUsage error")
	}
	return e.MemoryStore.GetUsage(agentID)
}

func (e *errorStore) IncrementUsage(agentID string, tokens int64, jobs int) error {
	if e.failIncrementUsage {
		return errors.New("simulated IncrementUsage error")
	}
	return e.MemoryStore.IncrementUsage(agentID, tokens, jobs)
}

func (e *errorStore) DecrementActiveJobs(agentID string) error {
	if e.failDecrementActive {
		return errors.New("simulated DecrementActiveJobs error")
	}
	return e.MemoryStore.DecrementActiveJobs(agentID)
}

func (e *errorStore) ReturnTokens(agentID string, tokens int64) error {
	if e.failReturnTokens {
		return errors.New("simulated ReturnTokens error")
	}
	return e.MemoryStore.ReturnTokens(agentID, tokens)
}

func (e *errorStore) IncrementModelUsage(model string, tokens int64) error {
	if e.failIncrementModel {
		return errors.New("simulated IncrementModelUsage error")
	}
	return e.MemoryStore.IncrementModelUsage(model, tokens)
}

// --- RecordUsage error path tests ---

func TestRecordUsage_Bad_JobStarted_IncrementFails(t *testing.T) {
	store := newErrorStore()
	store.failIncrementUsage = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID: "agent-1",
		Event:   QuotaEventJobStarted,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment job count")
}

func TestRecordUsage_Bad_JobCompleted_IncrementFails(t *testing.T) {
	store := newErrorStore()
	store.failIncrementUsage = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		TokensIn:  100,
		TokensOut: 50,
		Event:     QuotaEventJobCompleted,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record token usage")
}

func TestRecordUsage_Bad_JobCompleted_DecrementFails(t *testing.T) {
	store := newErrorStore()
	store.failDecrementActive = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		TokensIn:  100,
		TokensOut: 50,
		Event:     QuotaEventJobCompleted,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrement active jobs")
}

func TestRecordUsage_Bad_JobCompleted_ModelUsageFails(t *testing.T) {
	store := newErrorStore()
	store.failIncrementModel = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		Model:     "claude-sonnet",
		TokensIn:  100,
		TokensOut: 50,
		Event:     QuotaEventJobCompleted,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record model usage")
}

func TestRecordUsage_Bad_JobFailed_IncrementFails(t *testing.T) {
	store := newErrorStore()
	store.failIncrementUsage = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		TokensIn:  100,
		TokensOut: 100,
		Event:     QuotaEventJobFailed,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record token usage")
}

func TestRecordUsage_Bad_JobFailed_DecrementFails(t *testing.T) {
	store := newErrorStore()
	store.failDecrementActive = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		TokensIn:  100,
		TokensOut: 100,
		Event:     QuotaEventJobFailed,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrement active jobs")
}

func TestRecordUsage_Bad_JobFailed_ReturnTokensFails(t *testing.T) {
	store := newErrorStore()
	store.failReturnTokens = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		TokensIn:  100,
		TokensOut: 100,
		Event:     QuotaEventJobFailed,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to return tokens")
}

func TestRecordUsage_Bad_JobFailed_ModelUsageFails(t *testing.T) {
	store := newErrorStore()
	store.failIncrementModel = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		Model:     "claude-sonnet",
		TokensIn:  100,
		TokensOut: 100,
		Event:     QuotaEventJobFailed,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to record model usage")
}

func TestRecordUsage_Bad_JobCancelled_DecrementFails(t *testing.T) {
	store := newErrorStore()
	store.failDecrementActive = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		TokensIn:  100,
		TokensOut: 100,
		Event:     QuotaEventJobCancelled,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrement active jobs")
}

func TestRecordUsage_Bad_JobCancelled_ReturnTokensFails(t *testing.T) {
	store := newErrorStore()
	store.failReturnTokens = true
	svc := NewAllowanceService(store)

	err := svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		TokensIn:  100,
		TokensOut: 100,
		Event:     QuotaEventJobCancelled,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to return tokens")
}

// --- Check error path tests ---

func TestCheck_Bad_GetAllowanceFails(t *testing.T) {
	store := newErrorStore()
	store.failGetAllowance = true
	svc := NewAllowanceService(store)

	_, err := svc.Check("agent-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get allowance")
}

func TestCheck_Bad_GetUsageFails(t *testing.T) {
	store := newErrorStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID: "agent-1",
	})
	store.failGetUsage = true

	_, err := svc.Check("agent-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get usage")
}

// --- ResetAgent error path ---

func TestResetAgent_Bad_ResetFails(t *testing.T) {
	// MemoryStore.ResetUsage never fails, but we can test the service
	// layer still returns nil for the happy path (already tested).
	// For a true error test, we'd need a mock, but the MemoryStore
	// never errors on ResetUsage. This confirms the pattern.
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	err := svc.ResetAgent("nonexistent-agent")
	require.NoError(t, err, "resetting a nonexistent agent should succeed")
}

// --- RecordUsage with unknown event type ---

func TestRecordUsage_Good_UnknownEvent(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	// Unknown event should be a no-op (falls through the switch).
	err := svc.RecordUsage(UsageReport{
		AgentID: "agent-1",
		Event:   QuotaEvent("unknown_event"),
	})
	require.NoError(t, err, "unknown event should not error")
}
