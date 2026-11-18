// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"testing"

	core "github.com/dappcore/go"
)

// TestLemmaSubsystem_Shutdown_Good — shutdown is a clean no-op (the subsystem
// holds no long-lived resources).
func TestLemmaSubsystem_Shutdown_Good(t *testing.T) {
	s := newLemmaSubsystem()
	core.AssertNoError(t, s.Shutdown(context.Background()))
}
