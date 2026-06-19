// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPlatform_parseCreditBalance_Good — the credit-balance parser maps the
// agent_id field out of a response map.
func TestPlatform_parseCreditBalance_Good(t *testing.T) {
	cb := parseCreditBalance(map[string]any{"agent_id": "agent-7"})
	core.AssertEqual(t, "agent-7", cb.AgentID)
}
