// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestCommands_MorePlatformGuards — the remaining platform/sprint command
// wrappers: stats/task-next/sprint-create/sprint-list fail on empty input;
// sync push/pull no-op successfully with an empty working set.
func TestCommands_MorePlatformGuards(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdFleetStats(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdFleetTaskNext(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdSprintCreate(core.NewOptions()).OK)
		core.AssertFalse(t, s.cmdSprintList(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdSyncPush(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdSyncPull(core.NewOptions()).OK)
	})
}
