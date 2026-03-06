package lifecycle

import (
	"context"
	"sync"
	"time"
)

// EventType identifies the kind of lifecycle event.
type EventType string

const (
	// EventTaskDispatched is emitted when a task is successfully routed and claimed.
	EventTaskDispatched EventType = "task_dispatched"
	// EventTaskClaimed is emitted when a task claim succeeds via the API client.
	EventTaskClaimed EventType = "task_claimed"
	// EventDispatchFailedNoAgent is emitted when no eligible agent is available.
	EventDispatchFailedNoAgent EventType = "dispatch_failed_no_agent"
	// EventDispatchFailedQuota is emitted when an agent's quota is exceeded.
	EventDispatchFailedQuota EventType = "dispatch_failed_quota"
	// EventTaskDeadLettered is emitted when a task exceeds its retry limit.
	EventTaskDeadLettered EventType = "task_dead_lettered"
	// EventQuotaWarning is emitted when an agent reaches 80%+ quota usage.
	EventQuotaWarning EventType = "quota_warning"
	// EventQuotaExceeded is emitted when an agent exceeds their quota.
	EventQuotaExceeded EventType = "quota_exceeded"
	// EventUsageRecorded is emitted when usage is recorded for an agent.
	EventUsageRecorded EventType = "usage_recorded"
)

// Event represents a lifecycle event in the agentic system.
type Event struct {
	// Type identifies what happened.
	Type EventType `json:"type"`
	// TaskID is the task involved, if any.
	TaskID string `json:"task_id,omitempty"`
	// AgentID is the agent involved, if any.
	AgentID string `json:"agent_id,omitempty"`
	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
	// Payload carries additional event-specific data.
	Payload any `json:"payload,omitempty"`
}

// EventEmitter is the interface for publishing lifecycle events.
type EventEmitter interface {
	// Emit publishes an event. Implementations should be non-blocking.
	Emit(ctx context.Context, event Event) error
}

// ChannelEmitter is an in-process EventEmitter backed by a buffered channel.
// Events are dropped (not blocked) when the buffer is full.
type ChannelEmitter struct {
	ch chan Event
}

// NewChannelEmitter creates a ChannelEmitter with the given buffer size.
func NewChannelEmitter(bufSize int) *ChannelEmitter {
	if bufSize < 1 {
		bufSize = 64
	}
	return &ChannelEmitter{ch: make(chan Event, bufSize)}
}

// Emit sends an event to the channel. If the buffer is full, the event is
// dropped silently to avoid blocking the dispatch path.
func (e *ChannelEmitter) Emit(_ context.Context, event Event) error {
	select {
	case e.ch <- event:
	default:
		// Buffer full — drop the event rather than blocking.
	}
	return nil
}

// Events returns the underlying channel for consumers to read from.
func (e *ChannelEmitter) Events() <-chan Event {
	return e.ch
}

// Close closes the underlying channel, signalling consumers to stop reading.
func (e *ChannelEmitter) Close() {
	close(e.ch)
}

// MultiEmitter fans out events to multiple emitters. Emission continues even
// if one emitter fails — errors are collected but not returned.
type MultiEmitter struct {
	mu       sync.RWMutex
	emitters []EventEmitter
}

// NewMultiEmitter creates a MultiEmitter that fans out to the given emitters.
func NewMultiEmitter(emitters ...EventEmitter) *MultiEmitter {
	return &MultiEmitter{emitters: emitters}
}

// Emit sends the event to all registered emitters. Non-blocking: each emitter
// is called in sequence but ChannelEmitter.Emit is itself non-blocking.
func (m *MultiEmitter) Emit(ctx context.Context, event Event) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, em := range m.emitters {
		_ = em.Emit(ctx, event)
	}
	return nil
}

// Add appends an emitter to the fan-out list.
func (m *MultiEmitter) Add(emitter EventEmitter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.emitters = append(m.emitters, emitter)
}
