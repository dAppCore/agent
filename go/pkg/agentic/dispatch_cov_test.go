// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

// --- dispatchTimeoutReason (minute / second / sub-second arms) ---

func TestDispatch_DispatchTimeoutReason_Good_WholeMinutes(t *testing.T) {
	core.AssertEqual(t, "Agent timed out after 5m", dispatchTimeoutReason(5*time.Minute))
	core.AssertEqual(t, "Agent timed out after 1m", dispatchTimeoutReason(time.Minute))
}

func TestDispatch_DispatchTimeoutReason_Good_WholeSeconds(t *testing.T) {
	// 90s is not a whole minute → falls to the seconds branch.
	core.AssertEqual(t, "Agent timed out after 90s", dispatchTimeoutReason(90*time.Second))
}

func TestDispatch_DispatchTimeoutReason_Ugly_SubSecond(t *testing.T) {
	// 1500ms is neither whole minutes nor whole seconds → duration string.
	got := dispatchTimeoutReason(1500 * time.Millisecond)
	core.AssertContains(t, got, "Agent timed out after")
	core.AssertContains(t, got, "1.5s")
}

func TestDispatch_DispatchTimeoutReason_Bad_Zero(t *testing.T) {
	// Zero timeout falls to the default duration-string arm.
	got := dispatchTimeoutReason(0)
	core.AssertContains(t, got, "Agent timed out after")
}

// --- dispatchTimeoutReasonFromWorkspace + clearDispatchTimeoutReason ---

func TestDispatch_TimeoutReasonFromWorkspace_Good_RoundTrip(t *testing.T) {
	wsDir := t.TempDir()
	metaDir := WorkspaceMetaDir(wsDir)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)
	core.RequireTrue(t, fs.Write(workspaceTimeoutPath(wsDir), "Agent timed out after 5m\n").OK)

	// Reads back trimmed.
	core.AssertEqual(t, "Agent timed out after 5m", dispatchTimeoutReasonFromWorkspace(wsDir))

	// Clearing removes the marker so the next dispatch starts clean.
	clearDispatchTimeoutReason(wsDir)
	core.AssertFalse(t, fs.Exists(workspaceTimeoutPath(wsDir)))
	core.AssertEmpty(t, dispatchTimeoutReasonFromWorkspace(wsDir))
}

func TestDispatch_TimeoutReasonFromWorkspace_Bad_NoMarker(t *testing.T) {
	// No marker file → empty string, and clearing a missing marker is a no-op.
	wsDir := t.TempDir()
	core.AssertEmpty(t, dispatchTimeoutReasonFromWorkspace(wsDir))
	core.AssertNotPanics(t, func() { clearDispatchTimeoutReason(wsDir) })
}

// --- localAgentCommandScript (LEM profile vs ollama arm) ---

func TestDispatch_LocalAgentCommandScript_Good_LEMProfile(t *testing.T) {
	// A known LEM profile routes through codex --profile, not --oss/ollama.
	script := localAgentCommandScript("lemmy", "Review the last commit")
	core.AssertContains(t, script, "--profile")
	core.AssertContains(t, script, "'lemmy'")
	core.AssertNotContains(t, script, "--oss")
}

func TestDispatch_LocalAgentCommandScript_Bad_OllamaModel(t *testing.T) {
	// A non-LEM model routes through the ollama local provider path.
	script := localAgentCommandScript("devstral-24b", "Do the thing")
	core.AssertContains(t, script, "--oss --local-provider ollama")
	core.AssertContains(t, script, "'devstral-24b'")
	core.AssertNotContains(t, script, "--profile")
}

// --- resolveContainerRuntime (vz fall-through without opt-in) ---

func TestDispatch_ResolveContainerRuntime_Ugly_VZWithoutOptIn(t *testing.T) {
	// Without CONTAINER_VZ_LIVE=1, a vz preference falls through to the OCI
	// auto path and never returns vz. The concrete result is host-dependent
	// (docker/podman/apple), so we only assert vz is never selected.
	t.Setenv("CONTAINER_VZ_LIVE", "")
	resolved := resolveContainerRuntime(RuntimeVZ)
	core.AssertNotEqual(t, RuntimeVZ, resolved)
}

func TestDispatch_ResolveContainerRuntime_Bad_EmptyFallsBackDocker(t *testing.T) {
	// An empty preference resolves via the auto order; docker is the guaranteed
	// final fallback so dispatch never silently breaks.
	resolved := resolveContainerRuntime("")
	core.AssertNotEqual(t, RuntimeVZ, resolved)
	core.AssertNotEmpty(t, resolved)
}

// --- runtimeAvailable (apple requires darwin) ---

func TestDispatch_RuntimeAvailable_Bad_AppleNotOnNonDarwin(t *testing.T) {
	// Apple Containers are gated on macOS; force the darwin flag false and the
	// apple runtime is reported unavailable without probing go-container.
	original := goosIsDarwin
	t.Cleanup(func() { goosIsDarwin = original })
	goosIsDarwin = false

	core.AssertFalse(t, runtimeAvailable(RuntimeApple))
}

// --- resolveOCIRuntime (never returns vz) ---

func TestDispatch_ResolveOCIRuntime_Good_NeverVZ(t *testing.T) {
	resolved := resolveOCIRuntime()
	core.AssertNotEqual(t, RuntimeVZ, resolved)
	core.AssertNotEmpty(t, resolved)
}
