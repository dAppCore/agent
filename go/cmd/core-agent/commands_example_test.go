// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go"
)

func Example_registerApplicationCommands() {
	c := core.New(core.WithOption("name", "core-agent"))
	registerApplicationCommands(c)

	core.Println(len(c.Commands()))
	// Output: 11
}

func Example_applyLogLevel() {
	args := applyLogLevel([]string{"--debug", "status"})

	core.Println(args[0])
	// Output: status
}
