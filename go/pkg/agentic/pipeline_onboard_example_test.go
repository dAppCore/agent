// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePipelineOnboardInput() {
	input := PipelineOnboardInput{Repo: "core/go-io", Agent: "codex"}
	core.Println(input.Repo)
	// Output: core/go-io
}
