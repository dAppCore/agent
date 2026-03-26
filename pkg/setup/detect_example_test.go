// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
)

func ExampleDetect_go() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	(&core.Fs{}).NewUnrestricted().Write(core.JoinPath(dir, "go.mod"), "module test")
	core.Println(Detect(dir))
	// Output: go
}

func ExampleDetect_php() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	(&core.Fs{}).NewUnrestricted().Write(core.JoinPath(dir, "composer.json"), "{}")
	core.Println(Detect(dir))
	// Output: php
}

func ExampleDetect_node() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	(&core.Fs{}).NewUnrestricted().Write(core.JoinPath(dir, "package.json"), "{}")
	core.Println(Detect(dir))
	// Output: node
}
