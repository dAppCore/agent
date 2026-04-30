// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

func Example_baseAgent() {
	core.Println(baseAgent("codex:gpt-5.4"))
	core.Println(baseAgent("claude"))
	// Output:
	// codex
	// claude
}

func ExampleConcurrencyLimit_UnmarshalYAML() {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte(`total: 3
codex: 2
`), &limit)
	core.Println(err == nil)
	core.Println(limit.Total)
	core.Println(limit.Models["codex"])
	// Output:
	// true
	// 3
	// 2
}
