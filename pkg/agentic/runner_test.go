// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// StartRunner and Poke delegate to pkg/runner.Service when it is registered.

func TestRunner_StartRunner_Good(t *testing.T) {
	s := newPrepWithProcess()
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNotNil(t, s)
	core.AssertNotNil(t, s.ServiceRuntime)
}

func TestRunner_StartRunner_Bad_AlreadyRunning(t *testing.T) {
	s := newPrepWithProcess()
	s.StartRunner()
	core.AssertNotPanics(t, func() { s.StartRunner() })
}

func TestRunner_Poke_Ugly_NilChannel(t *testing.T) {
	s := newPrepWithProcess()
	core.AssertNotPanics(t, func() { s.Poke() })
	core.AssertNil(t, s.pokeCh)
	core.AssertNotNil(t, s.ServiceRuntime)
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

	core.AssertTrue(t, called)
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

	core.AssertTrue(t, called)
}
