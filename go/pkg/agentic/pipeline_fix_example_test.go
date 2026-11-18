// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExamplePipelineFixInput() {
	input := PipelineFixInput{Repo: "core/go-io", Number: 42}
	core.Println(input.Number)
	// Output: 42
}
