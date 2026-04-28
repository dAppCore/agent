// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePrepSubsystem_Shutdown() {
	s := newPrepWithProcess()
	err := s.Shutdown(nil)
	core.Println(err == nil)
	// Output: true
}
