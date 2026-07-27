// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestOpencodeHostModels_NilCore_Guard — the nil-core guard returns an
// error WITHOUT shelling out to `opencode models`. Covers the early-return
// branch that precedes any process spawn (the live-spawn path is exercised
// only against a real host opencode, out of scope for unit tests).
func TestOpencodeHostModels_NilCore_Guard(t *testing.T) {
	models, err := OpencodeHostModels(context.Background(), nil)
	core.AssertTrue(t, err != nil)
	core.AssertTrue(t, models == nil)
}
