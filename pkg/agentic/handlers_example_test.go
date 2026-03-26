// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func Example_resolveWorkspace() {
	// Non-existent workspace → empty string
	resolved := resolveWorkspace("nonexistent/workspace")
	core.Println(resolved == "")
	// Output: true
}
