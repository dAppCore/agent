// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestAgentic_EpicFleet_Usage — epic rejects empty options; the fleet command
// prints usage and succeeds without connecting.
func TestAgentic_EpicFleet_Usage(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdEpic(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdFleet(core.NewOptions()).OK)
	})
}
