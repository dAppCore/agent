// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// StartRunner and Poke delegate to pkg/runner.Service when it is registered.

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

func TestRunner_StartRunner_Good_DelegatesToRunnerStartAction(t *testing.T) {
	coreApp := core.New(core.WithOption("name", "test"))
	called := false
	coreApp.Action("runner.start", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(coreApp, AgentOptions{})}
	s.StartRunner()

	assert.True(t, called)
}

func TestRunner_Poke_Good_DelegatesToRunnerPokeAction(t *testing.T) {
	coreApp := core.New(core.WithOption("name", "test"))
	called := false
	coreApp.Action("runner.poke", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(coreApp, AgentOptions{})}
	s.Poke()

	assert.True(t, called)
}
