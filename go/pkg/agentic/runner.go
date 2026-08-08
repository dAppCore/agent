// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

// subsystem := agentic.NewPrep()
// subsystem.StartRunner()
func (s *PrepSubsystem) StartRunner() {
	s.runRunnerAction("runner.start")
}

// subsystem := agentic.NewPrep()
// subsystem.Poke()
func (s *PrepSubsystem) Poke() {
	s.runRunnerAction("runner.poke")
}

func (s *PrepSubsystem) runRunnerAction(name string) {
	if s == nil || s.ServiceRuntime == nil {
		return
	}

	action := s.Core().Action(name)
	if action == nil || !action.Exists() {
		return
	}

	// Reported: the caller has no other signal that this did not run.
	if r := action.Run(context.Background(), core.NewOptions()); !r.OK {
		core.Warn("agentic: runner action failed", "reason", r.Value)
	}
}
