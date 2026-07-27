// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestActions_ForgeHandlers_GuardOnEmptyOptions — each forge action handler
// delegates to its cmd* form, which refuses empty (no repo/number) input
// before any forge call.
func TestActions_ForgeHandlers_GuardOnEmptyOptions(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()
	handlers := []func(context.Context, core.Options) core.Result{
		s.handleIssueGet, s.handleIssueList, s.handleIssueCreate,
		s.handlePRGet, s.handlePRList, s.handlePRMerge, s.handlePRClose,
	}
	captureStdout(t, func() {
		for _, h := range handlers {
			core.AssertFalse(t, h(ctx, core.NewOptions()).OK)
		}
	})
}
