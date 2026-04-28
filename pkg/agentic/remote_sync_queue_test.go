// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestRemotesyncqueue_DrainReady_Good_EnqueueAndDrain(t *testing.T) {
	store := &remoteSyncQueueTestStore{}
	store.write([]syncQueuedPush{{
		AgentID:    "charon",
		QueuedAt:   time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC),
		Dispatches: []map[string]any{{"workspace": "core/go-io/task-5", "status": "completed"}},
	}})

	pushed := make(chan syncQueuedPush, 1)
	controller := remoteSyncQueueController{
		clock:       newRemoteSyncQueueStubClock(time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)),
		maxAttempts: 3,
		read:        store.read,
		write:       store.write,
		push: func(_ context.Context, queued syncQueuedPush) error {
			pushed <- queued
			return nil
		},
	}

	synced := controller.drainReady(context.Background())
	core.AssertEqual(t, 1, synced)
	core.AssertEmpty(t, store.snapshot())

	select {
	case queued := <-pushed:
		core.AssertEqual(t, "charon", queued.AgentID)
		core.AssertLen(t, queued.Dispatches, 1)
		core.AssertEqual(t, "core/go-io/task-5", queued.Dispatches[0]["workspace"])
	case <-time.After(time.Second):
		t.Fatal("expected queued push to be drained")
	}
}

func TestRemotesyncqueue_Run_Bad_RetriesWithBackoff(t *testing.T) {
	start := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	store := &remoteSyncQueueTestStore{}
	store.write([]syncQueuedPush{{
		AgentID:    "charon",
		QueuedAt:   start,
		Dispatches: []map[string]any{{"workspace": "core/go-io/task-5", "status": "completed"}},
	}})

	clock := newRemoteSyncQueueStubClock(start)
	attempts := make(chan int, 3)
	var calls int

	controller := remoteSyncQueueController{
		clock:       clock,
		idlePoll:    time.Hour,
		maxAttempts: 10,
		read:        store.read,
		write:       store.write,
		push: func(_ context.Context, _ syncQueuedPush) error {
			calls++
			attempts <- calls
			if calls < 3 {
				return errors.New("offline")
			}
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		controller.run(ctx)
		close(done)
	}()

	core.AssertEqual(t, 1, remoteSyncQueueReceiveAttempt(t, attempts))
	requireEventually(t, func() bool {
		queued := store.snapshot()
		return len(queued) == 1 &&
			queued[0].Attempts == 1 &&
			queued[0].NextAttempt.Equal(start.Add(time.Second))
	}, time.Second, 10*time.Millisecond)
	requireEventually(t, func() bool { return clock.WaiterCount() == 1 }, time.Second, 10*time.Millisecond)

	clock.Advance(999 * time.Millisecond)
	remoteSyncQueueAssertNoAttempt(t, attempts)

	clock.Advance(time.Millisecond)
	core.AssertEqual(t, 2, remoteSyncQueueReceiveAttempt(t, attempts))
	requireEventually(t, func() bool {
		queued := store.snapshot()
		return len(queued) == 1 &&
			queued[0].Attempts == 2 &&
			queued[0].NextAttempt.Equal(start.Add(3*time.Second))
	}, time.Second, 10*time.Millisecond)
	requireEventually(t, func() bool { return clock.WaiterCount() == 1 }, time.Second, 10*time.Millisecond)

	clock.Advance(2 * time.Second)
	core.AssertEqual(t, 3, remoteSyncQueueReceiveAttempt(t, attempts))
	requireEventually(t, func() bool { return len(store.snapshot()) == 0 }, time.Second, 10*time.Millisecond)

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("remote sync drainer did not stop")
	}
}

func TestRemotesyncqueue_Run_Ugly_ContextCancellationStopsDrainer(t *testing.T) {
	clock := newRemoteSyncQueueStubClock(time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC))
	controller := remoteSyncQueueController{
		clock:    clock,
		idlePoll: time.Hour,
		read:     func() []syncQueuedPush { return nil },
		write:    func([]syncQueuedPush) {},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		controller.run(ctx)
		close(done)
	}()

	requireEventually(t, func() bool { return clock.WaiterCount() == 1 }, time.Second, 10*time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("remote sync drainer did not stop after cancellation")
	}
}

func TestRemotesyncqueue_DrainReady_Ugly_MaxAttemptsExhaustedLogsAndDrops(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	store := &remoteSyncQueueTestStore{}
	store.write([]syncQueuedPush{{
		AgentID:    "charon",
		QueuedAt:   time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC),
		Attempts:   1,
		Dispatches: []map[string]any{{"workspace": "core/go-io/task-5", "status": "completed"}},
	}})

	var warnings []string
	controller := remoteSyncQueueController{
		clock:       newRemoteSyncQueueStubClock(time.Date(2026, 4, 25, 10, 0, 1, 0, time.UTC)),
		maxAttempts: 2,
		read:        store.read,
		write:       store.write,
		push: func(_ context.Context, _ syncQueuedPush) error {
			return errors.New("offline")
		},
		onDrop: func(queued syncQueuedPush, err error, at time.Time) {
			recordSyncDrop(queued.AgentID, queued.FleetNodeID, queued.Dispatches, queued.Attempts, err, at)
		},
		warn: func(message string, _ ...any) {
			warnings = append(warnings, message)
		},
	}

	synced := controller.drainReady(context.Background())
	core.AssertEqual(t, 0, synced)
	core.AssertEmpty(t, store.snapshot())
	core.AssertLen(t, warnings, 1)
	core.AssertContains(t, warnings[0], "dropping sync queue entry")

	records := readSyncRecords()
	core.AssertLen(t, records, 1)
	core.AssertEqual(t, "drop", records[0].Direction)
	core.AssertEqual(t, 2, records[0].Attempts)
	core.AssertEqual(t, 1, records[0].ItemsCount)
	core.AssertContains(t, records[0].Reason, "offline")
}

func TestRemotesyncqueue_FileQueue_Good_PersistsAcrossRestart(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	writeSyncQueue([]syncQueuedPush{{
		AgentID:    "charon",
		QueuedAt:   time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC),
		Dispatches: []map[string]any{{"workspace": "core/go-io/task-5", "status": "completed"}},
	}})

	core.AssertTrue(t, fs.Exists(syncQueuePath()))

	restarted := readSyncQueue()
	core.AssertLen(t, restarted, 1)
	core.AssertEqual(t, "charon", restarted[0].AgentID)
	core.AssertLen(t, restarted[0].Dispatches, 1)
	core.AssertEqual(t, "core/go-io/task-5", restarted[0].Dispatches[0]["workspace"])
}

type remoteSyncQueueTestStore struct {
	mu     sync.Mutex
	queued []syncQueuedPush
}

func (s *remoteSyncQueueTestStore) read() []syncQueuedPush {
	s.mu.Lock()
	defer s.mu.Unlock()
	return remoteSyncQueueClone(s.queued)
}

func (s *remoteSyncQueueTestStore) write(queued []syncQueuedPush) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queued = remoteSyncQueueClone(queued)
}

func (s *remoteSyncQueueTestStore) snapshot() []syncQueuedPush {
	return s.read()
}

type remoteSyncQueueStubClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []remoteSyncQueueStubWaiter
}

type remoteSyncQueueStubWaiter struct {
	at time.Time
	ch chan time.Time
}

func newRemoteSyncQueueStubClock(now time.Time) *remoteSyncQueueStubClock {
	return &remoteSyncQueueStubClock{now: now}
}

func (c *remoteSyncQueueStubClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *remoteSyncQueueStubClock) After(delay time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan time.Time, 1)
	if delay <= 0 {
		ch <- c.now
		return ch
	}

	c.waiters = append(c.waiters, remoteSyncQueueStubWaiter{
		at: c.now.Add(delay),
		ch: ch,
	})
	return ch
}

func (c *remoteSyncQueueStubClock) Advance(delay time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(delay)
	now := c.now

	var due []remoteSyncQueueStubWaiter
	pending := make([]remoteSyncQueueStubWaiter, 0, len(c.waiters))
	for _, waiter := range c.waiters {
		if waiter.at.After(now) {
			pending = append(pending, waiter)
			continue
		}
		due = append(due, waiter)
	}
	c.waiters = pending
	c.mu.Unlock()

	for _, waiter := range due {
		waiter.ch <- now
	}
}

func (c *remoteSyncQueueStubClock) WaiterCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.waiters)
}

func remoteSyncQueueClone(queued []syncQueuedPush) []syncQueuedPush {
	payload := core.JSONMarshalString(queued)
	var clone []syncQueuedPush
	_ = core.JSONUnmarshalString(payload, &clone)
	return clone
}

func remoteSyncQueueReceiveAttempt(t *testing.T, attempts <-chan int) int {
	t.Helper()

	select {
	case attempt := <-attempts:
		return attempt
	case <-time.After(time.Second):
		t.Fatal("expected remote sync attempt")
		return 0
	}
}

func remoteSyncQueueAssertNoAttempt(t *testing.T, attempts <-chan int) {
	t.Helper()

	select {
	case attempt := <-attempts:
		t.Fatalf("unexpected remote sync attempt %d", attempt)
	case <-time.After(100 * time.Millisecond):
	}
}
