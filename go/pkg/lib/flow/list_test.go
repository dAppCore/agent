// SPDX-License-Identifier: EUPL-1.2

package flow

import "testing"

func TestList_ListEmbedded_Good_OnlyReturnsParseableFlows(t *testing.T) {
	// Every returned flow must parse cleanly and carry a name or steps —
	// prose markdown without a YAML body must be skipped.
	for _, definition := range ListEmbedded() {
		if definition.Name == "" && len(definition.Steps) == 0 {
			t.Fatalf("ListEmbedded returned an empty flow: %+v", definition)
		}
	}
}

func TestList_ListEmbedded_Bad_SkipsProseMarkdown(t *testing.T) {
	// go.md is prose, not a structured flow, so it cannot appear by name.
	for _, definition := range ListEmbedded() {
		if definition.Name == "Go Build Flow" {
			t.Fatal("ListEmbedded surfaced a prose markdown file as a flow")
		}
	}
}
