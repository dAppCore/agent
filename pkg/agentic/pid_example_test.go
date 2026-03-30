// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"

	core "dappco.re/go/core"
)

func ExamplePIDAlive() {
	core.Println(PIDAlive(os.Getpid()))
	// Output: true
}

func ExamplePIDTerminate() {
	core.Println(PIDTerminate(0))
	// Output: false
}
