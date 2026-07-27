// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommandsPlatform_CmdGuards — the platform CLI command wrappers reject
// invocations missing their required identifier before touching the network.
func TestCommandsPlatform_CmdGuards(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdCreditsBalance(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdCreditsHistory(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdCreditsAward(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdFleetHeartbeat(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdFleetDeregister(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdFleetTaskAssign(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdFleetTaskComplete(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdSubscriptionBudget(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdSubscriptionUpdateBudget(core.NewOptions()).OK)
	})
}
