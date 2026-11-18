// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "github.com/dappcore/go"

func ExampleScanInput() {
	input := ScanInput{Org: "core", Limit: 10}
	core.Println(input.Org, input.Limit)
	// Output: core 10
}
