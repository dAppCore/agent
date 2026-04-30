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

func TestRunner_PrepSubsystem_StartRunner_Good(t *testing.T) {
	// StartRunner is now a no-op — queue drain is owned by pkg/runner.Service.
	// Verify it does not panic and does not set pokeCh.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	core.AssertNil(t, s.pokeCh)

	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s.pokeCh, "no-op StartRunner should not initialise pokeCh")
}

func TestRunner_PrepSubsystem_StartRunner_Bad(t *testing.T) {
	// StartRunner is now a no-op — frozen state and pokeCh are owned by pkg/runner.Service.
	// Verify the no-op does not panic and does not modify state.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "")

	s := NewPrep()
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s.pokeCh, "no-op StartRunner should not create pokeCh")
}

func TestRunner_PrepSubsystem_StartRunner_Ugly(t *testing.T) {
	// StartRunner is now a no-op — calling it multiple times must not panic.
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_DISPATCH", "1")

	s := NewPrep()

	// Call twice — both are no-ops, must not panic
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNotPanics(t, func() { s.StartRunner() })
	core.AssertNil(t, s.pokeCh, "no-op StartRunner should not create pokeCh")
}

func TestRunner_PrepSubsystem_Poke_Good(t *testing.T) {
	// Poke is now a no-op — queue poke is owned by pkg/runner.Service.
	// Verify it does not send to the channel and does not panic.
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)

	core.AssertNotPanics(t, func() { s.Poke() })
	core.AssertLen(t, s.pokeCh, 0, "no-op poke should not enqueue a signal")
}

func TestRunner_PrepSubsystem_Poke_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), pokeCh: nil}
	// Must not panic when pokeCh is nil
	core.AssertNotPanics(t, func() {
		s.Poke()
	})
}

func TestRunner_PrepSubsystem_Poke_Ugly(t *testing.T) {
	// Poke on a closed channel — the select with default protects against panic
	// but closing + sending would panic. However, Poke uses non-blocking send,
	// so we test that pokeCh=nil is safe (already tested), and that
	// double-filling is safe (already tested). Here we test rapid multi-poke.
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.pokeCh = make(chan struct{}, 1)

	// Rapid-fire pokes — should all be safe
	for i := 0; i < 100; i++ {
		core.AssertNotPanics(t, func() { s.Poke() })
	}
	// Channel should have at most 1 signal
	core.AssertLessOrEqual(t, len(s.pokeCh), 1)
}
