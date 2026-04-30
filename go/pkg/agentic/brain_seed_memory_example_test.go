// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleBrainSeedMemoryInput() {
	input := BrainSeedMemoryInput{AgentID: "codex", Path: "notes.md"}
	core.Println(input.AgentID)
	// Output: codex
}
