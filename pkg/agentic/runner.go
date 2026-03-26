// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"time"

	core "dappco.re/go/core"
)

// StartRunner begins the background queue runner.
// Queue is frozen by default — use agentic_dispatch_start to unfreeze,
// or set CORE_AGENT_DISPATCH=1 to auto-start.
//
//	prep.StartRunner()
func (s *PrepSubsystem) StartRunner() {
	s.pokeCh = make(chan struct{}, 1)

	// Frozen by default — explicit start required
	if core.Env("CORE_AGENT_DISPATCH") == "1" {
		s.frozen = false
		core.Print(nil, "dispatch: auto-start enabled (CORE_AGENT_DISPATCH=1)")
	} else {
		s.frozen = true
	}

	go s.runLoop()
}

func (s *PrepSubsystem) runLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.drainQueue()
		case <-s.pokeCh:
			s.drainQueue()
		}
	}
}

// Poke signals the runner to check the queue immediately.
// Non-blocking — if a poke is already pending, this is a no-op.
//
//	s.Poke() // after agent completion
func (s *PrepSubsystem) Poke() {
	if s.pokeCh == nil {
		return
	}
	select {
	case s.pokeCh <- struct{}{}:
	default:
	}
}
