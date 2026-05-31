// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"sync"
	"testing"
)

// --- SetAuditSink ---

// TestSetAuditSink_InstallAndDispatch_Good — after installing an audit
// sink, dispatchAudit must forward events to it. Clearing with nil must
// restore the no-op behaviour.
func TestSetAuditSink_InstallAndDispatch_Good(t *testing.T) {
	var called bool
	var lastEvent, lastScope, lastOutcome, lastRequestID string

	SetAuditSink(func(event, scope, outcome, requestID string, meta map[string]any) {
		called = true
		lastEvent = event
		lastScope = scope
		lastOutcome = outcome
		lastRequestID = requestID
	})
	t.Cleanup(func() { SetAuditSink(nil) })

	dispatchAudit("opencode.test", "sandbox", "ok", "req-123", map[string]any{"key": "val"})

	if !called {
		t.Fatal("audit sink was not called")
	}
	if lastEvent != "opencode.test" {
		t.Errorf("event = %q; want opencode.test", lastEvent)
	}
	if lastScope != "sandbox" {
		t.Errorf("scope = %q; want sandbox", lastScope)
	}
	if lastOutcome != "ok" {
		t.Errorf("outcome = %q; want ok", lastOutcome)
	}
	if lastRequestID != "req-123" {
		t.Errorf("requestID = %q; want req-123", lastRequestID)
	}
}

// TestSetAuditSink_NilSinkNoOp_Good — when no sink is installed,
// dispatchAudit must not panic and must be a no-op.
func TestSetAuditSink_NilSinkNoOp_Good(t *testing.T) {
	// Ensure no sink is installed.
	SetAuditSink(nil)

	// dispatchAudit must not panic.
	dispatchAudit("opencode.test", "sandbox", "ok", "req-456", map[string]any{"a": "b"})
}

// TestSetAuditSink_ClearRestoresNoOp_Good — calling SetAuditSink(nil)
// after installing a sink must prevent further dispatches.
func TestSetAuditSink_ClearRestoresNoOp_Good(t *testing.T) {
	callCount := 0
	SetAuditSink(func(event, scope, outcome, requestID string, meta map[string]any) {
		callCount++
	})

	dispatchAudit("e1", "s1", "ok", "r1", nil)
	if callCount != 1 {
		t.Fatalf("first dispatch: callCount = %d; want 1", callCount)
	}

	// Clear.
	SetAuditSink(nil)
	dispatchAudit("e2", "s2", "ok", "r2", nil)
	if callCount != 1 {
		t.Fatalf("after clear: callCount = %d; want 1 (no new call)", callCount)
	}
}

// TestSetAuditSink_EmptyMeta_Good — a nil meta map must be forwarded
// safely.
func TestSetAuditSink_EmptyMeta_Good(t *testing.T) {
	var capturedMeta map[string]any
	SetAuditSink(func(event, scope, outcome, requestID string, meta map[string]any) {
		capturedMeta = meta
	})
	t.Cleanup(func() { SetAuditSink(nil) })

	dispatchAudit("e", "s", "ok", "r", nil)
	if capturedMeta != nil {
		t.Errorf("meta = %v; want nil", capturedMeta)
	}

	dispatchAudit("e2", "s2", "ok", "r2", map[string]any{})
	if capturedMeta == nil {
		t.Fatal("meta with empty map was not captured")
	}
	if len(capturedMeta) != 0 {
		t.Errorf("meta len = %d; want 0", len(capturedMeta))
	}
}

// TestAuditSink_Concurrent_Good — SetAuditSink and dispatchAudit must
// be safe for concurrent use.
func TestAuditSink_Concurrent_Good(t *testing.T) {
	SetAuditSink(func(event, scope, outcome, requestID string, meta map[string]any) {
		// no-op sink
	})
	t.Cleanup(func() { SetAuditSink(nil) })

	var wg sync.WaitGroup
	const goroutines = 20
	const iterations = 100

	// Concurrent dispatchers.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				dispatchAudit("e", "s", "ok", "r", nil)
			}
		}()
	}

	// Concurrent setter.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < iterations; j++ {
			SetAuditSink(nil)
			SetAuditSink(func(event, scope, outcome, requestID string, meta map[string]any) {})
		}
	}()

	wg.Wait()
}

// TestAuditSink_NilSinkConcurrent_Good — concurrent dispatchAudit calls
// against a nil sink must not race.
func TestAuditSink_NilSinkConcurrent_Good(t *testing.T) {
	SetAuditSink(nil)

	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				dispatchAudit("e", "s", "ok", "r", nil)
			}
		}()
	}

	wg.Wait()
}
