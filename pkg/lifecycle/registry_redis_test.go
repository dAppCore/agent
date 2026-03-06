package lifecycle

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRedisRegistry creates a RedisRegistry with a unique prefix for test isolation.
// Skips the test if Redis is unreachable.
func newTestRedisRegistry(t *testing.T) *RedisRegistry {
	t.Helper()
	prefix := fmt.Sprintf("test_reg_%d", time.Now().UnixNano())
	reg, err := NewRedisRegistry(testRedisAddr,
		WithRegistryRedisPrefix(prefix),
		WithRegistryTTL(5*time.Minute),
	)
	if err != nil {
		t.Skipf("Redis unavailable at %s: %v", testRedisAddr, err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_ = reg.FlushPrefix(ctx)
		_ = reg.Close()
	})
	return reg
}

// --- Register tests ---

func TestRedisRegistry_Register_Good(t *testing.T) {
	reg := newTestRedisRegistry(t)
	err := reg.Register(AgentInfo{
		ID:           "agent-1",
		Name:         "Test Agent",
		Capabilities: []string{"go", "testing"},
		Status:       AgentAvailable,
		MaxLoad:      5,
	})
	require.NoError(t, err)

	got, err := reg.Get("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "agent-1", got.ID)
	assert.Equal(t, "Test Agent", got.Name)
	assert.Equal(t, []string{"go", "testing"}, got.Capabilities)
	assert.Equal(t, AgentAvailable, got.Status)
	assert.Equal(t, 5, got.MaxLoad)
}

func TestRedisRegistry_Register_Good_Overwrite(t *testing.T) {
	reg := newTestRedisRegistry(t)
	_ = reg.Register(AgentInfo{ID: "agent-1", Name: "Original", MaxLoad: 3})
	err := reg.Register(AgentInfo{ID: "agent-1", Name: "Updated", MaxLoad: 10})
	require.NoError(t, err)

	got, err := reg.Get("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, 10, got.MaxLoad)
}

func TestRedisRegistry_Register_Bad_EmptyID(t *testing.T) {
	reg := newTestRedisRegistry(t)
	err := reg.Register(AgentInfo{ID: "", Name: "No ID"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")
}

// --- Deregister tests ---

func TestRedisRegistry_Deregister_Good(t *testing.T) {
	reg := newTestRedisRegistry(t)
	_ = reg.Register(AgentInfo{ID: "agent-1", Name: "To Remove"})

	err := reg.Deregister("agent-1")
	require.NoError(t, err)

	_, err = reg.Get("agent-1")
	require.Error(t, err)
}

func TestRedisRegistry_Deregister_Bad_NotFound(t *testing.T) {
	reg := newTestRedisRegistry(t)
	err := reg.Deregister("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

// --- Get tests ---

func TestRedisRegistry_Get_Good(t *testing.T) {
	reg := newTestRedisRegistry(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	_ = reg.Register(AgentInfo{
		ID:            "agent-1",
		Name:          "Getter",
		Status:        AgentBusy,
		CurrentLoad:   2,
		MaxLoad:       5,
		LastHeartbeat: now,
	})

	got, err := reg.Get("agent-1")
	require.NoError(t, err)
	assert.Equal(t, AgentBusy, got.Status)
	assert.Equal(t, 2, got.CurrentLoad)
	assert.WithinDuration(t, now, got.LastHeartbeat, time.Millisecond)
}

func TestRedisRegistry_Get_Bad_NotFound(t *testing.T) {
	reg := newTestRedisRegistry(t)
	_, err := reg.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

func TestRedisRegistry_Get_Good_ReturnsCopy(t *testing.T) {
	reg := newTestRedisRegistry(t)
	_ = reg.Register(AgentInfo{ID: "agent-1", Name: "Original", CurrentLoad: 1})

	got, _ := reg.Get("agent-1")
	got.CurrentLoad = 99
	got.Name = "Tampered"

	// Re-read — should be unchanged (deserialized from Redis).
	again, _ := reg.Get("agent-1")
	assert.Equal(t, "Original", again.Name)
	assert.Equal(t, 1, again.CurrentLoad)
}

// --- List tests ---

func TestRedisRegistry_List_Good_Empty(t *testing.T) {
	reg := newTestRedisRegistry(t)
	agents := reg.List()
	assert.Empty(t, agents)
}

func TestRedisRegistry_List_Good_Multiple(t *testing.T) {
	reg := newTestRedisRegistry(t)
	_ = reg.Register(AgentInfo{ID: "a", Name: "Alpha"})
	_ = reg.Register(AgentInfo{ID: "b", Name: "Beta"})
	_ = reg.Register(AgentInfo{ID: "c", Name: "Charlie"})

	agents := reg.List()
	assert.Len(t, agents, 3)

	// Sort by ID for deterministic assertion.
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	assert.Equal(t, "a", agents[0].ID)
	assert.Equal(t, "b", agents[1].ID)
	assert.Equal(t, "c", agents[2].ID)
}

// --- Heartbeat tests ---

func TestRedisRegistry_Heartbeat_Good(t *testing.T) {
	reg := newTestRedisRegistry(t)
	past := time.Now().UTC().Add(-5 * time.Minute)
	_ = reg.Register(AgentInfo{
		ID:            "agent-1",
		Status:        AgentAvailable,
		LastHeartbeat: past,
	})

	err := reg.Heartbeat("agent-1")
	require.NoError(t, err)

	got, _ := reg.Get("agent-1")
	assert.True(t, got.LastHeartbeat.After(past))
	assert.Equal(t, AgentAvailable, got.Status)
}

func TestRedisRegistry_Heartbeat_Good_RecoverFromOffline(t *testing.T) {
	reg := newTestRedisRegistry(t)
	_ = reg.Register(AgentInfo{
		ID:     "agent-1",
		Status: AgentOffline,
	})

	err := reg.Heartbeat("agent-1")
	require.NoError(t, err)

	got, _ := reg.Get("agent-1")
	assert.Equal(t, AgentAvailable, got.Status)
}

func TestRedisRegistry_Heartbeat_Good_BusyStaysBusy(t *testing.T) {
	reg := newTestRedisRegistry(t)
	_ = reg.Register(AgentInfo{
		ID:     "agent-1",
		Status: AgentBusy,
	})

	err := reg.Heartbeat("agent-1")
	require.NoError(t, err)

	got, _ := reg.Get("agent-1")
	assert.Equal(t, AgentBusy, got.Status)
}

func TestRedisRegistry_Heartbeat_Bad_NotFound(t *testing.T) {
	reg := newTestRedisRegistry(t)
	err := reg.Heartbeat("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

// --- Reap tests ---

func TestRedisRegistry_Reap_Good_StaleAgent(t *testing.T) {
	reg := newTestRedisRegistry(t)
	stale := time.Now().UTC().Add(-10 * time.Minute)
	fresh := time.Now().UTC()

	_ = reg.Register(AgentInfo{ID: "stale-1", Status: AgentAvailable, LastHeartbeat: stale})
	_ = reg.Register(AgentInfo{ID: "fresh-1", Status: AgentAvailable, LastHeartbeat: fresh})

	reaped := reg.Reap(5 * time.Minute)
	assert.Len(t, reaped, 1)
	assert.Contains(t, reaped, "stale-1")

	got, _ := reg.Get("stale-1")
	assert.Equal(t, AgentOffline, got.Status)

	got, _ = reg.Get("fresh-1")
	assert.Equal(t, AgentAvailable, got.Status)
}

func TestRedisRegistry_Reap_Good_AlreadyOfflineSkipped(t *testing.T) {
	reg := newTestRedisRegistry(t)
	stale := time.Now().UTC().Add(-10 * time.Minute)

	_ = reg.Register(AgentInfo{ID: "already-off", Status: AgentOffline, LastHeartbeat: stale})

	reaped := reg.Reap(5 * time.Minute)
	assert.Empty(t, reaped)
}

func TestRedisRegistry_Reap_Good_NoStaleAgents(t *testing.T) {
	reg := newTestRedisRegistry(t)
	now := time.Now().UTC()

	_ = reg.Register(AgentInfo{ID: "a", Status: AgentAvailable, LastHeartbeat: now})
	_ = reg.Register(AgentInfo{ID: "b", Status: AgentBusy, LastHeartbeat: now})

	reaped := reg.Reap(5 * time.Minute)
	assert.Empty(t, reaped)
}

func TestRedisRegistry_Reap_Good_BusyAgentReaped(t *testing.T) {
	reg := newTestRedisRegistry(t)
	stale := time.Now().UTC().Add(-10 * time.Minute)

	_ = reg.Register(AgentInfo{ID: "busy-stale", Status: AgentBusy, LastHeartbeat: stale})

	reaped := reg.Reap(5 * time.Minute)
	assert.Len(t, reaped, 1)
	assert.Contains(t, reaped, "busy-stale")

	got, _ := reg.Get("busy-stale")
	assert.Equal(t, AgentOffline, got.Status)
}

// --- Concurrent access ---

func TestRedisRegistry_Concurrent_Good(t *testing.T) {
	reg := newTestRedisRegistry(t)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "agent-" + string(rune('a'+n%5))
			_ = reg.Register(AgentInfo{
				ID:            id,
				Name:          "Concurrent",
				Status:        AgentAvailable,
				LastHeartbeat: time.Now().UTC(),
			})
			_, _ = reg.Get(id)
			_ = reg.Heartbeat(id)
			_ = reg.List()
			_ = reg.Reap(1 * time.Minute)
		}(i)
	}
	wg.Wait()

	// No race conditions — test passes under -race.
	agents := reg.List()
	assert.True(t, len(agents) > 0)
}

// --- Constructor error case ---

func TestNewRedisRegistry_Bad_Unreachable(t *testing.T) {
	_, err := NewRedisRegistry("127.0.0.1:1") // almost certainly unreachable
	require.Error(t, err)
	apiErr, ok := err.(*APIError)
	require.True(t, ok, "expected *APIError")
	assert.Equal(t, 500, apiErr.Code)
	assert.Contains(t, err.Error(), "failed to connect to Redis")
}

// --- Config-based factory with redis backend ---

func TestNewAgentRegistryFromConfig_Good_Redis(t *testing.T) {
	cfg := RegistryConfig{
		RegistryBackend:   "redis",
		RegistryRedisAddr: testRedisAddr,
	}
	reg, err := NewAgentRegistryFromConfig(cfg)
	if err != nil {
		t.Skipf("Redis unavailable at %s: %v", testRedisAddr, err)
	}
	rr, ok := reg.(*RedisRegistry)
	assert.True(t, ok, "expected RedisRegistry")
	_ = rr.Close()
}
