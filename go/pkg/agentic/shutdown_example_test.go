// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

func ExamplePrepSubsystem_Shutdown_process() {
	s := newPrepWithProcess()
	err := s.Shutdown(context.TODO())
	core.Println(err == nil)
	// Output: true
}
