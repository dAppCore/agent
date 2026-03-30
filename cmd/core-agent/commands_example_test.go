// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"dappco.re/go/core"
)

func Example_registerAppCommands() {
	c := core.New(core.WithOptions(core.NewOptions(core.Option{Key: "name", Value: "core-agent"})))
	registerAppCommands(c)

	core.Println(len(c.Commands()))
	// Output: 3
}

func Example_applyLogLevel() {
	args := applyLogLevel([]string{"--debug", "status"})

	core.Println(args[0])
	// Output: status
}
