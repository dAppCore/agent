// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExamplePrepSubsystem_Shutdown_process() {
	s := newPrepWithProcess()
	err := s.Shutdown(nil)
	core.Println(err == nil)
	// Output: true
}
