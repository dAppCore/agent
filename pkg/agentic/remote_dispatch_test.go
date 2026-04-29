// SPDX-License-Identifier: EUPL-1.2

// Tests for remote.go — dispatchRemote, resolveHost, remoteToken.

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- dispatchRemote ---

func TestRemote_DispatchRemote_Good_Case(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "test-session")
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
						{"text": `{"success":true,"agent":"codex","repo":"go-io","workspace_dir":"/ws/go-io","pid":12345}`},
					},
				},
			}
			core.Print(w, "data: %s\n", core.JSONMarshalString(result))
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Host: srv.Listener.Addr().String(), Repo: "go-io", Task: "Fix tests",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "go-io", out.Repo)
}

func TestRemote_DispatchRemote_Bad_Case(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}

	// Missing host
	_, _, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{Repo: "go-io", Task: "do"})
	core.AssertContains(t, err.Error(), "host is required")

	// Missing repo
	_, _, err = s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{Host: "charon", Task: "do"})
	core.AssertContains(t, err.Error(), "repo is required")

	// Missing task
	_, _, err = s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{Host: "charon", Repo: "go-io"})
	core.AssertContains(t, err.Error(), "task is required")

	// Init fails (server error)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	_, _, err = s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Host: srv.Listener.Addr().String(), Repo: "go-io", Task: "test",
	})
	core.AssertContains(t, err.Error(), "MCP initialize failed")
}

func TestRemote_DispatchRemote_Ugly_Case(t *testing.T) {
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
			// JSON-RPC error response
			result := map[string]any{"error": map[string]any{"message": "tool not found"}}
			core.Print(w, "data: %s\n", core.JSONMarshalString(result))
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Host: srv.Listener.Addr().String(), Repo: "go-io", Task: "test",
		Agent: "claude:opus", Org: "core", Template: "coding", Persona: "eng",
		Variables: map[string]string{"key": "val"},
	})
	core.RequireNoError(t, err)
	core.AssertFalse(t, out.Success)
	core.AssertContains(t, out.Error, "tool not found")
}
