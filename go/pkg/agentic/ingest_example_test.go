// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func Example_ingestWorkspaceRoot() {
	root := WorkspaceRoot()
	core.Println(core.HasSuffix(root, "workspace"))
	// Output: true
}
