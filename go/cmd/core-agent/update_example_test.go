// SPDX-License-Identifier: EUPL-1.2

package main

import (
	agentpkg "github.com/dappcore/agent/go"
	core "github.com/dappcore/go"
)

func Example_updateChannel() {
	oldVersion := agentpkg.Version
	agentpkg.Version = "0.15.0-alpha"
	defer func() { agentpkg.Version = oldVersion }()

	core.Println(updateChannel())
	// Output: prerelease
}
