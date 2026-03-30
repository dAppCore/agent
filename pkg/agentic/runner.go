// SPDX-License-Identifier: EUPL-1.2

package agentic

// StartRunner preserves the legacy PrepSubsystem call after queue ownership moved to pkg/runner.Service.
//
//	subsystem := agentic.NewPrep()
//	subsystem.StartRunner()
//
// The runner service registers as core.WithService(runner.Register) and
// manages its own background loop, frozen state, and concurrency checks.
func (s *PrepSubsystem) StartRunner() {}

// Poke preserves the legacy queue signal after queue ownership moved to pkg/runner.Service.
//
//	subsystem := agentic.NewPrep()
//	subsystem.Poke()
//
// Runner catches AgentCompleted via HandleIPCEvents and pokes itself.
func (s *PrepSubsystem) Poke() {}
