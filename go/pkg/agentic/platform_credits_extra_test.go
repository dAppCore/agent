// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPlatform_parseCreditBalance_Good — the credit-balance parser maps the
// agent_id field out of a response map.
func TestPlatform_parseCreditBalance_Good(t *testing.T) {
	cb := parseCreditBalance(map[string]any{"agent_id": "agent-7"})
	core.AssertEqual(t, "agent-7", cb.AgentID)
}

// TestPlatform_parseFleetStats_Good — the fleet-stats parser maps the numeric
// counters out of a response map.
func TestPlatform_parseFleetStats_Good(t *testing.T) {
	fs := parseFleetStats(map[string]any{"nodes_online": float64(3), "tasks_today": float64(12)})
	core.AssertEqual(t, 3, fs.NodesOnline)
	core.AssertEqual(t, 12, fs.TasksToday)
}
