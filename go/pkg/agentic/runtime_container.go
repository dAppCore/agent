// SPDX-License-Identifier: EUPL-1.2

package agentic

import "dappco.re/go/container"

// containerRuntimeAvailable reports whether a runtime is usable on this host,
// delegating to go-container's detection (single source of truth, replaces the
// old $PATH probe). Unknown names are never available.
//
//	containerRuntimeAvailable("docker") // true if dockerd reachable
func containerRuntimeAvailable(name string) bool {
	switch name {
	case RuntimeApple, RuntimeVZ, RuntimeDocker, RuntimePodman:
		return container.HasRuntime(container.RuntimeType(name))
	default:
		return false
	}
}

// runtimeUsesProvider reports whether a runtime is driven through go-container's
// in-process provider (vz) rather than the OCI argv path (docker/apple/podman).
//
//	runtimeUsesProvider("vz") // true
func runtimeUsesProvider(name string) bool { return name == RuntimeVZ }

// vzDispatchEnabled gates whether `auto` may resolve to vz. SP1 keeps it OFF so
// the OCI path is unchanged; SP2 flips it on once the boot fork exists.
func vzDispatchEnabled() bool { return false }
