// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

func ExampleConcurrencyLimit_UnmarshalYAML() {
	input := `
total: 5
gpt-5.4: 1
`
	var limit ConcurrencyLimit
	_ = yaml.Unmarshal([]byte(input), &limit)

	core.Println(limit.Total)
	core.Println(limit.Models["gpt-5.4"])
	// Output:
	// 5
	// 1
}
