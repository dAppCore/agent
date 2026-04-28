// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
)

func Example_baseAgent() {
	core.Println(baseAgent("codex:gpt-5.4"))
	core.Println(baseAgent("claude"))
	// Output:
	// codex
	// claude
}
