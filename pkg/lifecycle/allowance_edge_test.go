package lifecycle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Allowance exhaustion edge cases ---

func TestAllowanceExhaustion_ExactlyAtTokenLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "edge-agent",
		DailyTokenLimit: 10000,
	})
	// Use exactly the limit.
	_ = store.IncrementUsage("edge-agent", 10000, 0)

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed, "should be denied at exactly the limit")
	assert.Equal(t, AllowanceExceeded, result.Status)
	assert.Equal(t, int64(0), result.RemainingTokens)
	assert.Contains(t, result.Reason, "daily token limit exceeded")
}

func TestAllowanceExhaustion_OneOverTokenLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "edge-agent",
		DailyTokenLimit: 10000,
	})
	_ = store.IncrementUsage("edge-agent", 10001, 0)

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, AllowanceExceeded, result.Status)
	assert.True(t, result.RemainingTokens < 0, "remaining should be negative")
}

func TestAllowanceExhaustion_OneUnderTokenLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "edge-agent",
		DailyTokenLimit: 10000,
	})
	_ = store.IncrementUsage("edge-agent", 9999, 0)

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed, "should be allowed with 1 token remaining")
	assert.Equal(t, AllowanceWarning, result.Status, "99.99% usage should be warning")
	assert.Equal(t, int64(1), result.RemainingTokens)
}

func TestAllowanceExhaustion_ZeroAllowance(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	// DailyTokenLimit=0 means unlimited.
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "unlimited-agent",
		DailyTokenLimit: 0,
		DailyJobLimit:   0,
		ConcurrentJobs:  0,
	})
	_ = store.IncrementUsage("unlimited-agent", 999999999, 999)

	result, err := svc.Check("unlimited-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed, "unlimited agent should always be allowed")
	assert.Equal(t, AllowanceOK, result.Status)
	assert.Equal(t, int64(-1), result.RemainingTokens, "unlimited should show -1")
	assert.Equal(t, -1, result.RemainingJobs, "unlimited should show -1")
}

func TestAllowanceExhaustion_ExactlyAtJobLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:       "edge-agent",
		DailyJobLimit: 5,
	})
	_ = store.IncrementUsage("edge-agent", 0, 5)

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed, "should be denied at exactly the job limit")
	assert.Equal(t, AllowanceExceeded, result.Status)
	assert.Equal(t, 0, result.RemainingJobs)
	assert.Contains(t, result.Reason, "daily job limit exceeded")
}

func TestAllowanceExhaustion_OneUnderJobLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:       "edge-agent",
		DailyJobLimit: 5,
	})
	_ = store.IncrementUsage("edge-agent", 0, 4)

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed, "should be allowed with 1 job remaining")
	assert.Equal(t, 1, result.RemainingJobs)
}

func TestAllowanceExhaustion_ConcurrentJobsExactlyAtLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "edge-agent",
		ConcurrentJobs: 2,
	})
	// Start 2 concurrent jobs.
	_ = store.IncrementUsage("edge-agent", 0, 2)

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed, "should be denied at concurrent limit")
	assert.Contains(t, result.Reason, "concurrent job limit reached")
}

func TestAllowanceExhaustion_ConcurrentJobsOneUnderLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "edge-agent",
		ConcurrentJobs: 3,
	})
	_ = store.IncrementUsage("edge-agent", 0, 2)

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed, "should be allowed with 1 concurrent slot remaining")
}

func TestAllowanceExhaustion_ConcurrentJobsFreedByCompletion(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "edge-agent",
		ConcurrentJobs: 1,
	})

	// Start a job - fills the slot.
	_ = svc.RecordUsage(UsageReport{
		AgentID: "edge-agent",
		JobID:   "job-1",
		Event:   QuotaEventJobStarted,
	})

	result, err := svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed, "should be denied, 1 active job")

	// Complete the job - frees the slot.
	_ = svc.RecordUsage(UsageReport{
		AgentID:   "edge-agent",
		JobID:     "job-1",
		TokensIn:  100,
		TokensOut: 50,
		Event:     QuotaEventJobCompleted,
	})

	result, err = svc.Check("edge-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed, "should be allowed after job completes")
}

func TestAllowanceExhaustion_TokenWarningThreshold(t *testing.T) {
	tests := []struct {
		name           string
		limit          int64
		used           int64
		expectedStatus AllowanceStatus
		expectedAllow  bool
	}{
		{
			name:           "79% usage is OK",
			limit:          10000,
			used:           7900,
			expectedStatus: AllowanceOK,
			expectedAllow:  true,
		},
		{
			name:           "80% usage is warning",
			limit:          10000,
			used:           8000,
			expectedStatus: AllowanceWarning,
			expectedAllow:  true,
		},
		{
			name:           "90% usage is warning",
			limit:          10000,
			used:           9000,
			expectedStatus: AllowanceWarning,
			expectedAllow:  true,
		},
		{
			name:           "99% usage is warning",
			limit:          10000,
			used:           9999,
			expectedStatus: AllowanceWarning,
			expectedAllow:  true,
		},
		{
			name:           "100% usage is exceeded",
			limit:          10000,
			used:           10000,
			expectedStatus: AllowanceExceeded,
			expectedAllow:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()
			svc := NewAllowanceService(store)

			_ = store.SetAllowance(&AgentAllowance{
				AgentID:         "threshold-agent",
				DailyTokenLimit: tt.limit,
			})
			_ = store.IncrementUsage("threshold-agent", tt.used, 0)

			result, err := svc.Check("threshold-agent", "")
			require.NoError(t, err)
			assert.Equal(t, tt.expectedAllow, result.Allowed)
			assert.Equal(t, tt.expectedStatus, result.Status)
		})
	}
}

func TestAllowanceExhaustion_ResetRestoresCapacity(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "reset-agent",
		DailyTokenLimit: 10000,
		DailyJobLimit:   5,
	})

	// Exhaust all limits.
	_ = store.IncrementUsage("reset-agent", 10000, 5)

	result, err := svc.Check("reset-agent", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed, "should be denied when exhausted")

	// Reset the agent (simulates midnight reset).
	err = svc.ResetAgent("reset-agent")
	require.NoError(t, err)

	result, err = svc.Check("reset-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed, "should be allowed after reset")
	assert.Equal(t, int64(10000), result.RemainingTokens)
	assert.Equal(t, 5, result.RemainingJobs)
}

func TestAllowanceExhaustion_GlobalModelBudgetExactlyAtLimit(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID: "model-edge-agent",
	})
	store.SetModelQuota(&ModelQuota{
		Model:            "claude-opus-4-6",
		DailyTokenBudget: 50000,
	})
	_ = store.IncrementModelUsage("claude-opus-4-6", 50000)

	result, err := svc.Check("model-edge-agent", "claude-opus-4-6")
	require.NoError(t, err)
	assert.False(t, result.Allowed, "should be denied at exact model budget")
	assert.Contains(t, result.Reason, "global model token budget exceeded")
}

func TestAllowanceExhaustion_GlobalModelBudgetOneUnder(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID: "model-edge-agent",
	})
	store.SetModelQuota(&ModelQuota{
		Model:            "claude-opus-4-6",
		DailyTokenBudget: 50000,
	})
	_ = store.IncrementModelUsage("claude-opus-4-6", 49999)

	result, err := svc.Check("model-edge-agent", "claude-opus-4-6")
	require.NoError(t, err)
	assert.True(t, result.Allowed, "should be allowed with 1 token remaining in model budget")
}

func TestAllowanceExhaustion_FailedJobWithZeroTokens(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = svc.RecordUsage(UsageReport{
		AgentID: "zero-token-agent",
		JobID:   "job-1",
		Event:   QuotaEventJobStarted,
	})

	// Job fails but consumed zero tokens.
	err := svc.RecordUsage(UsageReport{
		AgentID:   "zero-token-agent",
		JobID:     "job-1",
		Model:     "claude-sonnet",
		TokensIn:  0,
		TokensOut: 0,
		Event:     QuotaEventJobFailed,
	})
	require.NoError(t, err)

	usage, _ := store.GetUsage("zero-token-agent")
	assert.Equal(t, int64(0), usage.TokensUsed, "no tokens should be charged")
	assert.Equal(t, 0, usage.ActiveJobs)

	// Model usage should be zero too (50% of 0 = 0).
	modelUsage, _ := store.GetModelUsage("claude-sonnet")
	assert.Equal(t, int64(0), modelUsage)
}

func TestAllowanceExhaustion_CancelledJobWithZeroTokens(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = svc.RecordUsage(UsageReport{
		AgentID: "zero-token-agent",
		JobID:   "job-2",
		Event:   QuotaEventJobStarted,
	})

	// Job cancelled with zero tokens.
	err := svc.RecordUsage(UsageReport{
		AgentID:   "zero-token-agent",
		JobID:     "job-2",
		TokensIn:  0,
		TokensOut: 0,
		Event:     QuotaEventJobCancelled,
	})
	require.NoError(t, err)

	usage, _ := store.GetUsage("zero-token-agent")
	assert.Equal(t, int64(0), usage.TokensUsed)
	assert.Equal(t, 0, usage.ActiveJobs)
}

func TestAllowanceExhaustion_CompletedJobWithNoModel(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = svc.RecordUsage(UsageReport{
		AgentID: "no-model-agent",
		JobID:   "job-1",
		Event:   QuotaEventJobStarted,
	})

	// Complete with empty model -- should skip model-level usage recording.
	err := svc.RecordUsage(UsageReport{
		AgentID:   "no-model-agent",
		JobID:     "job-1",
		Model:     "",
		TokensIn:  500,
		TokensOut: 200,
		Event:     QuotaEventJobCompleted,
	})
	require.NoError(t, err)

	usage, _ := store.GetUsage("no-model-agent")
	assert.Equal(t, int64(700), usage.TokensUsed)
	assert.Equal(t, 0, usage.ActiveJobs)
}

func TestAllowanceExhaustion_FailedJobWithNoModel(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = svc.RecordUsage(UsageReport{
		AgentID: "no-model-fail-agent",
		JobID:   "job-1",
		Event:   QuotaEventJobStarted,
	})

	// Fail with empty model.
	err := svc.RecordUsage(UsageReport{
		AgentID:   "no-model-fail-agent",
		JobID:     "job-1",
		Model:     "",
		TokensIn:  600,
		TokensOut: 400,
		Event:     QuotaEventJobFailed,
	})
	require.NoError(t, err)

	usage, _ := store.GetUsage("no-model-fail-agent")
	// 1000 tokens used, 500 returned = 500 net.
	assert.Equal(t, int64(500), usage.TokensUsed)
	assert.Equal(t, 0, usage.ActiveJobs)
}

func TestAllowanceExhaustion_MultipleChecksWithIncrementalUsage(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "incremental-agent",
		DailyTokenLimit: 1000,
	})

	// First check: fresh agent.
	result, err := svc.Check("incremental-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, AllowanceOK, result.Status)
	assert.Equal(t, int64(1000), result.RemainingTokens)

	// Use 500 tokens.
	_ = store.IncrementUsage("incremental-agent", 500, 0)

	result, err = svc.Check("incremental-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, AllowanceOK, result.Status)
	assert.Equal(t, int64(500), result.RemainingTokens)

	// Use another 300 tokens (total 800, at 80% threshold).
	_ = store.IncrementUsage("incremental-agent", 300, 0)

	result, err = svc.Check("incremental-agent", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, AllowanceWarning, result.Status)
	assert.Equal(t, int64(200), result.RemainingTokens)

	// Use remaining 200 tokens (total 1000, at 100%).
	_ = store.IncrementUsage("incremental-agent", 200, 0)

	result, err = svc.Check("incremental-agent", "")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, AllowanceExceeded, result.Status)
	assert.Equal(t, int64(0), result.RemainingTokens)
}

// --- MemoryStore additional edge cases ---

func TestMemoryStore_GetUsage_NewAgentReturnsDefaults(t *testing.T) {
	store := NewMemoryStore()

	usage, err := store.GetUsage("brand-new-agent")
	require.NoError(t, err)
	assert.Equal(t, "brand-new-agent", usage.AgentID)
	assert.Equal(t, int64(0), usage.TokensUsed)
	assert.Equal(t, 0, usage.JobsStarted)
	assert.Equal(t, 0, usage.ActiveJobs)
	assert.Equal(t, startOfDay(time.Now().UTC()), usage.PeriodStart)
}

func TestMemoryStore_ReturnTokens_NonexistentAgent(t *testing.T) {
	store := NewMemoryStore()

	// ReturnTokens on a nonexistent agent should be a no-op.
	err := store.ReturnTokens("ghost-agent", 5000)
	require.NoError(t, err)
}

func TestMemoryStore_DecrementActiveJobs_NonexistentAgent(t *testing.T) {
	store := NewMemoryStore()

	// DecrementActiveJobs on a nonexistent agent should be a no-op.
	err := store.DecrementActiveJobs("ghost-agent")
	require.NoError(t, err)
}

func TestMemoryStore_GetModelQuota_NotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.GetModelQuota("nonexistent-model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model quota not found")
}

func TestMemoryStore_GetModelUsage_NewModelReturnsZero(t *testing.T) {
	store := NewMemoryStore()

	usage, err := store.GetModelUsage("brand-new-model")
	require.NoError(t, err)
	assert.Equal(t, int64(0), usage)
}

func TestMemoryStore_SetAllowance_Overwrite(t *testing.T) {
	store := NewMemoryStore()

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "overwrite-agent",
		DailyTokenLimit: 5000,
	})

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "overwrite-agent",
		DailyTokenLimit: 9000,
	})

	a, err := store.GetAllowance("overwrite-agent")
	require.NoError(t, err)
	assert.Equal(t, int64(9000), a.DailyTokenLimit, "should have overwritten the old allowance")
}

func TestMemoryStore_SetAllowance_IsolatesOriginal(t *testing.T) {
	store := NewMemoryStore()

	original := &AgentAllowance{
		AgentID:         "isolated-agent",
		DailyTokenLimit: 5000,
	}
	_ = store.SetAllowance(original)

	// Mutate the original.
	original.DailyTokenLimit = 99999

	got, err := store.GetAllowance("isolated-agent")
	require.NoError(t, err)
	assert.Equal(t, int64(5000), got.DailyTokenLimit, "store should hold a copy, not the original")
}

func TestMemoryStore_GetAllowance_IsolatesReturn(t *testing.T) {
	store := NewMemoryStore()

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "isolated-agent",
		DailyTokenLimit: 5000,
	})

	got1, _ := store.GetAllowance("isolated-agent")
	got1.DailyTokenLimit = 99999

	got2, _ := store.GetAllowance("isolated-agent")
	assert.Equal(t, int64(5000), got2.DailyTokenLimit, "returned value should be a copy")
}

func TestMemoryStore_IncrementUsage_MultipleIncrements(t *testing.T) {
	store := NewMemoryStore()

	_ = store.IncrementUsage("multi-agent", 100, 1)
	_ = store.IncrementUsage("multi-agent", 200, 1)
	_ = store.IncrementUsage("multi-agent", 300, 0)

	usage, err := store.GetUsage("multi-agent")
	require.NoError(t, err)
	assert.Equal(t, int64(600), usage.TokensUsed)
	assert.Equal(t, 2, usage.JobsStarted)
	assert.Equal(t, 2, usage.ActiveJobs)
}

func TestMemoryStore_IncrementUsage_ZeroJobsDoesNotIncrementActive(t *testing.T) {
	store := NewMemoryStore()

	_ = store.IncrementUsage("token-only-agent", 5000, 0)

	usage, err := store.GetUsage("token-only-agent")
	require.NoError(t, err)
	assert.Equal(t, int64(5000), usage.TokensUsed)
	assert.Equal(t, 0, usage.JobsStarted)
	assert.Equal(t, 0, usage.ActiveJobs, "zero jobs should not increment active count")
}

// --- AllowanceService Check priority ordering ---

func TestAllowanceServiceCheck_ModelAllowlistCheckedFirst(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	// Agent is over token limit AND using a disallowed model.
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "order-agent",
		DailyTokenLimit: 1000,
		ModelAllowlist:  []string{"claude-haiku"},
	})
	_ = store.IncrementUsage("order-agent", 2000, 0)

	result, err := svc.Check("order-agent", "claude-opus-4-6")
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	// Model allowlist is checked first in the code, so it should be the reason.
	assert.Contains(t, result.Reason, "model not in allowlist")
}

func TestAllowanceServiceCheck_EmptyModelAllowlistPermitsAll(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = store.SetAllowance(&AgentAllowance{
		AgentID:        "any-model-agent",
		ModelAllowlist: nil,
	})

	result, err := svc.Check("any-model-agent", "any-model-at-all")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
}

// --- QuotaEvent constants ---

func TestQuotaEvent_Values(t *testing.T) {
	assert.Equal(t, QuotaEvent("job_started"), QuotaEventJobStarted)
	assert.Equal(t, QuotaEvent("job_completed"), QuotaEventJobCompleted)
	assert.Equal(t, QuotaEvent("job_failed"), QuotaEventJobFailed)
	assert.Equal(t, QuotaEvent("job_cancelled"), QuotaEventJobCancelled)
}

func TestAllowanceExhaustion_FailedJobWithOddTokenCount(t *testing.T) {
	store := NewMemoryStore()
	svc := NewAllowanceService(store)

	_ = svc.RecordUsage(UsageReport{
		AgentID: "odd-agent",
		JobID:   "job-1",
		Event:   QuotaEventJobStarted,
	})

	// Odd total: 7 tokens. 50% return = 3 (integer division).
	err := svc.RecordUsage(UsageReport{
		AgentID:   "odd-agent",
		JobID:     "job-1",
		Model:     "claude-sonnet",
		TokensIn:  4,
		TokensOut: 3,
		Event:     QuotaEventJobFailed,
	})
	require.NoError(t, err)

	usage, _ := store.GetUsage("odd-agent")
	// 7 charged - 3 returned = 4 net.
	assert.Equal(t, int64(4), usage.TokensUsed)

	// Model gets 7 - 3 = 4.
	modelUsage, _ := store.GetModelUsage("claude-sonnet")
	assert.Equal(t, int64(4), modelUsage)
}
