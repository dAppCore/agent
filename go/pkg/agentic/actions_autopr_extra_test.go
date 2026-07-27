// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestAgentic_handleAutoPR_DisabledGate — auto-pr returns OK without acting when
// the feature is not enabled (the default).
func TestAgentic_handleAutoPR_DisabledGate(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	captureStdout(t, func() {
		core.AssertTrue(t, s.handleAutoPR(context.Background(), core.NewOptions()).OK)
	})
}
