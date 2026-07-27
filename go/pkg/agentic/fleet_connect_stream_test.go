// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

// TestFleetConnect_EventStream_Success_Good — a 200 SSE stream carrying one event
// is scanned, parsed and counted, and the runtime state flips to "connected".
// (The existing Connect tests only drive the 503 failure path, so the scan-loop
// success path was uncovered.)
func TestFleetConnect_EventStream_Success_Good(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/fleet/events", r.URL.Path)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: task.assigned\ndata: {\"repo\":\"core/go-io\"}\n\n"))
	}))
	defer server.Close()

	s := testPrepWithPlatformServer(t, server, "secret-token")
	config := fleetClientConfig{APIURL: server.URL, AgentID: "charon", AgentAPIKey: "secret-token"}
	result := s.connectFleetEventStream(context.Background(), config)

	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, 1, result.Value)
	core.AssertEqual(t, "connected", fleetRuntimeSnapshotValue().State)
}

// TestFleetConnect_PollFallbackHelpers_Good — the poll-fallback channel helpers:
// a nil cancel is a no-op, a live cancel is invoked and waited on, an open done
// is left intact, and a closed done clears the cancel + done handles.
func TestFleetConnect_PollFallbackHelpers_Good(t *testing.T) {
	fleetStopPollFallback(nil, nil) // nil cancel → no-op, no panic

	stopped := make(chan struct{})
	cancelled := false
	fleetStopPollFallback(func() { cancelled = true; close(stopped) }, stopped)
	core.AssertTrue(t, cancelled)

	open := make(chan struct{})
	openDone, openCancel := open, context.CancelFunc(func() {})
	fleetClearCompletedPollFallback(&openCancel, &openDone)
	core.AssertNotNil(t, openDone) // not yet done → left intact

	closed := make(chan struct{})
	close(closed)
	closedDone, closedCancel := closed, context.CancelFunc(func() {})
	fleetClearCompletedPollFallback(&closedCancel, &closedDone)
	core.AssertNil(t, closedDone) // completed → cleared
}

// TestFleetConnect_StartPollFallback_Good — the launcher spins a poll goroutine
// that hits the task endpoint; cancelling it closes the done channel.
func TestFleetConnect_StartPollFallback_Good(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	originalSleep := fleetSleep
	t.Cleanup(func() {
		fleetSleep = originalSleep
		resetFleetRuntimeState()
	})
	fleetSleep = func(ctx context.Context, _ time.Duration) bool { return ctx.Err() == nil }

	hit := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case hit <- struct{}{}:
		default:
		}
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	s := testPrepWithPlatformServer(t, server, "secret-token")
	config := fleetClientConfig{APIURL: server.URL, AgentID: "charon", AgentAPIKey: "secret-token"}
	cancel, done := s.startFleetPollFallback(context.Background(), config)
	<-hit
	cancel()
	<-done
}
