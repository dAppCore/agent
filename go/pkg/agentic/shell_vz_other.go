// SPDX-License-Identifier: EUPL-1.2

//go:build !darwin

package agentic

import core "dappco.re/go"

// vzInteractiveShell is darwin-only: the in-process Virtualization.framework
// provider exists only on Apple silicon, so a vz interactive shell cannot be
// served from any other host. docker/podman are the cross-platform path.
func vzInteractiveShell(id, shell string) core.Result {
	return core.Fail(core.E("agentic.vzInteractiveShell", "vz interactive shell is only available on macOS (Apple Virtualization.framework); use docker or podman", nil))
}
