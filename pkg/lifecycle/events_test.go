package lifecycle

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelEmitter_EmitAndReceive(t *testing.T) {
	em := NewChannelEmitter(10)
	ctx := context.Background()

	event := Event{
		Type:      EventTaskDispatched,
		TaskID:    "task-1",
		AgentID:   "agent-1",
		Timestamp: time.Now().UTC(),
		Payload:   "test payload",
	}

	err := em.Emit(ctx, event)
	require.NoError(t, err)

	select {
	case got := <-em.Events():
		assert.Equal(t, EventTaskDispatched, got.Type)
		assert.Equal(t, "task-1", got.TaskID)
		assert.Equal(t, "agent-1", got.AgentID)
		assert.Equal(t, "test payload", got.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestChannelEmitter_BufferOverflowDrops(t *testing.T) {
	em := NewChannelEmitter(2)
	ctx := context.Background()

	// Fill the buffer.
	require.NoError(t, em.Emit(ctx, Event{Type: EventTaskDispatched, TaskID: "1"}))
	require.NoError(t, em.Emit(ctx, Event{Type: EventTaskDispatched, TaskID: "2"}))

	// Third event should be dropped, not block.
	err := em.Emit(ctx, Event{Type: EventTaskDispatched, TaskID: "3"})
	require.NoError(t, err)

	// Only 2 events in the channel.
	assert.Len(t, em.ch, 2)
}

func TestChannelEmitter_DefaultBufferSize(t *testing.T) {
	em := NewChannelEmitter(0)
	assert.Equal(t, 64, cap(em.ch))
}

func TestMultiEmitter_FanOut(t *testing.T) {
	em1 := NewChannelEmitter(10)
	em2 := NewChannelEmitter(10)
	multi := NewMultiEmitter(em1, em2)
	ctx := context.Background()

	event := Event{
		Type:    EventQuotaWarning,
		AgentID: "agent-x",
	}
	err := multi.Emit(ctx, event)
	require.NoError(t, err)

	// Both emitters should have received the event.
	select {
	case got := <-em1.Events():
		assert.Equal(t, EventQuotaWarning, got.Type)
	case <-time.After(time.Second):
		t.Fatal("em1: timed out")
	}

	select {
	case got := <-em2.Events():
		assert.Equal(t, EventQuotaWarning, got.Type)
	case <-time.After(time.Second):
		t.Fatal("em2: timed out")
	}
}

func TestMultiEmitter_Add(t *testing.T) {
	em1 := NewChannelEmitter(10)
	multi := NewMultiEmitter(em1)
	ctx := context.Background()

	em2 := NewChannelEmitter(10)
	multi.Add(em2)

	err := multi.Emit(ctx, Event{Type: EventUsageRecorded})
	require.NoError(t, err)

	assert.Len(t, em1.ch, 1)
	assert.Len(t, em2.ch, 1)
}

func TestMultiEmitter_ContinuesOnFailure(t *testing.T) {
	failing := &failingEmitter{}
	good := NewChannelEmitter(10)
	multi := NewMultiEmitter(failing, good)
	ctx := context.Background()

	err := multi.Emit(ctx, Event{Type: EventTaskClaimed})
	require.NoError(t, err) // MultiEmitter swallows errors.

	// The good emitter should still have received the event.
	assert.Len(t, good.ch, 1)
}

func TestChannelEmitter_ConcurrentEmit(t *testing.T) {
	em := NewChannelEmitter(100)
	ctx := context.Background()

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			_ = em.Emit(ctx, Event{Type: EventTaskDispatched})
		})
	}
	wg.Wait()

	assert.Equal(t, 50, len(em.ch))
}

func TestEventTypes_AllDefined(t *testing.T) {
	types := []EventType{
		EventTaskDispatched,
		EventTaskClaimed,
		EventDispatchFailedNoAgent,
		EventDispatchFailedQuota,
		EventTaskDeadLettered,
		EventQuotaWarning,
		EventQuotaExceeded,
		EventUsageRecorded,
	}
	for _, et := range types {
		assert.NotEmpty(t, string(et))
	}
}

// failingEmitter always returns an error.
type failingEmitter struct{}

func (f *failingEmitter) Emit(_ context.Context, _ Event) error {
	return &APIError{Code: 500, Message: "emitter failed"}
}
