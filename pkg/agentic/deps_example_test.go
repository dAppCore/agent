// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go/core"

func Example_forgeSSHURL() {
	core.Println(forgeSSHURL("core", "go-io"))
	// Output: ssh://git@forge.lthn.ai:2223/core/go-io.git
}

func Example_parseCoreDeps() {
	goMod := `require (
	dappco.re/go/core v0.8.0
	dappco.re/go/core/process v0.3.0
)`

	core.Println(len(parseCoreDeps(goMod)))
	// Output: 2
}
