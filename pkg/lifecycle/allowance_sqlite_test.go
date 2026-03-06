package lifecycle

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSQLiteStore creates a SQLiteStore in a temp directory.
func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// --- SetAllowance / GetAllowance ---

func TestSQLiteStore_SetGetAllowance_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

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

func TestSQLiteStore_GetAllowance_Bad_NotFound(t *testing.T) {
	s := newTestSQLiteStore(t)
	_, err := s.GetAllowance("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowance not found")
}

func TestSQLiteStore_SetAllowance_Good_Overwrite(t *testing.T) {
	s := newTestSQLiteStore(t)

	_ = s.SetAllowance(&AgentAllowance{AgentID: "agent-1", DailyTokenLimit: 100})
	_ = s.SetAllowance(&AgentAllowance{AgentID: "agent-1", DailyTokenLimit: 200})

	got, err := s.GetAllowance("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(200), got.DailyTokenLimit)
}

// --- GetUsage / IncrementUsage ---

func TestSQLiteStore_GetUsage_Good_Default(t *testing.T) {
	s := newTestSQLiteStore(t)

	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "agent-1", u.AgentID)
	assert.Equal(t, int64(0), u.TokensUsed)
	assert.Equal(t, 0, u.JobsStarted)
	assert.Equal(t, 0, u.ActiveJobs)
}

func TestSQLiteStore_IncrementUsage_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

	err := s.IncrementUsage("agent-1", 5000, 1)
	require.NoError(t, err)

	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(5000), u.TokensUsed)
	assert.Equal(t, 1, u.JobsStarted)
	assert.Equal(t, 1, u.ActiveJobs)
}

func TestSQLiteStore_IncrementUsage_Good_Accumulates(t *testing.T) {
	s := newTestSQLiteStore(t)

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

func TestSQLiteStore_DecrementActiveJobs_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

	_ = s.IncrementUsage("agent-1", 0, 2)
	_ = s.DecrementActiveJobs("agent-1")

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, 1, u.ActiveJobs)
}

func TestSQLiteStore_DecrementActiveJobs_Good_FloorAtZero(t *testing.T) {
	s := newTestSQLiteStore(t)

	_ = s.DecrementActiveJobs("agent-1") // no usage record yet
	_ = s.IncrementUsage("agent-1", 0, 0)
	_ = s.DecrementActiveJobs("agent-1") // should stay at 0

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, 0, u.ActiveJobs)
}

// --- ReturnTokens ---

func TestSQLiteStore_ReturnTokens_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

	_ = s.IncrementUsage("agent-1", 10000, 0)
	err := s.ReturnTokens("agent-1", 5000)
	require.NoError(t, err)

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(5000), u.TokensUsed)
}

func TestSQLiteStore_ReturnTokens_Good_FloorAtZero(t *testing.T) {
	s := newTestSQLiteStore(t)

	_ = s.IncrementUsage("agent-1", 1000, 0)
	_ = s.ReturnTokens("agent-1", 5000) // more than used

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(0), u.TokensUsed)
}

func TestSQLiteStore_ReturnTokens_Good_NoRecord(t *testing.T) {
	s := newTestSQLiteStore(t)

	// Return tokens for agent with no usage record -- should create one at 0
	err := s.ReturnTokens("agent-1", 500)
	require.NoError(t, err)

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(0), u.TokensUsed)
}

// --- ResetUsage ---

func TestSQLiteStore_ResetUsage_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

	_ = s.IncrementUsage("agent-1", 50000, 5)

	err := s.ResetUsage("agent-1")
	require.NoError(t, err)

	u, _ := s.GetUsage("agent-1")
	assert.Equal(t, int64(0), u.TokensUsed)
	assert.Equal(t, 0, u.JobsStarted)
	assert.Equal(t, 0, u.ActiveJobs)
}

// --- ModelQuota ---

func TestSQLiteStore_GetModelQuota_Bad_NotFound(t *testing.T) {
	s := newTestSQLiteStore(t)
	_, err := s.GetModelQuota("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model quota not found")
}

func TestSQLiteStore_SetGetModelQuota_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

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

func TestSQLiteStore_ModelUsage_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

	_ = s.IncrementModelUsage("claude-sonnet", 10000)
	_ = s.IncrementModelUsage("claude-sonnet", 5000)

	usage, err := s.GetModelUsage("claude-sonnet")
	require.NoError(t, err)
	assert.Equal(t, int64(15000), usage)
}

func TestSQLiteStore_GetModelUsage_Good_Default(t *testing.T) {
	s := newTestSQLiteStore(t)

	usage, err := s.GetModelUsage("unknown-model")
	require.NoError(t, err)
	assert.Equal(t, int64(0), usage)
}

// --- Persistence: close and reopen ---

func TestSQLiteStore_Persistence_Good(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist.db")

	// Phase 1: write data
	s1, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)

	_ = s1.SetAllowance(&AgentAllowance{
		AgentID:         "agent-1",
		DailyTokenLimit: 100000,
		MaxJobDuration:  15 * time.Minute,
	})
	_ = s1.IncrementUsage("agent-1", 25000, 3)
	_ = s1.SetModelQuota(&ModelQuota{Model: "claude-opus-4-6", DailyTokenBudget: 500000})
	_ = s1.IncrementModelUsage("claude-opus-4-6", 42000)
	require.NoError(t, s1.Close())

	// Phase 2: reopen and verify
	s2, err := NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer func() { _ = s2.Close() }()

	a, err := s2.GetAllowance("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(100000), a.DailyTokenLimit)
	assert.Equal(t, 15*time.Minute, a.MaxJobDuration)

	u, err := s2.GetUsage("agent-1")
	require.NoError(t, err)
	assert.Equal(t, int64(25000), u.TokensUsed)
	assert.Equal(t, 3, u.JobsStarted)
	assert.Equal(t, 3, u.ActiveJobs)

	q, err := s2.GetModelQuota("claude-opus-4-6")
	require.NoError(t, err)
	assert.Equal(t, int64(500000), q.DailyTokenBudget)

	mu, err := s2.GetModelUsage("claude-opus-4-6")
	require.NoError(t, err)
	assert.Equal(t, int64(42000), mu)
}

// --- Concurrent access ---

func TestSQLiteStore_ConcurrentIncrementUsage_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

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

func TestSQLiteStore_ConcurrentModelUsage_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

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

func TestSQLiteStore_ConcurrentMixed_Good(t *testing.T) {
	s := newTestSQLiteStore(t)

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

	// Just verify no panics and data is consistent
	u, err := s.GetUsage("agent-1")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, u.TokensUsed, int64(0))
	assert.GreaterOrEqual(t, u.ActiveJobs, 0)
}

// --- AllowanceService integration via SQLiteStore ---

func TestSQLiteStore_AllowanceServiceCheck_Good(t *testing.T) {
	s := newTestSQLiteStore(t)
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

func TestSQLiteStore_AllowanceServiceRecordUsage_Good(t *testing.T) {
	s := newTestSQLiteStore(t)
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

// --- Config-based factory ---

func TestNewAllowanceStoreFromConfig_Good_Memory(t *testing.T) {
	cfg := AllowanceConfig{StoreBackend: "memory"}
	s, err := NewAllowanceStoreFromConfig(cfg)
	require.NoError(t, err)
	_, ok := s.(*MemoryStore)
	assert.True(t, ok, "expected MemoryStore")
}

func TestNewAllowanceStoreFromConfig_Good_Default(t *testing.T) {
	cfg := AllowanceConfig{} // empty defaults to memory
	s, err := NewAllowanceStoreFromConfig(cfg)
	require.NoError(t, err)
	_, ok := s.(*MemoryStore)
	assert.True(t, ok, "expected MemoryStore for empty config")
}

func TestNewAllowanceStoreFromConfig_Good_SQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "factory.db")
	cfg := AllowanceConfig{
		StoreBackend: "sqlite",
		StorePath:    dbPath,
	}
	s, err := NewAllowanceStoreFromConfig(cfg)
	require.NoError(t, err)
	ss, ok := s.(*SQLiteStore)
	assert.True(t, ok, "expected SQLiteStore")
	_ = ss.Close()
}

func TestNewAllowanceStoreFromConfig_Bad_UnknownBackend(t *testing.T) {
	cfg := AllowanceConfig{StoreBackend: "cassandra"}
	_, err := NewAllowanceStoreFromConfig(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported store backend")
}

// --- NewSQLiteStore error case ---

func TestNewSQLiteStore_Bad_InvalidPath(t *testing.T) {
	_, err := NewSQLiteStore("/nonexistent/deeply/nested/dir/test.db")
	require.Error(t, err)
}
