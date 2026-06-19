// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestAgentic_PlanSprintIssue_Guards — plan read/get/update, sprint update/
// archive, and issue-comment action wrappers reject empty options.
func TestAgentic_PlanSprintIssue_Guards(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertFalse(t, s.handlePlanRead(ctx, core.NewOptions()).OK)
		core.AssertFalse(t, s.handlePlanGet(ctx, core.NewOptions()).OK)
		core.AssertFalse(t, s.handlePlanUpdateStatus(ctx, core.NewOptions()).OK)
		core.AssertFalse(t, s.handleSprintUpdate(ctx, core.NewOptions()).OK)
		core.AssertFalse(t, s.handleSprintArchive(ctx, core.NewOptions()).OK)
		core.AssertFalse(t, s.handleIssueRecordComment(ctx, core.NewOptions()).OK)
	})
}
