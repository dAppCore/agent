// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExamplePlanCleanupOutput() {
	output := PlanCleanupOutput{Archived: 3}
	core.Println(output.Archived)
	// Output: 3
}
