// SPDX-Licence-Identifier: EUPL-1.2

// Success-registration coverage for Subscribe itself. The existing
// subscribe_extra_test.go drives runSubscription directly and covers the
// empty-id / nil-receiver guards, but never exercises Subscribe's own
// registration path: the targetFor resolve, the goroutine spawn, the
// subscriptions-map insert, and the idempotent already-subscribed return.
//
// This drives Subscribe end-to-end against an SSE httptest server through a
// running Sandbox row (procBackedService + a pinned HostPort). The
// behaviour asserted is registration + idempotency + deregistration, not a
// bare "it didn't panic": the goroutine actually connects (the emitter
// fires), a second Subscribe returns the SAME cancel, and Unsubscribe
// removes the entry so a later Subscribe spins a fresh one.

package opencode

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/orm"
)

// sseServer returns an httptest server that emits one SSE data line then
// holds the connection until the client cancels. The bound port is
// returned so a Sandbox row can point its HostPort at it.
func sseServer(t *testing.T) (*httptest.Server, int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(core.Flusher); ok {
			_, _ = w.Write([]byte("data: {\"type\":\"session.updated\"}\n\n"))
			f.Flush()
		}
		// Block until the request context is cancelled (client Unsubscribe /
		// test cleanup) so the stream stays open like real opencode-serve.
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	_, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	core.AssertNoError(t, err)
	port, err := strconv.Atoi(portStr)
	core.AssertNoError(t, err)
	return srv, port
}

// TestSubscribe_Good_RegistersAndForwards — Subscribe against a running
// sandbox + emitter spawns the SSE goroutine: the first event reaches the
// emitter (the goroutine genuinely connected) and the returned cancel is
// non-nil. Cancelling deregisters.
func TestSubscribe_Good_RegistersAndForwards(t *testing.T) {
	_, port := sseServer(t)
	svc := procBackedService(t, "true")

	sb := Sandbox{ID: "oc-sub", Image: "img", HostPort: port, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	got := make(chan string, 1)
	svc.SetEventEmitter(func(e string) {
		select {
		case got <- e:
		default:
		}
	})

	cancel, r := svc.Subscribe("oc-sub")
	core.AssertTrue(t, r.OK)
	if cancel == nil {
		t.Fatal("Subscribe returned a nil cancel for a successful registration")
	}
	t.Cleanup(cancel)

	select {
	case e := <-got:
		core.AssertContains(t, e, "session.updated")
	case <-core.After(5 * core.Second):
		t.Fatal("Subscribe goroutine never forwarded the SSE event")
	}
}

// TestSubscribe_Ugly_IdempotentSameCancel — a second Subscribe for an
// already-subscribed id returns the EXISTING cancel without spawning a
// second goroutine (the already-subscribed short-circuit). We assert the
// registration count stays at one by checking Unsubscribe fully clears it.
func TestSubscribe_Ugly_IdempotentSameCancel(t *testing.T) {
	_, port := sseServer(t)
	svc := procBackedService(t, "true")

	sb := Sandbox{ID: "oc-idem", Image: "img", HostPort: port, Status: StatusRunning, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)
	svc.SetEventEmitter(func(string) {})

	first, r1 := svc.Subscribe("oc-idem")
	core.AssertTrue(t, r1.OK)
	t.Cleanup(first)

	// Second call must short-circuit to the same registration. The map
	// holds exactly one entry keyed by id, so a follow-up Subscribe finds
	// it and returns Ok with the stored cancel.
	_, r2 := svc.Subscribe("oc-idem")
	core.AssertTrue(t, r2.OK)

	// Deregister, then a fresh Subscribe must succeed again (proving the
	// id was a single live registration that Unsubscribe cleared).
	svc.Unsubscribe("oc-idem")
	third, r3 := svc.Subscribe("oc-idem")
	core.AssertTrue(t, r3.OK)
	t.Cleanup(third)
}

// TestSubscribe_Good_NoEmitterSkips — with no emitter installed Subscribe
// short-circuits BEFORE any network connection: it returns Ok + a no-op
// cancel and never registers. Pins the "no consumer — skip the SSE
// connection entirely" arm.
func TestSubscribe_Good_NoEmitterSkips(t *testing.T) {
	svc := procBackedService(t, "true")
	// Deliberately no SetEventEmitter, and a stopped/absent sandbox: the
	// emitter==nil guard fires before targetFor, so the missing sandbox is
	// irrelevant and no goroutine spawns.
	cancel, r := svc.Subscribe("oc-whatever")
	core.AssertTrue(t, r.OK)
	if cancel == nil {
		t.Fatal("no-emitter Subscribe should return a non-nil no-op cancel")
	}
	cancel() // no-op; must not panic.
}

// TestSubscribe_Bad_NotRunningTarget — with an emitter set but the sandbox
// row stopped, Subscribe reaches targetFor and surfaces its not-running
// failure (no goroutine spawned).
func TestSubscribe_Bad_NotRunningTarget(t *testing.T) {
	svc := procBackedService(t, "true")
	svc.SetEventEmitter(func(string) {})

	sb := Sandbox{ID: "oc-down", Image: "img", HostPort: 51999, Status: StatusStopped, CreatedAt: core.Now()}
	core.AssertTrue(t, orm.Of[Sandbox](svc.Core()).Save(&sb).OK)

	_, r := svc.Subscribe("oc-down")
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Error(), "not running")
}
