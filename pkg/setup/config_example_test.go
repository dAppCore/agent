// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go"
)

func ExampleGenerateBuildConfig() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	config := GenerateBuildConfig(dir, TypeGo)
	core.Println(config.OK)
	core.Println(core.Contains(config.Value.(string), "type: go"))
	// Output:
	// true
	// true
}

func ExampleGenerateTestConfig() {
	config := GenerateTestConfig(TypeGo)
	core.Println(config.OK)
	core.Println(core.Contains(config.Value.(string), "go test"))
	// Output:
	// true
	// true
}
