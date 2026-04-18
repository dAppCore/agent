// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func Example_parseIntString() {
	core.Println(parseIntString("42"))
	core.Println(parseIntString("abc"))
	core.Println(parseIntString(""))
	// Output:
	// 42
	// 0
	// 0
}
