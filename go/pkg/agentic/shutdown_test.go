// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestShutdown_Shutdown_Good(t *testing.T) {
	s := newPrepWithProcess()
	err := s.Shutdown(context.TODO())
	core.AssertNoError(t, err)
}

func TestShutdown_Shutdown_Bad_AlreadyFrozen(t *testing.T) {
	s := newPrepWithProcess()
	s.frozen = true
	err := s.Shutdown(context.TODO())
	core.AssertNoError(t, err)
}

func TestShutdown_Shutdown_Ugly_NilRuntime(t *testing.T) {
	s := &PrepSubsystem{}
	core.AssertNotPanics(t, func() {
		_ = s.Shutdown(context.TODO())
	})
}
