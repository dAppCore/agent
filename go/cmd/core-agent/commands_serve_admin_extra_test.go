// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lemma"
)

// TestServe_buildAdmin_Bad_MissingTokenFile — buildAdmin's failure branch:
// an explicit --admin-token-file pointing at a non-existent path makes
// lemma.NewAdmin fail (the token can't be loaded), so buildAdmin prints the
// hint lines and returns (nil, false). Covers the error half of buildAdmin
// (the Good test covers the success half).
func TestServe_buildAdmin_Bad_MissingTokenFile(t *testing.T) {
	missing := core.JoinPath(t.TempDir(), "no-such-admin.token")

	var admin *lemma.Admin
	var ok bool
	out := captureStdout(t, func() {
		admin, ok = buildAdmin(core.NewOptions(
			core.Option{Key: "base-url", Value: "http://localhost:11434"},
			core.Option{Key: "admin-token-file", Value: missing},
		))
	})

	core.AssertFalse(t, ok)
	core.AssertTrue(t, admin == nil)
	core.AssertContains(t, out, "admin client:")
}
