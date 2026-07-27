// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestFleetMode_printFleetUsage_Good — the fleet usage printer emits output.
func TestFleetMode_printFleetUsage_Good(t *testing.T) {
	out := captureStdout(t, func() { printFleetUsage() })
	core.AssertNotEmpty(t, out)
}
