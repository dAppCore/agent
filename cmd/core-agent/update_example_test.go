// SPDX-License-Identifier: EUPL-1.2

package main

import core "dappco.re/go/core"

func Example_updateChannel() {
	oldVersion := version
	version = "0.15.0-alpha"
	defer func() { version = oldVersion }()

	core.Println(updateChannel())
	// Output: prerelease
}
