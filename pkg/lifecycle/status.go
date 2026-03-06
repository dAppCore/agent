package lifecycle

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"forge.lthn.ai/core/go-log"
)

// StatusSummary aggregates status from the agent registry, task client, and
// allowance service for CLI display.
type StatusSummary struct {
	// Agents is the list of registered agents.
	Agents []AgentInfo
	// PendingTasks is the count of tasks with StatusPending.
	PendingTasks int
	// InProgressTasks is the count of tasks with StatusInProgress.
	InProgressTasks int
	// AllowanceRemaining maps agent ID to remaining daily tokens. -1 means unlimited.
	AllowanceRemaining map[string]int64
}

// GetStatus aggregates status from the registry, client, and allowance service.
// Any of registry, client, or allowanceSvc can be nil -- those sections are
// simply skipped. Returns what we can collect without failing on nil components.
func GetStatus(ctx context.Context, registry AgentRegistry, client *Client, allowanceSvc *AllowanceService) (*StatusSummary, error) {
	const op = "agentic.GetStatus"

	summary := &StatusSummary{
		AllowanceRemaining: make(map[string]int64),
	}

	// Collect agents from registry.
	if registry != nil {
		summary.Agents = registry.List()
	}

	// Count tasks by status via client.
	if client != nil {
		pending, err := client.ListTasks(ctx, ListOptions{Status: StatusPending})
		if err != nil {
			return nil, log.E(op, "failed to list pending tasks", err)
		}
		summary.PendingTasks = len(pending)

		inProgress, err := client.ListTasks(ctx, ListOptions{Status: StatusInProgress})
		if err != nil {
			return nil, log.E(op, "failed to list in-progress tasks", err)
		}
		summary.InProgressTasks = len(inProgress)
	}

	// Collect allowance remaining per agent.
	if allowanceSvc != nil {
		for _, agent := range summary.Agents {
			check, err := allowanceSvc.Check(agent.ID, "")
			if err != nil {
				// Skip agents whose allowance cannot be resolved.
				continue
			}
			summary.AllowanceRemaining[agent.ID] = check.RemainingTokens
		}
	}

	return summary, nil
}

// FormatStatus renders the summary as a human-readable table string suitable
// for CLI output.
func FormatStatus(s *StatusSummary) string {
	var b strings.Builder

	// Count agents by status.
	available := 0
	busy := 0
	for _, a := range s.Agents {
		switch a.Status {
		case AgentAvailable:
			available++
		case AgentBusy:
			busy++
		}
	}

	total := len(s.Agents)
	statusParts := make([]string, 0, 2)
	if available > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d available", available))
	}
	if busy > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d busy", busy))
	}
	offline := total - available - busy
	if offline > 0 {
		statusParts = append(statusParts, fmt.Sprintf("%d offline", offline))
	}

	if len(statusParts) > 0 {
		fmt.Fprintf(&b, "Agents: %d (%s)\n", total, strings.Join(statusParts, ", "))
	} else {
		fmt.Fprintf(&b, "Agents: %d\n", total)
	}

	fmt.Fprintf(&b, "Tasks:  %d pending, %d in progress\n", s.PendingTasks, s.InProgressTasks)

	if len(s.Agents) > 0 {
		// Sort agents by ID for deterministic output.
		agents := slices.Clone(s.Agents)
		slices.SortFunc(agents, func(a, b AgentInfo) int {
			return cmp.Compare(a.ID, b.ID)
		})

		fmt.Fprintf(&b, "%-16s%-12s%-8s%s\n", "Agent", "Status", "Load", "Remaining")
		for _, a := range agents {
			load := fmt.Sprintf("%d/%d", a.CurrentLoad, a.MaxLoad)
			if a.MaxLoad == 0 {
				load = fmt.Sprintf("%d/-", a.CurrentLoad)
			}

			remaining := "unknown"
			if tokens, ok := s.AllowanceRemaining[a.ID]; ok {
				if tokens < 0 {
					remaining = "unlimited"
				} else {
					remaining = fmt.Sprintf("%d tokens", tokens)
				}
			}

			fmt.Fprintf(&b, "%-16s%-12s%-8s%s\n", a.ID, string(a.Status), load, remaining)
		}
	}

	return b.String()
}
