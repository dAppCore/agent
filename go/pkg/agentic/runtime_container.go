// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
	"dappco.re/go/container"
)

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

// resolveOCIRuntime picks the best available OCI argv runtime, never vz. It is
// the landing target when the VZ fork falls back (SP2.4): the in-process path is
// unavailable, so the OCI `run --rm` path must take over without any chance of
// re-selecting vz (which has no argv form). Mirrors resolveContainerRuntime's
// apple→docker→podman order with vz excluded; docker is the final fallback so
// dispatch never silently breaks.
//
//	resolveOCIRuntime() // "apple" on macOS with Apple Containers, else "docker"
func resolveOCIRuntime() string {
	for _, candidate := range []string{RuntimeApple, RuntimeDocker, RuntimePodman} {
		if runtimeAvailable(candidate) {
			return candidate
		}
	}
	return RuntimeDocker
}

// vzDispatchEnabled gates whether `auto` may resolve to vz, and whether an
// explicit `vz` preference engages the in-process fork (SP2). It is true only
// when the framework is usable on this host (darwin + Apple silicon, classes
// resolved) AND the operator has opted in via CONTAINER_VZ_LIVE=1.
//
// The com.apple.security.virtualization entitlement cannot be probed before a
// VM is started (go-container RFC.vz.md §2.2), so this gate stops at "framework
// available + opt-in"; an unentitled binary still passes this gate and relies on
// the Run-time auto-fallback in spawnAgentVZ (SP2.4) to downgrade to OCI.
//
//	vzDispatchEnabled() // true on an Apple-silicon host with CONTAINER_VZ_LIVE=1
func vzDispatchEnabled() bool {
	return container.IsVZAvailable() && core.Env("CONTAINER_VZ_LIVE") == "1"
}
