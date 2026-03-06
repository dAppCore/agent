package lifecycle

import (
	"cmp"
	"errors"
	"slices"
)

// ErrNoEligibleAgent is returned when no agent matches the task requirements.
var ErrNoEligibleAgent = errors.New("no eligible agent for task")

// TaskRouter selects an agent for a given task from a list of candidates.
type TaskRouter interface {
	// Route picks the best agent for the task and returns its ID.
	// Returns ErrNoEligibleAgent if no agent qualifies.
	Route(task *Task, agents []AgentInfo) (string, error)
}

// DefaultRouter implements TaskRouter with capability matching and load-based
// scoring. For critical priority tasks it picks the least-loaded agent directly.
type DefaultRouter struct{}

// NewDefaultRouter creates a new DefaultRouter.
func NewDefaultRouter() *DefaultRouter {
	return &DefaultRouter{}
}

// Route selects the best agent for the task:
//  1. Filter by availability (Available, or Busy with capacity).
//  2. Filter by capabilities (task.Labels must be a subset of agent.Capabilities).
//  3. For critical tasks, pick the least-loaded agent.
//  4. For other tasks, score by load ratio and pick the highest-scored agent.
//  5. Ties are broken by agent ID (alphabetical) for determinism.
func (r *DefaultRouter) Route(task *Task, agents []AgentInfo) (string, error) {
	eligible := r.filterEligible(task, agents)
	if len(eligible) == 0 {
		return "", ErrNoEligibleAgent
	}

	if task.Priority == PriorityCritical {
		return r.leastLoaded(eligible), nil
	}

	return r.highestScored(eligible), nil
}

// filterEligible returns agents that are available (or busy with room) and
// possess all required capabilities.
func (r *DefaultRouter) filterEligible(task *Task, agents []AgentInfo) []AgentInfo {
	var result []AgentInfo
	for _, a := range agents {
		if !r.isAvailable(a) {
			continue
		}
		if !r.hasCapabilities(a, task.Labels) {
			continue
		}
		result = append(result, a)
	}
	return result
}

// isAvailable returns true if the agent can accept work.
func (r *DefaultRouter) isAvailable(a AgentInfo) bool {
	switch a.Status {
	case AgentAvailable:
		return true
	case AgentBusy:
		// Busy agents can still accept work if they have capacity.
		return a.MaxLoad == 0 || a.CurrentLoad < a.MaxLoad
	default:
		return false
	}
}

// hasCapabilities checks that the agent has all required labels. If the task
// has no labels, any agent qualifies.
func (r *DefaultRouter) hasCapabilities(a AgentInfo, labels []string) bool {
	if len(labels) == 0 {
		return true
	}
	for _, label := range labels {
		if !slices.Contains(a.Capabilities, label) {
			return false
		}
	}
	return true
}

// leastLoaded picks the agent with the lowest CurrentLoad. Ties are broken by
// agent ID (alphabetical).
func (r *DefaultRouter) leastLoaded(agents []AgentInfo) string {
	// Sort: lowest load first, then by ID for determinism.
	slices.SortFunc(agents, func(a, b AgentInfo) int {
		if a.CurrentLoad != b.CurrentLoad {
			return cmp.Compare(a.CurrentLoad, b.CurrentLoad)
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return agents[0].ID
}

// highestScored picks the agent with the highest availability score.
// Score = 1.0 - (CurrentLoad / MaxLoad). If MaxLoad is 0, score is 1.0.
// Ties are broken by agent ID (alphabetical).
func (r *DefaultRouter) highestScored(agents []AgentInfo) string {
	type scored struct {
		id    string
		score float64
	}

	scores := make([]scored, len(agents))
	for i, a := range agents {
		s := 1.0
		if a.MaxLoad > 0 {
			s = 1.0 - float64(a.CurrentLoad)/float64(a.MaxLoad)
		}
		scores[i] = scored{id: a.ID, score: s}
	}

	// Sort: highest score first, then by ID for determinism.
	slices.SortFunc(scores, func(a, b scored) int {
		if a.score != b.score {
			return cmp.Compare(b.score, a.score) // highest first
		}
		return cmp.Compare(a.id, b.id)
	})

	return scores[0].id
}
