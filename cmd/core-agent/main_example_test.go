// SPDX-License-Identifier: EUPL-1.2

package main

import (
	agentpkg "dappco.re/go/agent"
	core "dappco.re/go/core"
)

func Example_newCoreAgent() {
	oldVersion := agentpkg.Version
	agentpkg.Version = "0.15.0"
	defer func() { agentpkg.Version = oldVersion }()

	c := newCoreAgent()
	core.Println(c.App().Name)
	core.Println(c.App().Version)
	core.Println(len(c.Commands()) >= 3)
	// Output:
	// core-agent
	// 0.15.0
	// true
}

func Example_applicationVersion() {
	oldVersion := agentpkg.Version
	agentpkg.Version = "0.15.0"
	defer func() { agentpkg.Version = oldVersion }()

	core.Println(applicationVersion())
	// Output: 0.15.0
}
