// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestFleetConnect_RuntimeSnapshot — the fleet runtime snapshot persists +
// reloads connection/event state; an absent snapshot reads as offline.
func TestFleetConnect_RuntimeSnapshot(t *testing.T) {
	testPrepWithCore(t, nil) // sets a temp workspace for the snapshot path

	core.AssertEqual(t, "offline", loadFleetRuntimeSnapshot().State)

	fleetRememberConnected()
	fleetRememberEvent(FleetEvent{Repo: "go-io", TaskID: 1})

	core.AssertNotEmpty(t, loadFleetRuntimeSnapshot().LastConnectedAt)
}
