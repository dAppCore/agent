// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePrompt() {
	prompt := Prompt{Name: "review", Model: "gpt-5.4"}
	core.Println(prompt.Name)
	// Output: review
}
