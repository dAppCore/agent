// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommands_SubHandlers_Guards — the sprint/state/task sub-handlers reject
// invocations missing their required plan/slug identifier.
func TestCommands_SubHandlers_Guards(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdSprintGet(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdSprintUpdate(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdSprintArchive(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdStateGet(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdStateSet(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdStateList(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdStateDelete(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdTaskCreate(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdTaskToggle(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdTaskUpdate(core.NewOptions()).OK)
	})
}
