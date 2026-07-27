// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestService_Subscribe — subscribe rejects empty ids + nil receivers;
// unsubscribe is a safe no-op for unknown ids.
func TestService_Subscribe(t *testing.T) {
	svc := newTestService(t)

	_, r := svc.Subscribe("")
	core.AssertFalse(t, r.OK)

	cancel, _ := svc.Subscribe("oc-1")
	cancel()
	svc.Unsubscribe("oc-1")
	svc.Unsubscribe("nope")

	var nilSvc *Service
	_, rn := nilSvc.Subscribe("x")
	core.AssertFalse(t, rn.OK)
	nilSvc.Unsubscribe("x")
}

// TestSubscribe_runSubscription_ForwardsThenCancels — drives the
// runSubscription goroutine body directly against an SSE server. The loop
// calls streamEvents (one event forwarded to the emitter), the server
// closes the stream so streamEvents returns nil, the loop enters its
// clean-close 500ms backoff select, and the cancel we fire makes it exit
// via <-ctx.Done(). Covers the reconnect-loop body + its clean-close path.
func TestSubscribe_runSubscription_ForwardsThenCancels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		f, _ := w.(core.Flusher)
		_, _ = w.Write([]byte("data: {\"type\":\"x\"}\n\n"))
		if f != nil {
			f.Flush()
		}
		// Returning closes the connection → streamEvents sees EOF and
		// returns nil, exercising the loop's clean-close branch.
	}))
	defer server.Close()

	svc := newTestService(t)
	got := make(chan string, 1)
	svc.SetEventEmitter(func(e string) {
		select {
		case got <- e:
		default:
		}
	})

	ctx, cancel := core.WithCancel(core.Background())
	done := make(chan struct{})
	go func() {
		svc.runSubscription(ctx, "oc-1", server.URL, "")
		close(done)
	}()

	// First event must reach the emitter.
	select {
	case e := <-got:
		core.AssertContains(t, e, "type")
	case <-core.After(5 * core.Second):
		t.Fatal("runSubscription never forwarded the event")
	}

	// Cancelling must make the goroutine return promptly.
	cancel()
	select {
	case <-done:
	case <-core.After(5 * core.Second):
		t.Fatal("runSubscription did not exit after cancel")
	}
}

// TestSubscribe_runSubscription_ErrorBackoffThenCancel — points
// runSubscription at a dead target so streamEvents errors immediately;
// the loop enters its error-backoff select, and the cancel makes it exit
// via <-ctx.Done() rather than waiting out the backoff. Covers the
// error-reconnect branch of the loop.
func TestSubscribe_runSubscription_ErrorBackoffThenCancel(t *testing.T) {
	svc := newTestService(t)
	svc.SetEventEmitter(func(string) {})

	ctx, cancel := core.WithCancel(core.Background())
	done := make(chan struct{})
	go func() {
		// 127.0.0.1:1 refuses immediately → streamEvents returns an error.
		svc.runSubscription(ctx, "oc-1", "http://127.0.0.1:1", "")
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-core.After(5 * core.Second):
		t.Fatal("runSubscription did not exit after cancel on the error path")
	}
}
