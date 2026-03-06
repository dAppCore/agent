package lifecycle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeAgent(id string, status AgentStatus, caps []string, load, maxLoad int) AgentInfo {
	return AgentInfo{
		ID:            id,
		Name:          id,
		Capabilities:  caps,
		Status:        status,
		LastHeartbeat: time.Now().UTC(),
		CurrentLoad:   load,
		MaxLoad:       maxLoad,
	}
}

// --- Capability matching ---

func TestDefaultRouter_Route_Good_MatchesCapabilities(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Labels: []string{"go", "testing"}}
	agents := []AgentInfo{
		makeAgent("agent-a", AgentAvailable, []string{"go", "testing", "frontend"}, 0, 5),
		makeAgent("agent-b", AgentAvailable, []string{"python"}, 0, 5),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "agent-a", id)
}

func TestDefaultRouter_Route_Good_NoLabelsMatchesAll(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Labels: nil}
	agents := []AgentInfo{
		makeAgent("agent-a", AgentAvailable, []string{"go"}, 0, 5),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "agent-a", id)
}

func TestDefaultRouter_Route_Good_EmptyLabelsMatchesAll(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Labels: []string{}}
	agents := []AgentInfo{
		makeAgent("agent-a", AgentAvailable, nil, 0, 5),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "agent-a", id)
}

// --- Availability filtering ---

func TestDefaultRouter_Route_Good_SkipsOfflineAgents(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1"}
	agents := []AgentInfo{
		makeAgent("offline-1", AgentOffline, nil, 0, 5),
		makeAgent("online-1", AgentAvailable, nil, 0, 5),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "online-1", id)
}

func TestDefaultRouter_Route_Good_BusyWithCapacity(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1"}
	agents := []AgentInfo{
		makeAgent("busy-1", AgentBusy, nil, 2, 5), // has capacity
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "busy-1", id)
}

func TestDefaultRouter_Route_Good_BusyUnlimited(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1"}
	agents := []AgentInfo{
		makeAgent("busy-unlimited", AgentBusy, nil, 10, 0), // MaxLoad 0 = unlimited
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "busy-unlimited", id)
}

func TestDefaultRouter_Route_Bad_BusyAtCapacity(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1"}
	agents := []AgentInfo{
		makeAgent("full-1", AgentBusy, nil, 5, 5), // at capacity
	}

	_, err := router.Route(task, agents)
	require.ErrorIs(t, err, ErrNoEligibleAgent)
}

func TestDefaultRouter_Route_Bad_NoAgents(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1"}

	_, err := router.Route(task, nil)
	require.ErrorIs(t, err, ErrNoEligibleAgent)
}

func TestDefaultRouter_Route_Bad_NoCapableAgent(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Labels: []string{"rust"}}
	agents := []AgentInfo{
		makeAgent("go-agent", AgentAvailable, []string{"go"}, 0, 5),
		makeAgent("py-agent", AgentAvailable, []string{"python"}, 0, 5),
	}

	_, err := router.Route(task, agents)
	require.ErrorIs(t, err, ErrNoEligibleAgent)
}

func TestDefaultRouter_Route_Bad_AllOffline(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1"}
	agents := []AgentInfo{
		makeAgent("off-1", AgentOffline, nil, 0, 5),
		makeAgent("off-2", AgentOffline, nil, 0, 5),
	}

	_, err := router.Route(task, agents)
	require.ErrorIs(t, err, ErrNoEligibleAgent)
}

// --- Load balancing ---

func TestDefaultRouter_Route_Good_LeastLoaded(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Priority: PriorityMedium}
	agents := []AgentInfo{
		makeAgent("agent-a", AgentAvailable, nil, 3, 10),
		makeAgent("agent-b", AgentAvailable, nil, 1, 10),
		makeAgent("agent-c", AgentAvailable, nil, 5, 10),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	// agent-b has score 0.9, agent-a has 0.7, agent-c has 0.5
	assert.Equal(t, "agent-b", id)
}

func TestDefaultRouter_Route_Good_UnlimitedGetsMaxScore(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Priority: PriorityLow}
	agents := []AgentInfo{
		makeAgent("limited", AgentAvailable, nil, 1, 10), // score 0.9
		makeAgent("unlimited", AgentAvailable, nil, 5, 0), // score 1.0
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "unlimited", id)
}

// --- Critical priority ---

func TestDefaultRouter_Route_Good_CriticalPicksLeastLoaded(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Priority: PriorityCritical}
	agents := []AgentInfo{
		makeAgent("agent-a", AgentAvailable, nil, 4, 10),
		makeAgent("agent-b", AgentAvailable, nil, 1, 5), // lowest absolute load
		makeAgent("agent-c", AgentAvailable, nil, 2, 10),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	// Critical: picks least loaded by CurrentLoad, not by ratio.
	assert.Equal(t, "agent-b", id)
}

// --- Tie-breaking ---

func TestDefaultRouter_Route_Good_TieBreakByID(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Priority: PriorityMedium}
	agents := []AgentInfo{
		makeAgent("charlie", AgentAvailable, nil, 0, 5),
		makeAgent("alpha", AgentAvailable, nil, 0, 5),
		makeAgent("bravo", AgentAvailable, nil, 0, 5),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "alpha", id)
}

func TestDefaultRouter_Route_Good_CriticalTieBreakByID(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Priority: PriorityCritical}
	agents := []AgentInfo{
		makeAgent("charlie", AgentAvailable, nil, 0, 5),
		makeAgent("alpha", AgentAvailable, nil, 0, 5),
		makeAgent("bravo", AgentAvailable, nil, 0, 5),
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	assert.Equal(t, "alpha", id)
}

// --- Mixed scenarios ---

func TestDefaultRouter_Route_Good_MixedStatusAndCapabilities(t *testing.T) {
	router := NewDefaultRouter()
	task := &Task{ID: "t1", Labels: []string{"go"}, Priority: PriorityHigh}
	agents := []AgentInfo{
		makeAgent("offline-go", AgentOffline, []string{"go"}, 0, 5),
		makeAgent("busy-py", AgentBusy, []string{"python"}, 1, 5),
		makeAgent("busy-go-full", AgentBusy, []string{"go"}, 5, 5),  // at capacity
		makeAgent("busy-go-room", AgentBusy, []string{"go"}, 2, 5),  // has room
		makeAgent("avail-go", AgentAvailable, []string{"go"}, 1, 5), // available
	}

	id, err := router.Route(task, agents)
	require.NoError(t, err)
	// avail-go: score = 1.0 - 1/5 = 0.8
	// busy-go-room: score = 1.0 - 2/5 = 0.6
	assert.Equal(t, "avail-go", id)
}
