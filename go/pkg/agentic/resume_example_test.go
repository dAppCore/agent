// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleResumeInput() {
	input := ResumeInput{Workspace: "core/go-io/task-5", Answer: "Use v2 API"}
	core.Println(input.Workspace)
	// Output: core/go-io/task-5
}
