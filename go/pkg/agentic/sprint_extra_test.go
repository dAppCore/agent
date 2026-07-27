// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestSprint_sprintUpdate_Bad_RequiresIdentifier — sprint update without an id
// or slug is rejected before any platform call.
func TestSprint_sprintUpdate_Bad_RequiresIdentifier(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.sprintUpdate(context.Background(), SprintUpdateInput{})
	core.AssertFalse(t, r.OK)
}
