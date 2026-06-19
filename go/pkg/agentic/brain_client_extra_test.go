// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestBrainClient_brainValuePresent_GoodBadUgly — nil and empty/whitespace
// stringify to absent; any non-empty value is present.
func TestBrainClient_brainValuePresent_GoodBadUgly(t *testing.T) {
	core.AssertFalse(t, brainValuePresent(nil))
	core.AssertFalse(t, brainValuePresent(""))
	core.AssertFalse(t, brainValuePresent("   "))
	core.AssertTrue(t, brainValuePresent("x"))
	core.AssertTrue(t, brainValuePresent(42))
}
