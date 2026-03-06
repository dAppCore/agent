package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRedisAddr = "10.69.69.87:6379"

// newTestRedisStore creates a RedisStore with a unique prefix for test isolation.
// Skips the test if Redis is unreachable.
func newTestRedisStore(t *testing.T) *RedisStore {
	t.Helper()
	prefix := fmt.Sprintf("test_%d", time.Now().UnixNano())
	s, err := NewRedisStore(testRedisAddr, WithRedisPrefix(prefix))
	if err != nil {
		t.Skipf("Redis unavailable at %s: %v", testRedisAddr, err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_ = s.FlushPrefix(ctx)
		_ = s.Close()
	})
	return s
}

// --- SetAllowance / GetAllowance ---

func TestRedisStore_SetGetAllowance_Good(t *testing.T) {
	s := newTestRedisStore(t)

	a := &AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100000,
		DailyJobLimit:   10,
		ConcurrentJobs:  2,
		MaxJobDuration:  30 * time.Minute,
		ModelAllowlist:  []string{"claude-sonnet-4-5-20250929"},
	}

	err := s.SetAllowance(a)
	require.NoError(t, err)

	got, err := s.GetAllowance("agent-1")
	require.NoError(t, err)
	assert.Equal(t, a.AgentID, got.AgentID)
	assert.Equal(t, a.DailyTokenLimit, got.DailyTokenLimit)
	assert.Equal(t, a.DailyJobLimit, got.DailyJobLimit)
	assert.Equal(t, a.ConcurrentJobs, got.ConcurrentJobs)
	assert.Equal(t, a.MaxJobDuration, got.MaxJobDuration)
	assert.Equal(t, a.ModelAllowlist, got.ModelAllowlist)
}

func TestRedisStore_GetAllowance_Bad_NotFound(t *testing.T) {
	s := newTestRedisStore(t)
	_, err := s.GetAllowance("nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok, "expected *APIError")
	assert.Equal(t, 404, apiErr.Code)
	assert.Contains(t, err.Error(), "allowance not found")
}

func TestRedisStore_SetAllowance_Good_Overwrite(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.SetAllowance(&AgentAllowance{AgentID: "agent-1", DailyTokenLimit: 100})
	_ = s.SetAllowance(&AgentAllowance{AgentID: "agent-1", DailyTokenLimit: 200})

	got, err := s.GetAllowance("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(200), got.DailyTokenLimit)
}

// --- GetUsage / IncrementUsage ---

func TestRedisStore_GetUsage_Good_Default(t *testing.T) {
	s := newTestRedisStore(t)

	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "agent-1", u.AgentID)
	assert.Equal(t, int64(0), u.TokensUsed)
	assert.Equal(t, 0, u.JobsStarted)
	assert.Equal(t, 0, u.ActiveJobs)
}

func TestRedisStore_IncrementUsage_Good(t *testing.T) {
	s := newTestRedisStore(t)

	err := s.IncrementUsage("agent-1", 5000, 1)
	require.NoError(t, err)

	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(5000), u.TokensUsed)
	assert.Equal(t, 1, u.JobsStarted)
	assert.Equal(t, 1, u.ActiveJobs)
}

func TestRedisStore_IncrementUsage_Good_Accumulates(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.IncrementUsage("agent-1", 1000, 1)
	_ = s.IncrementUsage("agent-1", 2000, 1)
	_ = s.IncrementUsage("agent-1", 3000, 0)

	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(6000), u.TokensUsed)
	assert.Equal(t, 2, u.JobsStarted)
	assert.Equal(t, 2, u.ActiveJobs)
}

// --- DecrementActiveJobs ---

func TestRedisStore_DecrementActiveJobs_Good(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.IncrementUsage("agent-1", 0, 2)
	_ = s.DecrementActiveJobs("agent-1")

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, 1, u.ActiveJobs)
}

func TestRedisStore_DecrementActiveJobs_Good_FloorAtZero(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.DecrementActiveJobs("agent-1") // no usage record yet
	_ = s.IncrementUsage("agent-1", 0, 0)
	_ = s.DecrementActiveJobs("agent-1") // should stay at 0

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, 0, u.ActiveJobs)
}

// --- ReturnTokens ---

func TestRedisStore_ReturnTokens_Good(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.IncrementUsage("agent-1", 10000, 0)
	err := s.ReturnTokens("agent-1", 5000)
	require.NoError(t, err)

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(5000), u.TokensUsed)
}

func TestRedisStore_ReturnTokens_Good_FloorAtZero(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.IncrementUsage("agent-1", 1000, 0)
	_ = s.ReturnTokens("agent-1", 5000) // more than used

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(0), u.TokensUsed)
}

func TestRedisStore_ReturnTokens_Good_NoRecord(t *testing.T) {
	s := newTestRedisStore(t)

	// Return tokens for agent with no usage record -- should be a no-op
	err := s.ReturnTokens("agent-1", 500)
	require.NoError(t, err)

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(0), u.TokensUsed)
}

// --- ResetUsage ---

func TestRedisStore_ResetUsage_Good(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.IncrementUsage("agent-1", 50000, 5)

	err := s.ResetUsage("agent-1")
	require.NoError(t, err)

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(0), u.TokensUsed)
	assert.Equal(t, 0, u.JobsStarted)
	assert.Equal(t, 0, u.ActiveJobs)
}

// --- ModelQuota ---

func TestRedisStore_GetModelQuota_Bad_NotFound(t *testing.T) {
	s := newTestRedisStore(t)
	_, err := s.GetModelQuota("nonexistent")
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok, "expected *APIError")
	assert.Equal(t, 404, apiErr.Code)
	assert.Contains(t, err.Error(), "model quota not found")
}

func TestRedisStore_SetGetModelQuota_Good(t *testing.T) {
	s := newTestRedisStore(t)

	q := &ModelQuota{
		Model:            "claude-opus-4-6",
		DailyTokenBudget: 500000,
		HourlyRateLimit:  100,
		CostCeiling:      10000,
	}
	err := s.SetModelQuota(q)
	require.NoError(t, err)

	got, err := s.GetModelQuota("claude-opus-4-6")
	require.NoError(t, err)
	assert.Equal(t, q.Model, got.Model)
	assert.Equal(t, q.DailyTokenBudget, got.DailyTokenBudget)
	assert.Equal(t, q.HourlyRateLimit, got.HourlyRateLimit)
	assert.Equal(t, q.CostCeiling, got.CostCeiling)
}

// --- ModelUsage ---

func TestRedisStore_ModelUsage_Good(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.IncrementModelUsage("claude-sonnet", 10000)
	_ = s.IncrementModelUsage("claude-sonnet", 5000)

	usage, err := s.GetModelUsage("claude-sonnet")
	require.NoError(t, err)
	assert.Equal(t, int64(15000), usage)
}

func TestRedisStore_GetModelUsage_Good_Default(t *testing.T) {
	s := newTestRedisStore(t)

	usage, err := s.GetModelUsage("unknown-model")
	require.NoError(t, err)
	assert.Equal(t, int64(0), usage)
}

// --- Persistence: set, get, verify ---

func TestRedisStore_Persistence_Good(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100000,
		MaxJobDuration:  15 * time.Minute,
	})
	_ = s.IncrementUsage("agent-1", 25000, 3)
	_ = s.SetModelQuota(&ModelQuota{Model: "claude-opus-4-6", DailyTokenBudget: 500000})
	_ = s.IncrementModelUsage("claude-opus-4-6", 42000)

	// Verify all data persists (same connection, but data is in Redis)
	a, err := s.GetAllowance("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(100000), a.DailyTokenLimit)
	assert.Equal(t, 15*time.Minute, a.MaxJobDuration)

	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(25000), u.TokensUsed)
	assert.Equal(t, 3, u.JobsStarted)
	assert.Equal(t, 3, u.ActiveJobs)

	q, err := s.GetModelQuota("claude-opus-4-6")
	require.NoError(t, err)
	assert.Equal(t, int64(500000), q.DailyTokenBudget)

	mu, err := s.GetModelUsage("claude-opus-4-6")
	require.NoError(t, err)
	assert.Equal(t, int64(42000), mu)
}

// --- Concurrent access ---

func TestRedisStore_ConcurrentIncrementUsage_Good(t *testing.T) {
	s := newTestRedisStore(t)

	const goroutines = 10
	const tokensEach = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			err := s.IncrementUsage("agent-1", tokensEach, 1)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(goroutines*tokensEach), u.TokensUsed)
	assert.Equal(t, goroutines, u.JobsStarted)
	assert.Equal(t, goroutines, u.ActiveJobs)
}

func TestRedisStore_ConcurrentModelUsage_Good(t *testing.T) {
	s := newTestRedisStore(t)

	const goroutines = 10
	const tokensEach int64 = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			err := s.IncrementModelUsage("claude-opus-4-6", tokensEach)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	usage, err := s.GetModelUsage("claude-opus-4-6")
	require.NoError(t, err)
	assert.Equal(t, goroutines*tokensEach, usage)
}

func TestRedisStore_ConcurrentMixed_Good(t *testing.T) {
	s := newTestRedisStore(t)

	_ = s.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 1000000,
		DailyJobLimit:   100,
		ConcurrentJobs:  50,
	})

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Increment usage
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = s.IncrementUsage("agent-1", 100, 1)
		}()
	}

	// Decrement active jobs
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = s.DecrementActiveJobs("agent-1")
		}()
	}

	// Return tokens
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = s.ReturnTokens("agent-1", 10)
		}()
	}

	wg.Wait()

	// Verify no panics and data is consistent
	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, u.TokensUsed, int64(0))
	assert.GreaterOrEqual(t, u.ActiveJobs, 0)
}

// --- AllowanceService integration via RedisStore ---

func TestRedisStore_AllowanceServiceCheck_Good(t *testing.T) {
	s := newTestRedisStore(t)
	svc := NewAllowanceService(s)

	_ = s.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100000,
		DailyJobLimit:   10,
		ConcurrentJobs:  2,
	})

	result, err := svc.Check("agent-1", "")
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, AllowanceOK, result.Status)
}

func TestRedisStore_AllowanceServiceRecordUsage_Good(t *testing.T) {
	s := newTestRedisStore(t)
	svc := NewAllowanceService(s)

	_ = s.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100000,
	})

	// Start job
	err := svc.RecordUsage(UsageReport{
		AgentID: "agent-1",
		JobID:   "job-1",
		Event:   QuotaEventJobStarted,
	})
	require.NoError(t, err)

	// Complete job
	err = svc.RecordUsage(UsageReport{
		AgentID:   "agent-1",
		JobID:     "job-1",
		Model:     "claude-sonnet",
		TokensIn:  1000,
		TokensOut: 500,
		Event:     QuotaEventJobCompleted,
	})
	require.NoError(t, err)

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(1500), u.TokensUsed)
	assert.Equal(t, 0, u.ActiveJobs)
}

// --- Config-based factory with redis backend ---

func TestNewAllowanceStoreFromConfig_Good_Redis(t *testing.T) {
	cfg := AllowanceConfig{
		StoreBackend: "redis",
		RedisAddr:    testRedisAddr,
	}
	s, err := NewAllowanceStoreFromConfig(cfg)
	if err != nil {
		t.Skipf("Redis unavailable at %s: %v", testRedisAddr, err)
	}
	rs, ok := s.(*RedisStore)
	assert.True(t, ok, "expected RedisStore")
	_ = rs.Close()
}

// --- Constructor error case ---

func TestNewRedisStore_Bad_Unreachable(t *testing.T) {
	_, err := NewRedisStore("127.0.0.1:1") // almost certainly unreachable
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok, "expected *APIError")
	assert.Equal(t, 500, apiErr.Code)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}
