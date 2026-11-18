// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExamplePhaseGetInput() {
	input := PhaseGetInput{PlanSlug: "release", PhaseOrder: 2}
	core.Println(input.PhaseOrder)
	// Output: 2
}
