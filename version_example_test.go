// SPDX-License-Identifier: EUPL-1.2

package agent

import (
	core "dappco.re/go/core"
)

func ExampleVersion() {
	oldVersion := Version
	Version = "0.15.0"
	defer func() { Version = oldVersion }()

	core.Println(Version)
	// Output: 0.15.0
}
