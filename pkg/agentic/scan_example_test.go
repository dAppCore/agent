// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func ExampleScanInput() {
	input := ScanInput{Org: "core", Limit: 10}
	core.Println(input.Org, input.Limit)
	// Output: core 10
}
