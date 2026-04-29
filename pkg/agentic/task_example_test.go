// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleTaskUpdateInput() {
	input := TaskUpdateInput{PlanSlug: "release", PhaseOrder: 1, Status: "done"}
	core.Println(input.Status)
	// Output: done
}
