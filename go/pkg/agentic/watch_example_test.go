// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "github.com/dappcore/go"
)

func ExampleWorkspaceRoot_watch() {
	root := WorkspaceRoot()
	core.Println(core.HasSuffix(root, "workspace"))
	// Output: true
}
