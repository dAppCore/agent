// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommands_Dispatchers_Usage — state/task/sprint command dispatchers print
// usage and succeed when invoked with no action.
func TestCommands_Dispatchers_Usage(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertTrue(t, s.cmdState(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdTask(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdSprint(core.NewOptions()).OK)
	})
}
