// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePipelineTrainingCaptureInput() {
	input := PipelineTrainingCaptureInput{Repo: "core/go-io", Number: 42}
	core.Println(input.Number)
	// Output: 42
}
