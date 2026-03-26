// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func Example_extractField() {
	json := `{"status":"completed","repo":"go-io"}`
	core.Println(extractField(json, "status"))
	core.Println(extractField(json, "repo"))
	// Output:
	// completed
	// go-io
}
