// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunner_StartRunner_Good(t *testing.T) {
	s := newPrepWithProcess()
	assert.Nil(t, s.pokeCh)
	s.StartRunner()
	assert.NotNil(t, s.pokeCh)
}

func TestRunner_StartRunner_Bad_AlreadyRunning(t *testing.T) {
	s := newPrepWithProcess()
	s.StartRunner()
	// Second call should not panic
	assert.NotPanics(t, func() { s.StartRunner() })
}

func TestRunner_Poke_Ugly_NilChannel(t *testing.T) {
	s := newPrepWithProcess()
	assert.NotPanics(t, func() { s.Poke() })
}
