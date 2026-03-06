package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore implements AllowanceStore using Redis as the backing store.
// It provides persistent, network-accessible storage suitable for multi-node
// deployments where agents share quota state.
type RedisStore struct {
	client *redis.Client
	prefix string
}

// Allowances returns an iterator over all agent allowances.
func (r *RedisStore) Allowances() iter.Seq[*AgentAllowance] {
	return func(yield func(*AgentAllowance) bool) {
		ctx := context.Background()
		pattern := r.prefix + ":allowance:*"
		iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			val, err := r.client.Get(ctx, iter.Val()).Result()
			if err != nil {
				continue
			}
			var aj allowanceJSON
			if err := json.Unmarshal([]byte(val), &aj); err != nil {
				continue
			}
			if !yield(aj.toAgentAllowance()) {
				return
			}
		}
	}
}

// Usages returns an iterator over all usage records.
func (r *RedisStore) Usages() iter.Seq[*UsageRecord] {
	return func(yield func(*UsageRecord) bool) {
		ctx := context.Background()
		pattern := r.prefix + ":usage:*"
		iter := r.client.Scan(ctx, 0, pattern, 100).Iterator()
		for iter.Next(ctx) {
			val, err := r.client.Get(ctx, iter.Val()).Result()
			if err != nil {
				continue
			}
			var u UsageRecord
			if err := json.Unmarshal([]byte(val), &u); err != nil {
				continue
			}
			if !yield(&u) {
				return
			}
		}
	}
}

// redisConfig holds the configuration for a RedisStore.
type redisConfig struct {
	password string
	db       int
	prefix   string
}

// RedisOption is a functional option for configuring a RedisStore.
type RedisOption func(*redisConfig)

// WithRedisPassword sets the password for authenticating with Redis.
func WithRedisPassword(pw string) RedisOption {
	return func(c *redisConfig) {
		c.password = pw
	}
}

// WithRedisDB selects the Redis database number.
func WithRedisDB(db int) RedisOption {
	return func(c *redisConfig) {
		c.db = db
	}
}

// WithRedisPrefix sets the key prefix for all Redis keys. Default: "agentic".
func WithRedisPrefix(prefix string) RedisOption {
	return func(c *redisConfig) {
		c.prefix = prefix
	}
}

// NewRedisStore creates a new Redis-backed allowance store connecting to the
// given address (host:port). It pings the server to verify connectivity.
func NewRedisStore(addr string, opts ...RedisOption) (*RedisStore, error) {
	cfg := &redisConfig{
		prefix: "agentic",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.password,
		DB:       cfg.db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, &APIError{Code: 500, Message: "failed to connect to Redis: " + err.Error()}
	}

	return &RedisStore{
		client: client,
		prefix: cfg.prefix,
	}, nil
}

// Close releases the underlying Redis connection.
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// --- key helpers ---

func (r *RedisStore) allowanceKey(agentID string) string {
	return r.prefix + ":allowance:" + agentID
}

func (r *RedisStore) usageKey(agentID string) string {
	return r.prefix + ":usage:" + agentID
}

func (r *RedisStore) modelQuotaKey(model string) string {
	return r.prefix + ":model_quota:" + model
}

func (r *RedisStore) modelUsageKey(model string) string {
	return r.prefix + ":model_usage:" + model
}

// --- AllowanceStore interface ---

// GetAllowance returns the quota limits for an agent.
func (r *RedisStore) GetAllowance(agentID string) (*AgentAllowance, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.allowanceKey(agentID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, &APIError{Code: 404, Message: "allowance not found for agent: " + agentID}
		}
		return nil, &APIError{Code: 500, Message: "failed to get allowance: " + err.Error()}
	}
	var aj allowanceJSON
	if err := json.Unmarshal([]byte(val), &aj); err != nil {
		return nil, &APIError{Code: 500, Message: "failed to unmarshal allowance: " + err.Error()}
	}
	return aj.toAgentAllowance(), nil
}

// SetAllowance persists quota limits for an agent.
func (r *RedisStore) SetAllowance(a *AgentAllowance) error {
	ctx := context.Background()
	aj := newAllowanceJSON(a)
	data, err := json.Marshal(aj)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal allowance: " + err.Error()}
	}
	if err := r.client.Set(ctx, r.allowanceKey(a.AgentID), data, 0).Err(); err != nil {
		return &APIError{Code: 500, Message: "failed to set allowance: " + err.Error()}
	}
	return nil
}

// GetUsage returns the current usage record for an agent.
func (r *RedisStore) GetUsage(agentID string) (*UsageRecord, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.usageKey(agentID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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

// incrementUsageLua atomically reads, increments, and writes back a usage record.
// KEYS[1] = usage key
// ARGV[1] = tokens to add (int64)
// ARGV[2] = jobs to add (int)
// ARGV[3] = agent ID (for creating a new record)
// ARGV[4] = period start ISO string (for creating a new record)
var incrementUsageLua = redis.NewScript(`
local val = redis.call('GET', KEYS[1])
local u
if val then
    u = cjson.decode(val)
else
    u = {
        agent_id = ARGV[3],
        tokens_used = 0,
        jobs_started = 0,
        active_jobs = 0,
        period_start = ARGV[4]
    }
end
u.tokens_used = u.tokens_used + tonumber(ARGV[1])
u.jobs_started = u.jobs_started + tonumber(ARGV[2])
if tonumber(ARGV[2]) > 0 then
    u.active_jobs = u.active_jobs + tonumber(ARGV[2])
end
redis.call('SET', KEYS[1], cjson.encode(u))
return 'OK'
`)

// IncrementUsage atomically adds to an agent's usage counters.
func (r *RedisStore) IncrementUsage(agentID string, tokens int64, jobs int) error {
	ctx := context.Background()
	periodStart := startOfDay(time.Now().UTC()).Format(time.RFC3339)
	err := incrementUsageLua.Run(ctx, r.client,
		[]string{r.usageKey(agentID)},
		tokens, jobs, agentID, periodStart,
	).Err()
	if err != nil {
		return &APIError{Code: 500, Message: "failed to increment usage: " + err.Error()}
	}
	return nil
}

// decrementActiveJobsLua atomically decrements the active jobs counter, flooring at zero.
// KEYS[1] = usage key
// ARGV[1] = agent ID
// ARGV[2] = period start ISO string
var decrementActiveJobsLua = redis.NewScript(`
local val = redis.call('GET', KEYS[1])
if not val then
    return 'OK'
end
local u = cjson.decode(val)
if u.active_jobs and u.active_jobs > 0 then
    u.active_jobs = u.active_jobs - 1
end
redis.call('SET', KEYS[1], cjson.encode(u))
return 'OK'
`)

// DecrementActiveJobs reduces the active job count by 1.
func (r *RedisStore) DecrementActiveJobs(agentID string) error {
	ctx := context.Background()
	periodStart := startOfDay(time.Now().UTC()).Format(time.RFC3339)
	err := decrementActiveJobsLua.Run(ctx, r.client,
		[]string{r.usageKey(agentID)},
		agentID, periodStart,
	).Err()
	if err != nil {
		return &APIError{Code: 500, Message: "failed to decrement active jobs: " + err.Error()}
	}
	return nil
}

// returnTokensLua atomically subtracts tokens from usage, flooring at zero.
// KEYS[1] = usage key
// ARGV[1] = tokens to return (int64)
// ARGV[2] = agent ID
// ARGV[3] = period start ISO string
var returnTokensLua = redis.NewScript(`
local val = redis.call('GET', KEYS[1])
if not val then
    return 'OK'
end
local u = cjson.decode(val)
u.tokens_used = u.tokens_used - tonumber(ARGV[1])
if u.tokens_used < 0 then
    u.tokens_used = 0
end
redis.call('SET', KEYS[1], cjson.encode(u))
return 'OK'
`)

// ReturnTokens adds tokens back to the agent's remaining quota.
func (r *RedisStore) ReturnTokens(agentID string, tokens int64) error {
	ctx := context.Background()
	periodStart := startOfDay(time.Now().UTC()).Format(time.RFC3339)
	err := returnTokensLua.Run(ctx, r.client,
		[]string{r.usageKey(agentID)},
		tokens, agentID, periodStart,
	).Err()
	if err != nil {
		return &APIError{Code: 500, Message: "failed to return tokens: " + err.Error()}
	}
	return nil
}

// ResetUsage clears usage counters for an agent (daily reset).
func (r *RedisStore) ResetUsage(agentID string) error {
	ctx := context.Background()
	u := &UsageRecord{
		AgentID:     agentID,
		PeriodStart: startOfDay(time.Now().UTC()),
	}
	data, err := json.Marshal(u)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal usage: " + err.Error()}
	}
	if err := r.client.Set(ctx, r.usageKey(agentID), data, 0).Err(); err != nil {
		return &APIError{Code: 500, Message: "failed to reset usage: " + err.Error()}
	}
	return nil
}

// GetModelQuota returns global limits for a model.
func (r *RedisStore) GetModelQuota(model string) (*ModelQuota, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.modelQuotaKey(model)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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
func (r *RedisStore) GetModelUsage(model string) (int64, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.modelUsageKey(model)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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

// incrementModelUsageLua atomically increments the model usage counter.
// KEYS[1] = model usage key
// ARGV[1] = tokens to add
var incrementModelUsageLua = redis.NewScript(`
local val = redis.call('GET', KEYS[1])
local current = 0
if val then
    current = tonumber(val)
end
current = current + tonumber(ARGV[1])
redis.call('SET', KEYS[1], tostring(current))
return current
`)

// IncrementModelUsage atomically adds to a model's usage counter.
func (r *RedisStore) IncrementModelUsage(model string, tokens int64) error {
	ctx := context.Background()
	err := incrementModelUsageLua.Run(ctx, r.client,
		[]string{r.modelUsageKey(model)},
		tokens,
	).Err()
	if err != nil {
		return &APIError{Code: 500, Message: "failed to increment model usage: " + err.Error()}
	}
	return nil
}

// SetModelQuota persists global limits for a model.
func (r *RedisStore) SetModelQuota(q *ModelQuota) error {
	ctx := context.Background()
	data, err := json.Marshal(q)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal model quota: " + err.Error()}
	}
	if err := r.client.Set(ctx, r.modelQuotaKey(q.Model), data, 0).Err(); err != nil {
		return &APIError{Code: 500, Message: "failed to set model quota: " + err.Error()}
	}
	return nil
}

// FlushPrefix deletes all keys matching the store's prefix. Useful for testing cleanup.
func (r *RedisStore) FlushPrefix(ctx context.Context) error {
	iter := r.client.Scan(ctx, 0, r.prefix+":*", 100).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}
