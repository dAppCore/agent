package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GetStatus tests ---

func TestGetStatus_Good_AllNil(t *testing.T) {
	summary, err := GetStatus(context.Background(), nil, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, summary.Agents)
	assert.Equal(t, 0, summary.PendingTasks)
	assert.Equal(t, 0, summary.InProgressTasks)
	assert.Empty(t, summary.AllowanceRemaining)
}

func TestGetStatus_Good_RegistryOnly(t *testing.T) {
	reg := NewMemoryRegistry()
	_ = reg.Register(AgentInfo{
		ID:            "virgil",
		Name:          "Virgil",
		Status:        AgentAvailable,
		LastHeartbeat: time.Now().UTC(),
		MaxLoad:       5,
	})
	_ = reg.Register(AgentInfo{
		ID:            "charon",
		Name:          "Charon",
		Status:        AgentBusy,
		CurrentLoad:   3,
		MaxLoad:       5,
		LastHeartbeat: time.Now().UTC(),
	})

	summary, err := GetStatus(context.Background(), reg, nil, nil)
	require.NoError(t, err)
	assert.Len(t, summary.Agents, 2)
	assert.Equal(t, 0, summary.PendingTasks)
	assert.Equal(t, 0, summary.InProgressTasks)
}

func TestGetStatus_Good_FullSummary(t *testing.T) {
	// Set up mock server returning task counts.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")
		w.Header().Set("Content-Type", "application/json")
		switch status {
		case "pending":
			tasks := []Task{
				{ID: "t1", Status: StatusPending},
				{ID: "t2", Status: StatusPending},
				{ID: "t3", Status: StatusPending},
			}
			_ = json.NewEncoder(w).Encode(tasks)
		case "in_progress":
			tasks := []Task{
				{ID: "t4", Status: StatusInProgress},
			}
			_ = json.NewEncoder(w).Encode(tasks)
		default:
			_ = json.NewEncoder(w).Encode([]Task{})
		}
	}))
	defer server.Close()

	reg := NewMemoryRegistry()
	_ = reg.Register(AgentInfo{
		ID:            "virgil",
		Name:          "Virgil",
		Status:        AgentAvailable,
		LastHeartbeat: time.Now().UTC(),
		MaxLoad:       5,
	})
	_ = reg.Register(AgentInfo{
		ID:            "charon",
		Name:          "Charon",
		Status:        AgentBusy,
		CurrentLoad:   3,
		MaxLoad:       5,
		LastHeartbeat: time.Now().UTC(),
	})

	store := NewMemoryStore()
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "virgil",
		DailyTokenLimit: 50000,
	})
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "charon",
		DailyTokenLimit: 50000,
	})
	// Simulate charon has used 38000 tokens.
	_ = store.IncrementUsage("charon", 38000, 0)

	svc := NewAllowanceService(store)
	client := NewClient(server.URL, "test-token")

	summary, err := GetStatus(context.Background(), reg, client, svc)
	require.NoError(t, err)
	assert.Len(t, summary.Agents, 2)
	assert.Equal(t, 3, summary.PendingTasks)
	assert.Equal(t, 1, summary.InProgressTasks)
	assert.Equal(t, int64(50000), summary.AllowanceRemaining["virgil"])
	assert.Equal(t, int64(12000), summary.AllowanceRemaining["charon"])
}

func TestGetStatus_Good_UnlimitedAllowance(t *testing.T) {
	reg := NewMemoryRegistry()
	_ = reg.Register(AgentInfo{
		ID:            "darbs",
		Name:          "Darbs",
		Status:        AgentAvailable,
		LastHeartbeat: time.Now().UTC(),
		MaxLoad:       3,
	})

	store := NewMemoryStore()
	// DailyTokenLimit 0 means unlimited.
	_ = store.SetAllowance(&AgentAllowance{
		AgentID:         "darbs",
		DailyTokenLimit: 0,
	})
	svc := NewAllowanceService(store)

	summary, err := GetStatus(context.Background(), reg, nil, svc)
	require.NoError(t, err)
	// Unlimited: Check returns RemainingTokens = -1.
	assert.Equal(t, int64(-1), summary.AllowanceRemaining["darbs"])
}

func TestGetStatus_Good_AllowanceSkipsUnknownAgents(t *testing.T) {
	reg := NewMemoryRegistry()
	_ = reg.Register(AgentInfo{
		ID:            "unknown-agent",
		Name:          "Unknown",
		Status:        AgentAvailable,
		LastHeartbeat: time.Now().UTC(),
	})

	store := NewMemoryStore()
	// No allowance set for "unknown-agent" -- GetAllowance will error.
	svc := NewAllowanceService(store)

	summary, err := GetStatus(context.Background(), reg, nil, svc)
	require.NoError(t, err)
	// AllowanceRemaining should not have an entry for unknown-agent.
	_, exists := summary.AllowanceRemaining["unknown-agent"]
	assert.False(t, exists)
}

func TestGetStatus_Bad_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(APIError{Message: "server error"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	summary, err := GetStatus(context.Background(), nil, client, nil)
	assert.Error(t, err)
	assert.Nil(t, summary)
	assert.Contains(t, err.Error(), "pending tasks")
}

// --- FormatStatus tests ---

func TestFormatStatus_Good_Empty(t *testing.T) {
	s := &StatusSummary{
		AllowanceRemaining: make(map[string]int64),
	}
	output := FormatStatus(s)
	assert.Contains(t, output, "Agents: 0")
	assert.Contains(t, output, "Tasks:  0 pending, 0 in progress")
	// No agent table rows when there are no agents — only the summary lines.
	assert.NotContains(t, output, "Status")
}

func TestFormatStatus_Good_FullTable(t *testing.T) {
	s := &StatusSummary{
		Agents: []AgentInfo{
			{ID: "virgil", Status: AgentAvailable, CurrentLoad: 0, MaxLoad: 5},
			{ID: "charon", Status: AgentBusy, CurrentLoad: 3, MaxLoad: 5},
			{ID: "darbs", Status: AgentAvailable, CurrentLoad: 0, MaxLoad: 3},
		},
		PendingTasks:    5,
		InProgressTasks: 2,
		AllowanceRemaining: map[string]int64{
			"virgil": 45000,
			"charon": 12000,
			"darbs":  -1,
		},
	}

	output := FormatStatus(s)
	assert.Contains(t, output, "Agents: 3 (2 available, 1 busy)")
	assert.Contains(t, output, "Tasks:  5 pending, 2 in progress")
	assert.Contains(t, output, "virgil")
	assert.Contains(t, output, "available")
	assert.Contains(t, output, "45000 tokens")
	assert.Contains(t, output, "charon")
	assert.Contains(t, output, "busy")
	assert.Contains(t, output, "12000 tokens")
	assert.Contains(t, output, "darbs")
	assert.Contains(t, output, "unlimited")

	// Verify deterministic sort order (agents sorted by ID).
	lines := strings.Split(output, "\n")
	var agentLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "charon") || strings.HasPrefix(line, "darbs") || strings.HasPrefix(line, "virgil") {
			agentLines = append(agentLines, line)
		}
	}
	require.Len(t, agentLines, 3)
	assert.True(t, strings.HasPrefix(agentLines[0], "charon"))
	assert.True(t, strings.HasPrefix(agentLines[1], "darbs"))
	assert.True(t, strings.HasPrefix(agentLines[2], "virgil"))
}

func TestFormatStatus_Good_OfflineAgent(t *testing.T) {
	s := &StatusSummary{
		Agents: []AgentInfo{
			{ID: "offline-bot", Status: AgentOffline, CurrentLoad: 0, MaxLoad: 5},
		},
		AllowanceRemaining: map[string]int64{
			"offline-bot": 30000,
		},
	}

	output := FormatStatus(s)
	assert.Contains(t, output, "1 offline")
	assert.Contains(t, output, "offline-bot")
}

func TestFormatStatus_Good_UnlimitedMaxLoad(t *testing.T) {
	s := &StatusSummary{
		Agents: []AgentInfo{
			{ID: "unlimited", Status: AgentAvailable, CurrentLoad: 2, MaxLoad: 0},
		},
		AllowanceRemaining: map[string]int64{
			"unlimited": -1,
		},
	}

	output := FormatStatus(s)
	assert.Contains(t, output, "2/-")
	assert.Contains(t, output, "unlimited")
}

func TestFormatStatus_Good_UnknownAllowance(t *testing.T) {
	s := &StatusSummary{
		Agents: []AgentInfo{
			{ID: "mystery", Status: AgentAvailable, MaxLoad: 5},
		},
		AllowanceRemaining: make(map[string]int64),
	}

	output := FormatStatus(s)
	assert.Contains(t, output, "unknown")
}
