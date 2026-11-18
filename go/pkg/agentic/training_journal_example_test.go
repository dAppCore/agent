// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExamplePipelineTrainingEntry() {
	entry := PipelineTrainingEntry{Repo: "go-io", PRNumber: 42}
	core.Println(entry.Repo)
	// Output: go-io
}
