// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
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

	action.Run(context.Background(), core.NewOptions())
}
