// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleFlowRunStepOutput() {
	step := FlowRunStepOutput{Name: "build", Success: true}
	core.Println(step.Name)
	// Output: build
}
