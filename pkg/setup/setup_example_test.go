// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
)

func Example_defaultBuildCommand() {
	core.Println(defaultBuildCommand(TypeGo))
	core.Println(defaultBuildCommand(TypePHP))
	// Output:
	// go build ./...
	// composer test
}

func Example_formatFlow() {
	core.Println(formatFlow(TypeNode))
	// Output:
	// - Build: `npm run build`
	// - Test: `npm test`
}
