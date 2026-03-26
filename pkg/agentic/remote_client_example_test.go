// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func ExampleDispatchInput_remote() {
	input := DispatchInput{Repo: "go-io", Task: "Fix tests", Agent: "codex"}
	core.Println(input.Agent)
	// Output: codex
}
