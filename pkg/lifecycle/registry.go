package lifecycle

import (
	"iter"
	"slices"
	"sync"
	"time"
)

// AgentStatus represents the availability state of an agent.
type AgentStatus string

const (
	// AgentAvailable indicates the agent is ready to accept tasks.
	AgentAvailable AgentStatus = "available"
	// AgentBusy indicates the agent is working but may accept more tasks.
	AgentBusy AgentStatus = "busy"
	// AgentOffline indicates the agent has not sent a heartbeat recently.
	AgentOffline AgentStatus = "offline"
)

// AgentInfo describes a registered agent and its current state.
type AgentInfo struct {
	// ID is the unique identifier for the agent.
	ID string `json:"id"`
	// Name is the human-readable name of the agent.
	Name string `json:"name"`
	// Capabilities lists what the agent can handle (e.g. "go", "testing", "frontend").
	Capabilities []string `json:"capabilities,omitempty"`
	// Status is the current availability state.
	Status AgentStatus `json:"status"`
	// LastHeartbeat is the last time the agent reported in.
	LastHeartbeat time.Time `json:"last_heartbeat"`
	// CurrentLoad is the number of active jobs the agent is running.
	CurrentLoad int `json:"current_load"`
	// MaxLoad is the maximum concurrent jobs the agent supports. 0 means unlimited.
	MaxLoad int `json:"max_load"`
}

// AgentRegistry manages the set of known agents and their health.
type AgentRegistry interface {
	// Register adds or updates an agent in the registry.
	Register(agent AgentInfo) error
	// Deregister removes an agent from the registry.
	Deregister(id string) error
	// Get returns a copy of the agent info for the given ID.
	Get(id string) (AgentInfo, error)
	// List returns a copy of all registered agents.
	List() []AgentInfo
	// All returns an iterator over all registered agents.
	All() iter.Seq[AgentInfo]
	// Heartbeat updates the agent's LastHeartbeat timestamp and sets status
	// to Available if the agent was previously Offline.
	Heartbeat(id string) error
	// Reap marks agents as Offline if their last heartbeat is older than ttl.
	// Returns the IDs of agents that were reaped.
	Reap(ttl time.Duration) []string
	// Reaped returns an iterator over the IDs of agents that were reaped.
	Reaped(ttl time.Duration) iter.Seq[string]
}

// MemoryRegistry is an in-memory AgentRegistry implementation guarded by a
// read-write mutex. It uses copy-on-read semantics consistent with MemoryStore.
type MemoryRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentInfo
}

// NewMemoryRegistry creates a new in-memory agent registry.
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		agents: make(map[string]*AgentInfo),
	}
}

// Register adds or updates an agent in the registry. Returns an error if the
// agent ID is empty.
func (r *MemoryRegistry) Register(agent AgentInfo) error {
	if agent.ID == "" {
		return &APIError{Code: 400, Message: "agent ID is required"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := agent
	r.agents[agent.ID] = &cp
	return nil
}

// Deregister removes an agent from the registry. Returns an error if the agent
// is not found.
func (r *MemoryRegistry) Deregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.agents[id]; !ok {
		return &APIError{Code: 404, Message: "agent not found: " + id}
	}
	delete(r.agents, id)
	return nil
}

// Get returns a copy of the agent info for the given ID. Returns an error if
// the agent is not found.
func (r *MemoryRegistry) Get(id string) (AgentInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.agents[id]
	if !ok {
		return AgentInfo{}, &APIError{Code: 404, Message: "agent not found: " + id}
	}
	return *a, nil
}

// List returns a copy of all registered agents.
func (r *MemoryRegistry) List() []AgentInfo {
	return slices.Collect(r.All())
}

// All returns an iterator over all registered agents.
func (r *MemoryRegistry) All() iter.Seq[AgentInfo] {
	return func(yield func(AgentInfo) bool) {
		r.mu.RLock()
		defer r.mu.RUnlock()
		for _, a := range r.agents {
			if !yield(*a) {
				return
			}
		}
	}
}

// Heartbeat updates the agent's LastHeartbeat timestamp. If the agent was
// Offline, it transitions to Available.
func (r *MemoryRegistry) Heartbeat(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.agents[id]
	if !ok {
		return &APIError{Code: 404, Message: "agent not found: " + id}
	}
	a.LastHeartbeat = time.Now().UTC()
	if a.Status == AgentOffline {
		a.Status = AgentAvailable
	}
	return nil
}

// Reap marks agents as Offline if their last heartbeat is older than ttl.
// Returns the IDs of agents that were reaped.
func (r *MemoryRegistry) Reap(ttl time.Duration) []string {
	return slices.Collect(r.Reaped(ttl))
}

// Reaped returns an iterator over the IDs of agents that were reaped.
func (r *MemoryRegistry) Reaped(ttl time.Duration) iter.Seq[string] {
	return func(yield func(string) bool) {
		r.mu.Lock()
		defer r.mu.Unlock()
		now := time.Now().UTC()
		for id, a := range r.agents {
			if a.Status != AgentOffline && now.Sub(a.LastHeartbeat) > ttl {
				a.Status = AgentOffline
				if !yield(id) {
					return
				}
			}
		}
	}
}
