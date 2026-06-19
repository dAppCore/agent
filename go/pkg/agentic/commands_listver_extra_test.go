// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestAgentic_ListVersionCleanup_Good — plan-list, lang-list, and plan-cleanup
// succeed against an empty workspace; prompt-version rejects empty options.
func TestAgentic_ListVersionCleanup_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertTrue(t, s.handlePlanList(ctx, core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdLangList(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdPlanCleanup(core.NewOptions()).OK)
		core.AssertFalse(t, s.handlePromptVersion(ctx, core.NewOptions()).OK)
	})
}
