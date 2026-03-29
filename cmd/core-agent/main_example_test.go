// SPDX-License-Identifier: EUPL-1.2

package main

import core "dappco.re/go/core"

func Example_appVersion() {
	oldVersion := version
	version = "0.15.0"
	defer func() { version = oldVersion }()

	core.Println(appVersion())
	// Output: 0.15.0
}
