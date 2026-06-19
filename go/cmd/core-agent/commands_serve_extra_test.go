// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lemma"
)

// TestServe_buildAdmin_Good — a base-url + token builds an admin client.
func TestServe_buildAdmin_Good(t *testing.T) {
	var admin *lemma.Admin
	var ok bool
	captureStdout(t, func() {
		admin, ok = buildAdmin(core.NewOptions(
			core.Option{Key: "base-url", Value: "http://localhost:11434"},
			core.Option{Key: "admin-token", Value: "tok"},
		))
	})
	core.AssertTrue(t, ok)
	core.AssertTrue(t, admin != nil)
}
