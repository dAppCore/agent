package lifecycle

import (
	"encoding/json"
	"errors"
	"iter"
	"sync"
	"time"

	"forge.lthn.ai/core/go-store"
)

// SQLite group names for namespacing data in the KV store.
const (
	groupAllowances = "allowances"
	groupUsage      = "usage"
	groupModelQuota = "model_quotas"
	groupModelUsage = "model_usage"
)

// SQLiteStore implements AllowanceStore using go-store (SQLite KV).
// It provides persistent storage that survives process restarts.
type SQLiteStore struct {
	db *store.Store
	mu sync.Mutex // serialises read-modify-write operations
}

// Allowances returns an iterator over all agent allowances.
func (s *SQLiteStore) Allowances() iter.Seq[*AgentAllowance] {
	return func(yield func(*AgentAllowance) bool) {
		for kv, err := range s.db.All(groupAllowances) {
			if err != nil {
				continue
			}
			var a allowanceJSON
			if err := json.Unmarshal([]byte(kv.Value), &a); err != nil {
				continue
			}
			if !yield(a.toAgentAllowance()) {
				return
			}
		}
	}
}

// Usages returns an iterator over all usage records.
func (s *SQLiteStore) Usages() iter.Seq[*UsageRecord] {
	return func(yield func(*UsageRecord) bool) {
		for kv, err := range s.db.All(groupUsage) {
			if err != nil {
				continue
			}
			var u UsageRecord
			if err := json.Unmarshal([]byte(kv.Value), &u); err != nil {
				continue
			}
			if !yield(&u) {
				return
			}
		}
	}
}

// NewSQLiteStore creates a new SQLite-backed allowance store at the given path.
// Use ":memory:" for tests that do not need persistence.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := store.New(dbPath)
	if err != nil {
		return nil, &APIError{Code: 500, Message: "failed to open SQLite store: " + err.Error()}
	}
	return &SQLiteStore{db: db}, nil
}

// Close releases the underlying SQLite database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// GetAllowance returns the quota limits for an agent.
func (s *SQLiteStore) GetAllowance(agentID string) (*AgentAllowance, error) {
	val, err := s.db.Get(groupAllowances, agentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, &APIError{Code: 404, Message: "allowance not found for agent: " + agentID}
		}
		return nil, &APIError{Code: 500, Message: "failed to get allowance: " + err.Error()}
	}
	var a allowanceJSON
	if err := json.Unmarshal([]byte(val), &a); err != nil {
		return nil, &APIError{Code: 500, Message: "failed to unmarshal allowance: " + err.Error()}
	}
	return a.toAgentAllowance(), nil
}

// SetAllowance persists quota limits for an agent.
func (s *SQLiteStore) SetAllowance(a *AgentAllowance) error {
	aj := newAllowanceJSON(a)
	data, err := json.Marshal(aj)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal allowance: " + err.Error()}
	}
	if err := s.db.Set(groupAllowances, a.AgentID, string(data)); err != nil {
		return &APIError{Code: 500, Message: "failed to set allowance: " + err.Error()}
	}
	return nil
}

// GetUsage returns the current usage record for an agent.
func (s *SQLiteStore) GetUsage(agentID string) (*UsageRecord, error) {
	val, err := s.db.Get(groupUsage, agentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return &UsageRecord{
				AgentID:     agentID,
				PeriodStart: startOfDay(time.Now().UTC()),
			}, nil
		}
		return nil, &APIError{Code: 500, Message: "failed to get usage: " + err.Error()}
	}
	var u UsageRecord
	if err := json.Unmarshal([]byte(val), &u); err != nil {
		return nil, &APIError{Code: 500, Message: "failed to unmarshal usage: " + err.Error()}
	}
	return &u, nil
}

// IncrementUsage atomically adds to an agent's usage counters.
func (s *SQLiteStore) IncrementUsage(agentID string, tokens int64, jobs int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, err := s.getUsageLocked(agentID)
	if err != nil {
		return err
	}
	u.TokensUsed += tokens
	u.JobsStarted += jobs
	if jobs > 0 {
		u.ActiveJobs += jobs
	}
	return s.putUsageLocked(u)
}

// DecrementActiveJobs reduces the active job count by 1.
func (s *SQLiteStore) DecrementActiveJobs(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, err := s.getUsageLocked(agentID)
	if err != nil {
		return err
	}
	if u.ActiveJobs > 0 {
		u.ActiveJobs--
	}
	return s.putUsageLocked(u)
}

// ReturnTokens adds tokens back to the agent's remaining quota.
func (s *SQLiteStore) ReturnTokens(agentID string, tokens int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, err := s.getUsageLocked(agentID)
	if err != nil {
		return err
	}
	u.TokensUsed -= tokens
	if u.TokensUsed < 0 {
		u.TokensUsed = 0
	}
	return s.putUsageLocked(u)
}

// ResetUsage clears usage counters for an agent.
func (s *SQLiteStore) ResetUsage(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := &UsageRecord{
		AgentID:     agentID,
		PeriodStart: startOfDay(time.Now().UTC()),
	}
	return s.putUsageLocked(u)
}

// GetModelQuota returns global limits for a model.
func (s *SQLiteStore) GetModelQuota(model string) (*ModelQuota, error) {
	val, err := s.db.Get(groupModelQuota, model)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, &APIError{Code: 404, Message: "model quota not found: " + model}
		}
		return nil, &APIError{Code: 500, Message: "failed to get model quota: " + err.Error()}
	}
	var q ModelQuota
	if err := json.Unmarshal([]byte(val), &q); err != nil {
		return nil, &APIError{Code: 500, Message: "failed to unmarshal model quota: " + err.Error()}
	}
	return &q, nil
}

// GetModelUsage returns current token usage for a model.
func (s *SQLiteStore) GetModelUsage(model string) (int64, error) {
	val, err := s.db.Get(groupModelUsage, model)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 0, nil
		}
		return 0, &APIError{Code: 500, Message: "failed to get model usage: " + err.Error()}
	}
	var tokens int64
	if err := json.Unmarshal([]byte(val), &tokens); err != nil {
		return 0, &APIError{Code: 500, Message: "failed to unmarshal model usage: " + err.Error()}
	}
	return tokens, nil
}

// IncrementModelUsage atomically adds to a model's usage counter.
func (s *SQLiteStore) IncrementModelUsage(model string, tokens int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.getModelUsageLocked(model)
	if err != nil {
		return err
	}
	current += tokens
	data, err := json.Marshal(current)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal model usage: " + err.Error()}
	}
	if err := s.db.Set(groupModelUsage, model, string(data)); err != nil {
		return &APIError{Code: 500, Message: "failed to set model usage: " + err.Error()}
	}
	return nil
}

// SetModelQuota persists global limits for a model.
func (s *SQLiteStore) SetModelQuota(q *ModelQuota) error {
	data, err := json.Marshal(q)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal model quota: " + err.Error()}
	}
	if err := s.db.Set(groupModelQuota, q.Model, string(data)); err != nil {
		return &APIError{Code: 500, Message: "failed to set model quota: " + err.Error()}
	}
	return nil
}

// --- internal helpers (must be called with mu held) ---

// getUsageLocked reads a UsageRecord from the store. Caller must hold s.mu.
func (s *SQLiteStore) getUsageLocked(agentID string) (*UsageRecord, error) {
	val, err := s.db.Get(groupUsage, agentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return &UsageRecord{
				AgentID:     agentID,
				PeriodStart: startOfDay(time.Now().UTC()),
			}, nil
		}
		return nil, &APIError{Code: 500, Message: "failed to get usage: " + err.Error()}
	}
	var u UsageRecord
	if err := json.Unmarshal([]byte(val), &u); err != nil {
		return nil, &APIError{Code: 500, Message: "failed to unmarshal usage: " + err.Error()}
	}
	return &u, nil
}

// putUsageLocked writes a UsageRecord to the store. Caller must hold s.mu.
func (s *SQLiteStore) putUsageLocked(u *UsageRecord) error {
	data, err := json.Marshal(u)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal usage: " + err.Error()}
	}
	if err := s.db.Set(groupUsage, u.AgentID, string(data)); err != nil {
		return &APIError{Code: 500, Message: "failed to set usage: " + err.Error()}
	}
	return nil
}

// getModelUsageLocked reads model usage from the store. Caller must hold s.mu.
func (s *SQLiteStore) getModelUsageLocked(model string) (int64, error) {
	val, err := s.db.Get(groupModelUsage, model)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return 0, nil
		}
		return 0, &APIError{Code: 500, Message: "failed to get model usage: " + err.Error()}
	}
	var tokens int64
	if err := json.Unmarshal([]byte(val), &tokens); err != nil {
		return 0, &APIError{Code: 500, Message: "failed to unmarshal model usage: " + err.Error()}
	}
	return tokens, nil
}

// --- JSON serialisation helper for AgentAllowance ---
// time.Duration does not have a stable JSON representation. We serialise it
// as an int64 (nanoseconds) to avoid locale-dependent string parsing.

type allowanceJSON struct {
	AgentID          string   `json:"agent_id"`
	DailyTokenLimit  int64    `json:"daily_token_limit"`
	DailyJobLimit    int      `json:"daily_job_limit"`
	ConcurrentJobs   int      `json:"concurrent_jobs"`
	MaxJobDurationNs int64    `json:"max_job_duration_ns"`
	ModelAllowlist   []string `json:"model_allowlist,omitempty"`
}

func newAllowanceJSON(a *AgentAllowance) *allowanceJSON {
	return &allowanceJSON{
		AgentID:          a.AgentID,
		DailyTokenLimit:  a.DailyTokenLimit,
		DailyJobLimit:    a.DailyJobLimit,
		ConcurrentJobs:   a.ConcurrentJobs,
		MaxJobDurationNs: int64(a.MaxJobDuration),
		ModelAllowlist:   a.ModelAllowlist,
	}
}

func (aj *allowanceJSON) toAgentAllowance() *AgentAllowance {
	return &AgentAllowance{
		AgentID:         aj.AgentID,
		DailyTokenLimit: aj.DailyTokenLimit,
		DailyJobLimit:   aj.DailyJobLimit,
		ConcurrentJobs:  aj.ConcurrentJobs,
		MaxJobDuration:  time.Duration(aj.MaxJobDurationNs),
		ModelAllowlist:  aj.ModelAllowlist,
	}
}
