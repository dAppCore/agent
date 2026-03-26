// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
)

func ExampleGenerateBuildConfig() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	config, err := GenerateBuildConfig(dir, TypeGo)
	core.Println(err == nil)
	core.Println(core.Contains(config, "type: go"))
	// Output:
	// true
	// true
}

func ExampleGenerateTestConfig() {
	config, err := GenerateTestConfig(TypeGo)
	core.Println(err == nil)
	core.Println(core.Contains(config, "go test"))
	// Output:
	// true
	// true
}
