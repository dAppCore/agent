// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func Example_parseForgeArgs() {
	org, repo, num := parseForgeArgs(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "42"},
	))
	core.Println(org, repo, num)
	// Output: core go-io 42
}

func Example_fmtIndex() {
	core.Println(fmtIndex(42))
	// Output: 42
}
