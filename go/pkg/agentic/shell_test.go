// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// interactiveShellArgs yields the TTY-allocating OCI exec argv every runtime shares.
func TestContainerShell_InteractiveArgs_Good(t *testing.T) {
	got := interactiveShellArgs("vz-core-go-io-task-5", "/bin/sh")
	want := []string{"exec", "-i", "-t", "vz-core-go-io-task-5", "/bin/sh"}
	core.AssertEqual(t, len(want), len(got))
	for i := range want {
		core.AssertEqual(t, want[i], got[i])
	}
}

// An empty id is rejected before any runtime resolution or exec is attempted.
func TestContainerShell_EmptyID_Bad(t *testing.T) {
	core.AssertFalse(t, ContainerShell(ShellRequest{ID: "   "}).OK)
}

// An unknown runtime is rejected without attempting an exec.
func TestContainerShell_UnknownRuntime_Bad(t *testing.T) {
	core.AssertFalse(t, ContainerShell(ShellRequest{ID: "x", Runtime: "kubernetes"}).OK)
}

// VZ returns a clean not-yet error (no panic, no exec) until the vsock PTY lane lands.
func TestContainerShell_VZNotYet_Ugly(t *testing.T) {
	r := ContainerShell(ShellRequest{ID: "vz-x", Runtime: RuntimeVZ})
	core.AssertFalse(t, r.OK)
	core.AssertTrue(t, core.Contains(r.Error(), "VZ"))
}
