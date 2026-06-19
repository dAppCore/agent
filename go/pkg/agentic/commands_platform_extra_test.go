// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommandsPlatform_printFleetTask_Good — the fleet-task printer renders the
// task fields.
func TestCommandsPlatform_printFleetTask_Good(t *testing.T) {
	out := captureStdout(t, func() {
		printFleetTask(FleetTask{ID: 7, Repo: "go-io", Status: "running", Branch: "dev", AgentModel: "codex", Task: "fix"})
	})
	core.AssertContains(t, out, "go-io")
	core.AssertContains(t, out, "running")
	core.AssertContains(t, out, "codex")
	core.AssertContains(t, out, "fix")
}
