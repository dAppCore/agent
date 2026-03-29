// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go/core"
)

func Example_registerMCPService() {
	result := registerMCPService(core.New(core.WithOption("name", "core-agent")))

	core.Println(result.OK)
	// Output: true
}
