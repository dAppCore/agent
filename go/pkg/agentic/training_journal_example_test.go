// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePipelineTrainingEntry() {
	entry := PipelineTrainingEntry{Repo: "go-io", PRNumber: 42}
	core.Println(entry.Repo)
	// Output: go-io
}
