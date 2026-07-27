// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/container"
)

// Detect always returns a runtime record (RuntimeNone when nothing is found)
// — never panics, never an empty Type.
func TestRuntimeContainer_Detect_Good(t *testing.T) {
	rt := container.Detect()
	core.AssertNotEmpty(t, string(rt.Type))
}

// Docker/podman availability via the seam agrees with go-container's HasRuntime.
func TestRuntimeContainer_Available_Good(t *testing.T) {
	core.AssertEqual(t, container.HasRuntime(container.RuntimeDocker), containerRuntimeAvailable("docker"))
	core.AssertEqual(t, container.HasRuntime(container.RuntimePodman), containerRuntimeAvailable("podman"))
}

// Unknown runtimes are never available through the seam.
func TestRuntimeContainer_Available_Bad(t *testing.T) {
	core.AssertFalse(t, containerRuntimeAvailable(""))
	core.AssertFalse(t, containerRuntimeAvailable("kubernetes"))
}
