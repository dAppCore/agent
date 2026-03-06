package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"slices"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRegistry implements AgentRegistry using Redis as the backing store.
// It provides persistent, network-accessible agent registration suitable for
// multi-node deployments. Heartbeat refreshes key TTL for natural reaping via
// expiry.
type RedisRegistry struct {
	client     *redis.Client
	prefix     string
	defaultTTL time.Duration
}

// redisRegistryConfig holds the configuration for a RedisRegistry.
type redisRegistryConfig struct {
	password string
	db       int
	prefix   string
	ttl      time.Duration
}

// RedisRegistryOption is a functional option for configuring a RedisRegistry.
type RedisRegistryOption func(*redisRegistryConfig)

// WithRegistryRedisPassword sets the password for authenticating with Redis.
func WithRegistryRedisPassword(pw string) RedisRegistryOption {
	return func(c *redisRegistryConfig) {
		c.password = pw
	}
}

// WithRegistryRedisDB selects the Redis database number.
func WithRegistryRedisDB(db int) RedisRegistryOption {
	return func(c *redisRegistryConfig) {
		c.db = db
	}
}

// WithRegistryRedisPrefix sets the key prefix for all Redis keys.
// Default: "agentic".
func WithRegistryRedisPrefix(prefix string) RedisRegistryOption {
	return func(c *redisRegistryConfig) {
		c.prefix = prefix
	}
}

// WithRegistryTTL sets the default TTL for agent keys. Default: 5 minutes.
// Heartbeat refreshes this TTL. Agents whose keys expire are naturally reaped.
func WithRegistryTTL(ttl time.Duration) RedisRegistryOption {
	return func(c *redisRegistryConfig) {
		c.ttl = ttl
	}
}

// NewRedisRegistry creates a new Redis-backed agent registry connecting to the
// given address (host:port). It pings the server to verify connectivity.
func NewRedisRegistry(addr string, opts ...RedisRegistryOption) (*RedisRegistry, error) {
	cfg := &redisRegistryConfig{
		prefix: "agentic",
		ttl:    5 * time.Minute,
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

	return &RedisRegistry{
		client:     client,
		prefix:     cfg.prefix,
		defaultTTL: cfg.ttl,
	}, nil
}

// Close releases the underlying Redis connection.
func (r *RedisRegistry) Close() error {
	return r.client.Close()
}

// --- key helpers ---

func (r *RedisRegistry) agentKey(id string) string {
	return r.prefix + ":agent:" + id
}

func (r *RedisRegistry) agentPattern() string {
	return r.prefix + ":agent:*"
}

// --- AgentRegistry interface ---

// Register adds or updates an agent in the registry.
func (r *RedisRegistry) Register(agent AgentInfo) error {
	if agent.ID == "" {
		return &APIError{Code: 400, Message: "agent ID is required"}
	}
	ctx := context.Background()
	data, err := json.Marshal(agent)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal agent: " + err.Error()}
	}
	if err := r.client.Set(ctx, r.agentKey(agent.ID), data, r.defaultTTL).Err(); err != nil {
		return &APIError{Code: 500, Message: "failed to register agent: " + err.Error()}
	}
	return nil
}

// Deregister removes an agent from the registry. Returns an error if the agent
// is not found.
func (r *RedisRegistry) Deregister(id string) error {
	ctx := context.Background()
	n, err := r.client.Del(ctx, r.agentKey(id)).Result()
	if err != nil {
		return &APIError{Code: 500, Message: "failed to deregister agent: " + err.Error()}
	}
	if n == 0 {
		return &APIError{Code: 404, Message: "agent not found: " + id}
	}
	return nil
}

// Get returns a copy of the agent info for the given ID. Returns an error if
// the agent is not found.
func (r *RedisRegistry) Get(id string) (AgentInfo, error) {
	ctx := context.Background()
	val, err := r.client.Get(ctx, r.agentKey(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return AgentInfo{}, &APIError{Code: 404, Message: "agent not found: " + id}
		}
		return AgentInfo{}, &APIError{Code: 500, Message: "failed to get agent: " + err.Error()}
	}
	var a AgentInfo
	if err := json.Unmarshal([]byte(val), &a); err != nil {
		return AgentInfo{}, &APIError{Code: 500, Message: "failed to unmarshal agent: " + err.Error()}
	}
	return a, nil
}

// List returns a copy of all registered agents by scanning all agent keys.
func (r *RedisRegistry) List() []AgentInfo {
	return slices.Collect(r.All())
}

// All returns an iterator over all registered agents.
func (r *RedisRegistry) All() iter.Seq[AgentInfo] {
	return func(yield func(AgentInfo) bool) {
		ctx := context.Background()
		iter := r.client.Scan(ctx, 0, r.agentPattern(), 100).Iterator()
		for iter.Next(ctx) {
			val, err := r.client.Get(ctx, iter.Val()).Result()
			if err != nil {
				continue
			}
			var a AgentInfo
			if err := json.Unmarshal([]byte(val), &a); err != nil {
				continue
			}
			if !yield(a) {
				return
			}
		}
	}
}

// Heartbeat updates the agent's LastHeartbeat timestamp and refreshes the key
// TTL. If the agent was Offline, it transitions to Available.
func (r *RedisRegistry) Heartbeat(id string) error {
	ctx := context.Background()
	key := r.agentKey(id)

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &APIError{Code: 404, Message: "agent not found: " + id}
		}
		return &APIError{Code: 500, Message: "failed to get agent for heartbeat: " + err.Error()}
	}

	var a AgentInfo
	if err := json.Unmarshal([]byte(val), &a); err != nil {
		return &APIError{Code: 500, Message: "failed to unmarshal agent: " + err.Error()}
	}

	a.LastHeartbeat = time.Now().UTC()
	if a.Status == AgentOffline {
		a.Status = AgentAvailable
	}

	data, err := json.Marshal(a)
	if err != nil {
		return &APIError{Code: 500, Message: "failed to marshal agent: " + err.Error()}
	}

	if err := r.client.Set(ctx, key, data, r.defaultTTL).Err(); err != nil {
		return &APIError{Code: 500, Message: "failed to update agent heartbeat: " + err.Error()}
	}
	return nil
}

// Reap scans all agent keys and marks agents as Offline if their last heartbeat
// is older than ttl. This is a backup to natural TTL expiry. Returns the IDs
// of agents that were reaped.
func (r *RedisRegistry) Reap(ttl time.Duration) []string {
	return slices.Collect(r.Reaped(ttl))
}

// Reaped returns an iterator over the IDs of agents that were reaped.
func (r *RedisRegistry) Reaped(ttl time.Duration) iter.Seq[string] {
	return func(yield func(string) bool) {
		ctx := context.Background()
		cutoff := time.Now().UTC().Add(-ttl)

		iter := r.client.Scan(ctx, 0, r.agentPattern(), 100).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			val, err := r.client.Get(ctx, key).Result()
			if err != nil {
				continue
			}
			var a AgentInfo
			if err := json.Unmarshal([]byte(val), &a); err != nil {
				continue
			}

			if a.Status != AgentOffline && a.LastHeartbeat.Before(cutoff) {
				a.Status = AgentOffline
				data, err := json.Marshal(a)
				if err != nil {
					continue
				}
				// Preserve remaining TTL (or use default if none).
				remainingTTL, err := r.client.TTL(ctx, key).Result()
				if err != nil || remainingTTL <= 0 {
					remainingTTL = r.defaultTTL
				}
				if err := r.client.Set(ctx, key, data, remainingTTL).Err(); err != nil {
					continue
				}
				if !yield(a.ID) {
					return
				}
			}
		}
	}
}

// FlushPrefix deletes all keys matching the registry's prefix. Useful for
// testing cleanup.
func (r *RedisRegistry) FlushPrefix(ctx context.Context) error {
	iter := r.client.Scan(ctx, 0, r.prefix+":*", 100).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}
