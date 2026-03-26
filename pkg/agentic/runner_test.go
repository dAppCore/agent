// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// StartRunner and Poke are no-ops — queue drain is owned by pkg/runner.Service.

func TestRunner_StartRunner_Good(t *testing.T) {
	s := newPrepWithProcess()
	assert.NotPanics(t, func() { s.StartRunner() })
}

func TestRunner_StartRunner_Bad_AlreadyRunning(t *testing.T) {
	s := newPrepWithProcess()
	s.StartRunner()
	assert.NotPanics(t, func() { s.StartRunner() })
}

func TestRunner_Poke_Ugly_NilChannel(t *testing.T) {
	s := newPrepWithProcess()
	assert.NotPanics(t, func() { s.Poke() })
}
