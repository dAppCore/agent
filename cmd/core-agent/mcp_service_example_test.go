// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go/core"
)

func Example_registerMCPService() {
	result := registerMCPService(core.New(core.WithOptions(core.NewOptions(core.Option{Key: "name", Value: "core-agent"}))))

	core.Println(result.OK)
	// Output: true
}
