// SPDX-License-Identifier: EUPL-1.2

package brain

import "testing"

func TestAlias_BrainMemory_Good(t *testing.T) {
	var memory BrainMemory
	memory.ID = "mem-123"
	memory.Type = "convention"

	if memory.ID != "mem-123" {
		t.Fatalf("expected BrainMemory alias to behave like Memory")
	}
	if memory.Type != "convention" {
		t.Fatalf("expected BrainMemory alias to behave like Memory")
	}
}
