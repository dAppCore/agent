// SPDX-License-Identifier: EUPL-1.2

package agentic_test

import (
	core "dappco.re/go"

	"dappco.re/go/agent/pkg/agentic"
)

// ExampleContainerShell attaches the current terminal to a running container by
// name. A VZ guest is declined cleanly until the vsock PTY lane lands, so the
// example exercises that documented path deterministically — no guest boot.
func ExampleContainerShell() {
	r := agentic.ContainerShell(agentic.ShellRequest{ID: "vz-demo", Runtime: "vz"})
	core.Println(r.OK)
	// Output: false
}
