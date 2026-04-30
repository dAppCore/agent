// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleDispatchInput() {
	input := DispatchInput{Repo: "go-io", Task: "Fix tests", Agent: "codex"}
	core.Println(input.Repo, input.Agent)
	// Output: go-io codex
}
