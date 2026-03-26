// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func ExamplePrepSubsystem_Poke() {
	s := newPrepWithProcess()
	s.pokeCh = make(chan struct{}, 1)

	s.Poke()
	core.Println(len(s.pokeCh))
	// Output: 1
}
