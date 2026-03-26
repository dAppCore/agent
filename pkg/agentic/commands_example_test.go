// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func Example_parseIntStr() {
	core.Println(parseIntStr("42"))
	core.Println(parseIntStr("abc"))
	core.Println(parseIntStr(""))
	// Output:
	// 42
	// 0
	// 0
}
