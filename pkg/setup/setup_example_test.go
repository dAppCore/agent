// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
)

func ExampleDetect() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	// Empty dir — unknown type
	core.Println(Detect(dir))
	// Output: unknown
}

func ExampleDetectAll() {
	dir := (&core.Fs{}).NewUnrestricted().TempDir("example")
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	// Create a Go project
	(&core.Fs{}).NewUnrestricted().Write(core.JoinPath(dir, "go.mod"), "module test")
	types := DetectAll(dir)
	core.Println(types)
	// Output: [go]
}

func ExampleRegister() {
	c := core.New(core.WithService(Register))

	svc, ok := core.ServiceFor[*Service](c, "setup")
	core.Println(ok)
	_ = svc
	// Output: true
}
