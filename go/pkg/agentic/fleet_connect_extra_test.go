// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestFleetConnect_fleetTaskFromEvent_Good — the event→task mapper copies the
// envelope fields and pulls task/template/agent_model from the payload.
func TestFleetConnect_fleetTaskFromEvent_Good(t *testing.T) {
	task := fleetTaskFromEvent(FleetEvent{
		Repo:   "go-io",
		Branch: "dev",
		Status: "running",
		Payload: map[string]any{
			"task":        "fix",
			"template":    "coding",
			"agent_model": "codex",
		},
	})
	core.AssertEqual(t, "go-io", task.Repo)
	core.AssertEqual(t, "dev", task.Branch)
	core.AssertEqual(t, "running", task.Status)
	core.AssertEqual(t, "fix", task.Task)
	core.AssertEqual(t, "coding", task.Template)
	core.AssertEqual(t, "codex", task.AgentModel)
}

// TestFleetConnect_fleetSnapshotEmpty_Good — a zero snapshot is empty; any set
// field makes it non-empty.
func TestFleetConnect_fleetSnapshotEmpty_Good(t *testing.T) {
	core.AssertTrue(t, fleetSnapshotEmpty(fleetRuntimeSnapshot{}))
	core.AssertFalse(t, fleetSnapshotEmpty(fleetRuntimeSnapshot{AgentID: "a"}))
}
