// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func Example_truncate() {
	core.Println(truncate("hello world", 5))
	// Output: hello...
}
