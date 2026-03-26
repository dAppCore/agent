// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func ExamplePrepInput() {
	input := PrepInput{Repo: "go-io", Issue: 42}
	core.Println(input.Repo, input.Issue)
	// Output: go-io 42
}

func ExampleNewPrep() {
	prep := NewPrep()
	core.Println(prep != nil)
	// Output: true
}
