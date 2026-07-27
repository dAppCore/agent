// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestShutdown_Handlers_Good — dispatch shutdown + shutdown-now succeed when no
// dispatch loop is running (graceful no-op).
func TestShutdown_Handlers_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertTrue(t, s.handleDispatchShutdown(ctx, core.NewOptions()).OK)
		core.AssertTrue(t, s.handleDispatchShutdownNow(ctx, core.NewOptions()).OK)
	})
}
