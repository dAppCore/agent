// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExamplePipelineEpicCreateInput() {
	input := PipelineEpicCreateInput{Repo: "core/go-io", Theme: "stability"}
	core.Println(input.Theme)
	// Output: stability
}
