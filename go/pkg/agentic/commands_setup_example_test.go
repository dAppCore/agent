// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExampleSetupInput() {
	input := SetupInput{Template: "review", DryRun: true}
	core.Println(input.Template)
	// Output: review
}
