// SPDX-License-Identifier: EUPL-1.2

package main

import core "dappco.re/go/core"

func Example_newCoreAgent() {
	oldVersion := version
	version = "0.15.0"
	defer func() { version = oldVersion }()

	c := newCoreAgent()
	core.Println(c.App().Name)
	core.Println(c.App().Version)
	core.Println(len(c.Commands()) >= 3)
	// Output:
	// core-agent
	// 0.15.0
	// true
}

func Example_appVersion() {
	oldVersion := version
	version = "0.15.0"
	defer func() { version = oldVersion }()

	core.Println(appVersion())
	// Output: 0.15.0
}
