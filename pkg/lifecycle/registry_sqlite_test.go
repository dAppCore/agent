package lifecycle

import (
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSQLiteRegistry creates a SQLiteRegistry backed by :memory: for testing.
func newTestSQLiteRegistry(t *testing.T) *SQLiteRegistry {
	t.Helper()
	reg, err := NewSQLiteRegistry(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = reg.Close() })
	return reg
}

// --- Register tests ---

func TestSQLiteRegistry_Register_Good(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
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

func TestSQLiteRegistry_Register_Good_Overwrite(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	_ = reg.Register(AgentInfo{ID: "agent-1", Name: "Original", MaxLoad: 3})
	err := reg.Register(AgentInfo{ID: "agent-1", Name: "Updated", MaxLoad: 10})
	require.NoError(t, err)

	got, err := reg.Get("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)
	assert.Equal(t, 10, got.MaxLoad)
}

func TestSQLiteRegistry_Register_Bad_EmptyID(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	err := reg.Register(AgentInfo{ID: "", Name: "No ID"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent ID is required")
}

func TestSQLiteRegistry_Register_Good_NilCapabilities(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	err := reg.Register(AgentInfo{
		ID:           "agent-1",
		Name:         "No Caps",
		Capabilities: nil,
		Status:       AgentAvailable,
	})
	require.NoError(t, err)

	got, err := reg.Get("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "No Caps", got.Name)
	// nil capabilities serialised as JSON null, deserialised back to nil.
}

// --- Deregister tests ---

func TestSQLiteRegistry_Deregister_Good(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	_ = reg.Register(AgentInfo{ID: "agent-1", Name: "To Remove"})

	err := reg.Deregister("agent-1")
	require.NoError(t, err)

	_, err = reg.Get("agent-1")
	require.Error(t, err)
}

func TestSQLiteRegistry_Deregister_Bad_NotFound(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	err := reg.Deregister("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

// --- Get tests ---

func TestSQLiteRegistry_Get_Good(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
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
	// Heartbeat stored via RFC3339Nano — allow small time difference from serialisation.
	assert.WithinDuration(t, now, got.LastHeartbeat, time.Millisecond)
}

func TestSQLiteRegistry_Get_Bad_NotFound(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	_, err := reg.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

func TestSQLiteRegistry_Get_Good_ReturnsCopy(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	_ = reg.Register(AgentInfo{ID: "agent-1", Name: "Original", CurrentLoad: 1})

	got, _ := reg.Get("agent-1")
	got.CurrentLoad = 99
	got.Name = "Tampered"

	// Re-read — should be unchanged.
	again, _ := reg.Get("agent-1")
	assert.Equal(t, "Original", again.Name)
	assert.Equal(t, 1, again.CurrentLoad)
}

// --- List tests ---

func TestSQLiteRegistry_List_Good_Empty(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	agents := reg.List()
	assert.Empty(t, agents)
}

func TestSQLiteRegistry_List_Good_Multiple(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
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

func TestSQLiteRegistry_Heartbeat_Good(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
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

func TestSQLiteRegistry_Heartbeat_Good_RecoverFromOffline(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	_ = reg.Register(AgentInfo{
		ID:     "agent-1",
		Status: AgentOffline,
	})

	err := reg.Heartbeat("agent-1")
	require.NoError(t, err)

	got, _ := reg.Get("agent-1")
	assert.Equal(t, AgentAvailable, got.Status)
}

func TestSQLiteRegistry_Heartbeat_Good_BusyStaysBusy(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	_ = reg.Register(AgentInfo{
		ID:     "agent-1",
		Status: AgentBusy,
	})

	err := reg.Heartbeat("agent-1")
	require.NoError(t, err)

	got, _ := reg.Get("agent-1")
	assert.Equal(t, AgentBusy, got.Status)
}

func TestSQLiteRegistry_Heartbeat_Bad_NotFound(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	err := reg.Heartbeat("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent not found")
}

// --- Reap tests ---

func TestSQLiteRegistry_Reap_Good_StaleAgent(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
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

func TestSQLiteRegistry_Reap_Good_AlreadyOfflineSkipped(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	stale := time.Now().UTC().Add(-10 * time.Minute)

	_ = reg.Register(AgentInfo{ID: "already-off", Status: AgentOffline, LastHeartbeat: stale})

	reaped := reg.Reap(5 * time.Minute)
	assert.Empty(t, reaped)
}

func TestSQLiteRegistry_Reap_Good_NoStaleAgents(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	now := time.Now().UTC()

	_ = reg.Register(AgentInfo{ID: "a", Status: AgentAvailable, LastHeartbeat: now})
	_ = reg.Register(AgentInfo{ID: "b", Status: AgentBusy, LastHeartbeat: now})

	reaped := reg.Reap(5 * time.Minute)
	assert.Empty(t, reaped)
}

func TestSQLiteRegistry_Reap_Good_BusyAgentReaped(t *testing.T) {
	reg := newTestSQLiteRegistry(t)
	stale := time.Now().UTC().Add(-10 * time.Minute)

	_ = reg.Register(AgentInfo{ID: "busy-stale", Status: AgentBusy, LastHeartbeat: stale})

	reaped := reg.Reap(5 * time.Minute)
	assert.Len(t, reaped, 1)
	assert.Contains(t, reaped, "busy-stale")

	got, _ := reg.Get("busy-stale")
	assert.Equal(t, AgentOffline, got.Status)
}

// --- Concurrent access ---

func TestSQLiteRegistry_Concurrent_Good(t *testing.T) {
	reg := newTestSQLiteRegistry(t)

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

// --- Persistence: close and reopen ---

func TestSQLiteRegistry_Persistence_Good(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "registry.db")

	// Phase 1: write data
	r1, err := NewSQLiteRegistry(dbPath)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Microsecond)
	_ = r1.Register(AgentInfo{
		ID:            "agent-1",
		Name:          "Persistent",
		Capabilities:  []string{"go", "rust"},
		Status:        AgentBusy,
		LastHeartbeat: now,
		CurrentLoad:   3,
		MaxLoad:       10,
	})
	require.NoError(t, r1.Close())

	// Phase 2: reopen and verify
	r2, err := NewSQLiteRegistry(dbPath)
	require.NoError(t, err)
	defer func() { _ = r2.Close() }()

	got, err := r2.Get("agent-1")
	require.NoError(t, err)
	assert.Equal(t, "Persistent", got.Name)
	assert.Equal(t, []string{"go", "rust"}, got.Capabilities)
	assert.Equal(t, AgentBusy, got.Status)
	assert.Equal(t, 3, got.CurrentLoad)
	assert.Equal(t, 10, got.MaxLoad)
	assert.WithinDuration(t, now, got.LastHeartbeat, time.Millisecond)
}

// --- Constructor error case ---

func TestNewSQLiteRegistry_Bad_InvalidPath(t *testing.T) {
	_, err := NewSQLiteRegistry("/nonexistent/deeply/nested/dir/registry.db")
	require.Error(t, err)
}

// --- Config-based factory ---

func TestNewAgentRegistryFromConfig_Good_Memory(t *testing.T) {
	cfg := RegistryConfig{RegistryBackend: "memory"}
	reg, err := NewAgentRegistryFromConfig(cfg)
	require.NoError(t, err)
	_, ok := reg.(*MemoryRegistry)
	assert.True(t, ok, "expected MemoryRegistry")
}

func TestNewAgentRegistryFromConfig_Good_Default(t *testing.T) {
	cfg := RegistryConfig{} // empty defaults to memory
	reg, err := NewAgentRegistryFromConfig(cfg)
	require.NoError(t, err)
	_, ok := reg.(*MemoryRegistry)
	assert.True(t, ok, "expected MemoryRegistry for empty config")
}

func TestNewAgentRegistryFromConfig_Good_SQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "factory-registry.db")
	cfg := RegistryConfig{
		RegistryBackend: "sqlite",
		RegistryPath:    dbPath,
	}
	reg, err := NewAgentRegistryFromConfig(cfg)
	require.NoError(t, err)
	sr, ok := reg.(*SQLiteRegistry)
	assert.True(t, ok, "expected SQLiteRegistry")
	_ = sr.Close()
}

func TestNewAgentRegistryFromConfig_Bad_UnknownBackend(t *testing.T) {
	cfg := RegistryConfig{RegistryBackend: "cassandra"}
	_, err := NewAgentRegistryFromConfig(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported registry backend")
}
