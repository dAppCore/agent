// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
)

func Example_fileExists() {
	dir := core.MustCast[string]((&core.Fs{}).NewUnrestricted().TempDir("example"))
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	(&core.Fs{}).NewUnrestricted().Write(core.JoinPath(dir, "go.mod"), "module test")

	core.Println(fileExists(core.JoinPath(dir, "go.mod")))
	core.Println(fileExists(core.JoinPath(dir, "missing.txt")))
	// Output:
	// true
	// false
}
