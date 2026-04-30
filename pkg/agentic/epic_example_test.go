// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleEpicInput() {
	input := EpicInput{
		Repo:  "go-io",
		Title: "Port agentic plans",
		Tasks: []string{"Read PHP flow", "Implement Go MCP tools"},
	}
	core.Println(input.Repo)
	core.Println(len(input.Tasks))
	// Output:
	// go-io
	// 2
}
