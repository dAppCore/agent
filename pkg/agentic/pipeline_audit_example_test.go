// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePipelineIssueRef() {
	ref := PipelineIssueRef{Number: 17, Title: "Add audit coverage"}
	core.Println(ref.Number)
	// Output: 17
}
