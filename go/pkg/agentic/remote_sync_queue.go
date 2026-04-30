// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"strconv"
	"sync"
	"time"

	core "dappco.re/go"
)

const (
	remoteSyncQueueDefaultMaxAttempts = 100
	remoteSyncQueueMinBackoff         = time.Second
	remoteSyncQueueMaxBackoff         = 30 * time.Second
)

var remoteSyncQueueMu sync.Mutex

type remoteSyncClock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}

type remoteSyncRealClock struct{}

// clock := remoteSyncRealClock{}
// now := clock.Now()
func (remoteSyncRealClock) Now() time.Time {
	return time.Now()
}

// clock := remoteSyncRealClock{}
// <-clock.After(time.Second)
func (remoteSyncRealClock) After(delay time.Duration) <-chan time.Time {
	return time.After(delay)
}

type remoteSyncQueueController struct {
	mu          *sync.Mutex
	clock       remoteSyncClock
	idlePoll    time.Duration
	maxAttempts int
	read        func() []syncQueuedPush
	write       func([]syncQueuedPush)
	push        func(context.Context, syncQueuedPush) error
	onSuccess   func(syncQueuedPush, time.Time)
	onDrop      func(syncQueuedPush, error, time.Time)
	warn        func(string, ...any)
}

// delay := remoteSyncQueueBackoff(3) // 4s
func remoteSyncQueueBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return 0
	}

	shift := attempts - 1
	if shift > 5 {
		shift = 5
	}

	delay := remoteSyncQueueMinBackoff * time.Duration(1<<shift)
	if delay > remoteSyncQueueMaxBackoff {
		return remoteSyncQueueMaxBackoff
	}
	return delay
}

// attempts := remoteSyncQueueMaxAttempts() // 100
func remoteSyncQueueMaxAttempts() int {
	for _, key := range []string{"CORE_AGENT_SYNC_MAX_ATTEMPTS", "CORE_SYNC_MAX_ATTEMPTS"} {
		value := core.Trim(core.Env(key))
		if value == "" {
			continue
		}
		attempts, err := strconv.Atoi(value)
		if err == nil && attempts > 0 {
			return attempts
		}
	}
	return remoteSyncQueueDefaultMaxAttempts
}

// controller := s.remoteSyncQueueController(remoteSyncRealClock{}, time.Minute)
// controller.run(ctx)
func (s *PrepSubsystem) remoteSyncQueueController(clock remoteSyncClock, idlePoll time.Duration) remoteSyncQueueController {
	if clock == nil {
		clock = remoteSyncRealClock{}
	}

	return remoteSyncQueueController{
		mu:          &remoteSyncQueueMu,
		clock:       clock,
		idlePoll:    idlePoll,
		maxAttempts: remoteSyncQueueMaxAttempts(),
		read: func() []syncQueuedPush {
			if s == nil {
				return nil
			}
			return append([]syncQueuedPush(nil), s.readSyncQueue()...)
		},
		write: func(queued []syncQueuedPush) {
			if s == nil {
				return
			}
			s.writeSyncQueue(queued)
		},
		push: func(ctx context.Context, queued syncQueuedPush) error {
			if s == nil {
				return core.E("agentic.remoteSyncQueue", "nil subsystem", nil)
			}
			return postSyncPush(s, ctx, queued.AgentID, queued.Dispatches, s.syncToken())
		},
		onSuccess: func(queued syncQueuedPush, at time.Time) {
			markDispatchesSynced(queued.Dispatches)
			recordSyncPush(at)
			recordSyncHistory("push", queued.AgentID, queued.FleetNodeID, len(core.JSONMarshalString(map[string]any{
				"agent_id":   queued.AgentID,
				"dispatches": queued.Dispatches,
			})), len(queued.Dispatches), at)
		},
		onDrop: func(queued syncQueuedPush, err error, at time.Time) {
			recordSyncDrop(queued.AgentID, queued.FleetNodeID, queued.Dispatches, queued.Attempts, err, at)
		},
		warn: core.Warn,
	}
}

// queued := syncQueuedPush{AgentID: "charon", Dispatches: []map[string]any{{"workspace": "core/go-io/task-5"}}}
// s.enqueueSyncPush(queued)
func (s *PrepSubsystem) enqueueSyncPush(queued syncQueuedPush) {
	remoteSyncQueueMu.Lock()
	defer remoteSyncQueueMu.Unlock()

	current := s.readSyncQueue()
	current = append(current, queued)
	s.writeSyncQueue(current)
}

// synced := controller.drainReady(ctx)
func (c remoteSyncQueueController) drainReady(ctx context.Context) int {
	if ctx == nil {
		return 0
	}

	c.lock()
	defer c.unlock()

	synced := 0
	for {
		if ctx.Err() != nil {
			return synced
		}

		queued := c.readQueue()
		if len(queued) == 0 {
			return synced
		}

		head := queued[0]
		if len(head.Dispatches) == 0 {
			c.writeQueue(queued[1:])
			continue
		}

		now := c.now()
		if !head.NextAttempt.IsZero() && head.NextAttempt.After(now) {
			return synced
		}

		err := pushQueue(c, ctx, head)
		finishedAt := c.now()
		if err == nil {
			if c.onSuccess != nil {
				c.onSuccess(head, finishedAt)
			}
			c.writeQueue(queued[1:])
			synced += len(head.Dispatches)
			continue
		}

		head.Attempts++
		if head.Attempts >= c.maxAttemptLimit() {
			if c.warn != nil {
				c.warn("agentic: dropping sync queue entry after max attempts",
					"agent_id", head.AgentID,
					"attempts", head.Attempts,
					"reason", err)
			}
			if c.onDrop != nil {
				c.onDrop(head, err, finishedAt)
			}
			c.writeQueue(queued[1:])
			continue
		}

		head.NextAttempt = finishedAt.Add(remoteSyncQueueBackoff(head.Attempts))
		queued[0] = head
		c.writeQueue(queued)
		return synced
	}
}

// controller := s.remoteSyncQueueController(remoteSyncRealClock{}, time.Minute)
// controller.run(ctx)
func (c remoteSyncQueueController) run(ctx context.Context) {
	if ctx == nil {
		return
	}

	for {
		if ctx.Err() != nil {
			return
		}

		c.drainReady(ctx)
		delay := c.nextWait()
		if delay <= 0 {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-c.after(delay):
		}
	}
}

func (c remoteSyncQueueController) nextWait() time.Duration {
	c.lock()
	defer c.unlock()

	queued := c.readQueue()
	if len(queued) == 0 {
		return c.idleDelay()
	}

	head := queued[0]
	if len(head.Dispatches) == 0 {
		return 0
	}

	now := c.now()
	if head.NextAttempt.IsZero() || !head.NextAttempt.After(now) {
		return 0
	}
	return head.NextAttempt.Sub(now)
}

func (c remoteSyncQueueController) maxAttemptLimit() int {
	if c.maxAttempts > 0 {
		return c.maxAttempts
	}
	return remoteSyncQueueDefaultMaxAttempts
}

func (c remoteSyncQueueController) idleDelay() time.Duration {
	if c.idlePoll > 0 {
		return c.idlePoll
	}
	return remoteSyncQueueMinBackoff
}

func (c remoteSyncQueueController) now() time.Time {
	if c.clock == nil {
		return time.Now()
	}
	return c.clock.Now()
}

func (c remoteSyncQueueController) after(delay time.Duration) <-chan time.Time {
	if c.clock == nil {
		return time.After(delay)
	}
	return c.clock.After(delay)
}

func (c remoteSyncQueueController) readQueue() []syncQueuedPush {
	if c.read == nil {
		return nil
	}
	return c.read()
}

func (c remoteSyncQueueController) writeQueue(queued []syncQueuedPush) {
	if c.write == nil {
		return
	}
	c.write(queued)
}

var pushQueue = func(c remoteSyncQueueController, ctx context.Context, queued syncQueuedPush) error {
	if c.push == nil {
		return nil
	}
	return c.push(ctx, queued)
}

func (c remoteSyncQueueController) lock() {
	if c.mu != nil {
		c.mu.Lock()
	}
}

func (c remoteSyncQueueController) unlock() {
	if c.mu != nil {
		c.mu.Unlock()
	}
}
