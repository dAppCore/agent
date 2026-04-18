// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func ExampleReviewQueueInput() {
	input := ReviewQueueInput{Limit: 4, Reviewer: "coderabbit"}
	core.Println(input.Reviewer, input.Limit)
	// Output: coderabbit 4
}
