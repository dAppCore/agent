// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShutdown_Shutdown_Good(t *testing.T) {
	s := newPrepWithProcess()
	err := s.Shutdown(nil)
	assert.NoError(t, err)
}

func TestShutdown_Shutdown_Bad_AlreadyFrozen(t *testing.T) {
	s := newPrepWithProcess()
	s.frozen = true
	err := s.Shutdown(nil)
	assert.NoError(t, err)
}

func TestShutdown_Shutdown_Ugly_NilRuntime(t *testing.T) {
	s := &PrepSubsystem{}
	assert.NotPanics(t, func() {
		_ = s.Shutdown(nil)
	})
}
