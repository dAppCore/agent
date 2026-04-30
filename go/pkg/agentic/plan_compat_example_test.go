// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePlanCompatibilitySummary() {
	summary := PlanCompatibilitySummary{Slug: "release-plan", Status: "active"}
	core.Println(summary.Slug)
	// Output: release-plan
}
