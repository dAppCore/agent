// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePlanFromIssueInput() {
	input := PlanFromIssueInput{Slug: "release-plan"}
	core.Println(input.Slug)
	// Output: release-plan
}
