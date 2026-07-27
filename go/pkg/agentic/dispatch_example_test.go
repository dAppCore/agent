// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
)

func Example_detectFinalStatus() {
	dir := core.MustCast[string]((&core.Fs{}).NewUnrestricted().TempDir("example-ws"))
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	// Exit code 0 → completed
	status, _ := detectFinalStatus(dir, 0, "exited")
	core.Println(status)
	// Output: completed
}

func Example_detectFinalStatus_failed() {
	dir := core.MustCast[string]((&core.Fs{}).NewUnrestricted().TempDir("example-ws"))
	defer (&core.Fs{}).NewUnrestricted().DeleteAll(dir)

	// Non-zero exit → failed
	status, _ := detectFinalStatus(dir, 1, "exited")
	core.Println(status)
	// Output: failed
}
