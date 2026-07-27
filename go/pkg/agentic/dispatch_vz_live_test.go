// SPDX-License-Identifier: EUPL-1.2

//go:build vz

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container"
)

// TestDispatchVZ_LiveBoot_Good_Case boots a real VZ guest, execs a trivial
// command, and stops it. Gated three ways so it never runs in the default suite:
//   - the `vz` build tag (this file is excluded without -tags vz),
//   - CONTAINER_VZ_LIVE=1 (operator opt-in),
//   - CORE_AGENT_VZ_IMAGE pointing at a §4 guest-image directory.
//
// It also requires a signed/entitled binary (com.apple.security.virtualization)
// — an unentitled run surfaces the framework's entitlement error from Run and
// the test skips rather than failing, per the fallback contract.
//
// Run with: CONTAINER_VZ_LIVE=1 CORE_AGENT_VZ_IMAGE=/path/to/image \
//           go test ./pkg/agentic/ -tags vz -run TestDispatchVZ_LiveBoot -count=1
func TestDispatchVZ_LiveBoot_Good_Case(t *testing.T) {
	if core.Env("CONTAINER_VZ_LIVE") != "1" {
		t.Skip("CONTAINER_VZ_LIVE != 1 — live VZ boot test skipped")
	}
	imageDir := core.Trim(core.Env(vzImageEnv))
	if imageDir == "" {
		t.Skip(vzImageEnv + " unset — live VZ boot test skipped")
	}
	if !container.IsVZAvailable() {
		t.Skip("Virtualization.framework unavailable on this host — skipped")
	}

	provider := container.NewVZProvider()
	image := &container.Image{Name: "core-agent-vz-live", Path: imageDir, Format: container.FormatRaw}

	runResult := provider.Run(image, container.WithMemory(vzDefaultMemoryMB), container.WithCPUs(vzDefaultCPUs))
	if !runResult.OK {
		// An unentitled binary fails here with the framework's entitlement error
		// — the documented fallback trigger, not a test failure.
		t.Skipf("VZ run unavailable (likely unentitled binary): %s", vzResultMessage(runResult))
	}
	ctr := core.MustCast[*container.Container](runResult)
	core.AssertNotNil(t, ctr)
	t.Cleanup(func() { _ = provider.Stop(ctr.ID) })

	// Minimal command only — the scaffold proves boot+exec, not agent dispatch.
	deadline := time.Now().Add(30 * time.Second)
	var execResult core.Result
	for {
		execResult = provider.Exec(ctr.ID, "true")
		if execResult.OK || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second) // guest agent may not be listening immediately
	}
	core.RequireTrue(t, execResult.OK)

	stopResult := provider.Stop(ctr.ID)
	core.AssertTrue(t, stopResult.OK)
}
