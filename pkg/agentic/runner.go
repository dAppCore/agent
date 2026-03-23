// SPDX-License-Identifier: EUPL-1.2

package agentic

import "time"

// StartRunner begins the background queue runner.
// Ticks every 30s to drain queued tasks into available slots.
// Also responds to Poke() for immediate drain on completion events.
//
//	prep.StartRunner()
func (s *PrepSubsystem) StartRunner() {
	s.pokeCh = make(chan struct{}, 1)
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
