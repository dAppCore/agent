// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleCommitInput() {
	input := CommitInput{Workspace: "core/go-io/task-5"}
	core.Println(input.Workspace)
	// Output: core/go-io/task-5
}
