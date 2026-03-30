// SPDX-License-Identifier: EUPL-1.2

package main

import (
	agentpkg "dappco.re/go/agent"
	core "dappco.re/go/core"
)

func Example_updateChannel() {
	oldVersion := agentpkg.Version
	agentpkg.Version = "0.15.0-alpha"
	defer func() { agentpkg.Version = oldVersion }()

	core.Println(updateChannel())
	// Output: prerelease
}
