// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func ExamplePrepSubsystem_Poke() {
	// Poke is now a no-op — queue poke is owned by pkg/runner.Service.
	s := newPrepWithProcess()
	s.pokeCh = make(chan struct{}, 1)

	s.Poke()
	core.Println(len(s.pokeCh))
	// Output: 0
}
