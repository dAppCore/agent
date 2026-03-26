// SPDX-License-Identifier: EUPL-1.2

package agentic

// StartRunner is a no-op — queue drain is now owned by pkg/runner.Service.
// Kept for backward compatibility with OnStartup call.
//
// The runner service registers as core.WithService(runner.Register) and
// manages its own background loop, frozen state, and concurrency checks.
func (s *PrepSubsystem) StartRunner() {}

// Poke is a no-op — queue poke is now owned by pkg/runner.Service.
// Runner catches AgentCompleted via HandleIPCEvents and pokes itself.
func (s *PrepSubsystem) Poke() {}
