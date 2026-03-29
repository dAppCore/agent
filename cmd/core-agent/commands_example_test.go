// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go/core"
)

func Example_registerAppCommands() {
	c := core.New(core.WithOption("name", "core-agent"))
	registerAppCommands(c)

	core.Println(len(c.Commands()))
	// Output: 3
}
