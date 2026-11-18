// SPDX-License-Identifier: EUPL-1.2

package agent

import (
	core "github.com/dappcore/go"
)

func ExampleVersion() {
	oldVersion := Version
	Version = "0.15.0"
	defer func() { Version = oldVersion }()

	core.Println(Version)
	// Output: 0.15.0
}
