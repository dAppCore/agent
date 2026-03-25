// SPDX-License-Identifier: EUPL-1.2

// Tests for remote_status.go — statusRemote.

package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- statusRemote ---

func TestRemoteStatus_StatusRemote_Good(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount {
		case 1:
			fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
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
			data, _ := json.Marshal(result)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{
		Host: srv.Listener.Addr().String(),
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 5, out.Stats.Total)
	assert.Equal(t, 2, out.Stats.Running)
}

func TestRemoteStatus_StatusRemote_Bad(t *testing.T) {
	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}

	// Missing host
	_, _, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")

	// Unreachable
	_, out, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: "127.0.0.1:1"})
	assert.NoError(t, err)
	assert.Contains(t, out.Error, "unreachable")

	// Call fails after init
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount {
		case 1:
			fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
		case 2:
			w.WriteHeader(200)
		case 3:
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(srv.Close)

	_, out2, _ := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: srv.Listener.Addr().String()})
	assert.Contains(t, out2.Error, "call failed")
}

func TestRemoteStatus_StatusRemote_Ugly(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount {
		case 1:
			fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
		case 2:
			w.WriteHeader(200)
		case 3:
			// JSON-RPC error
			result := map[string]any{"error": map[string]any{"code": -32000, "message": "internal error"}}
			data, _ := json.Marshal(result)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, _ := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: srv.Listener.Addr().String()})
	assert.False(t, out.Success)
	assert.Contains(t, out.Error, "internal error")

	// Unparseable response
	callCount2 := 0
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount2++
		w.Header().Set("Mcp-Session-Id", "s")
		w.Header().Set("Content-Type", "text/event-stream")
		switch callCount2 {
		case 1:
			fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
		case 2:
			w.WriteHeader(200)
		case 3:
			fmt.Fprintf(w, "data: not-json\n\n")
		}
	}))
	t.Cleanup(srv2.Close)

	_, out2, _ := s.statusRemote(context.Background(), nil, RemoteStatusInput{Host: srv2.Listener.Addr().String()})
	assert.False(t, out2.Success)
	assert.Contains(t, out2.Error, "failed to parse")
}
