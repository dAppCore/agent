// SPDX-License-Identifier: EUPL-1.2

package agentic_test

import (
	core "dappco.re/go"

	"dappco.re/go/agent/pkg/agentic"
)

// ExampleContainerShell attaches the current terminal to a running container by
// name. A missing id is rejected before any runtime work, shown here as the
// deterministic guard (the interactive paths need a live container + TTY).
func ExampleContainerShell() {
	r := agentic.ContainerShell(agentic.ShellRequest{})
	core.Println(r.OK)
	// Output: false
}
