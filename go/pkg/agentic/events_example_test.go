// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "github.com/dappcore/go"
)

func Example_emitStartEvent() {
	// Events are appended to workspace/events.jsonl
	// This exercises the path without requiring a real workspace
	root := WorkspaceRoot()
	core.Println(core.HasSuffix(root, "workspace"))
	// Output: true
}
