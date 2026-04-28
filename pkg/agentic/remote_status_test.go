// SPDX-License-Identifier: EUPL-1.2

// Tests for remote_status.go — statusRemote.

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- statusRemote ---

func TestRemotestatus_StatusRemote_Good(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount {
		case 1:
			core.Print(w, "data: {\"result\":{}}\n")
		case 2:
			w.WriteHeader(200)
		case 3:
			result := map[string]any{
				"result": map[string]any{
					"content": []map[string]any{
						{"text": `{"total":5,"running":2,"completed":3,"failed":0}`},
					},
				},
			}
			core.Print(w, "data: %s\n", core.JSONMarshalString(result))
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{
		Host: srv.Listener.Addr().String(),
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 5, out.Stats.Total)
	core.AssertEqual(t, 2, out.Stats.Running)
}

func TestRemotestatus_StatusRemote_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}

	// Missing host
	_, _, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "host is required")

	// Unreachable
	_, out, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: "127.0.0.1:1"})
	core.AssertNoError(t, err)
	core.AssertContains(t, out.Error, "unreachable")

	// Call fails after init
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount {
		case 1:
			core.Print(w, "data: {\"result\":{}}\n")
		case 2:
			w.WriteHeader(200)
		case 3:
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(srv.Close)

	_, out2, _ := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: srv.Listener.Addr().String()})
	core.AssertContains(t, out2.Error, "call failed")
}

func TestRemotestatus_StatusRemote_Ugly(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount {
		case 1:
			core.Print(w, "data: {\"result\":{}}\n")
		case 2:
			w.WriteHeader(200)
		case 3:
			// JSON-RPC error
			result := map[string]any{"error": map[string]any{"code": -32000, "message": "internal error"}}
			core.Print(w, "data: %s\n", core.JSONMarshalString(result))
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, _ := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: srv.Listener.Addr().String()})
	core.AssertFalse(t, out.Success)
	core.AssertContains(t, out.Error, "internal error")

	// Unparseable response
	callCount2 := 0
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount2++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount2 {
		case 1:
			core.Print(w, "data: {\"result\":{}}\n")
		case 2:
			w.WriteHeader(200)
		case 3:
			core.Print(w, "data: not-json\n")
		}
	}))
	t.Cleanup(srv2.Close)

	_, out2, _ := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: srv2.Listener.Addr().String()})
	core.AssertFalse(t, out2.Success)
	core.AssertContains(t, out2.Error, "failed to parse")
}
