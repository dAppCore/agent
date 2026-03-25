// SPDX-License-Identifier: EUPL-1.2

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

// --- dispatchRemote ---

func TestDispatchRemote_Bad_MissingHost(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, _, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Repo: "go-io", Task: "do it",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestDispatchRemote_Bad_MissingRepo(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, _, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Host: "charon", Task: "do it",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestDispatchRemote_Bad_MissingTask(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, _, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Host: "charon", Repo: "go-io",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task is required")
}

func TestDispatchRemote_Good_FullRoundtrip(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "test-session")
		w.Header().Set("Content-Type", "text/event-stream")

		switch callCount {
		case 1:
			// Initialize response
			fmt.Fprintf(w, "data: {\"result\":{}}\n\n")
		case 2:
			// Initialized notification — just accept
			w.WriteHeader(200)
		case 3:
			// Tool call response
			result := map[string]any{
				"result": map[string]any{
					"content": []map[string]any{
						{"text": `{"success":true,"agent":"codex","repo":"go-io","workspace_dir":"/ws/go-io","pid":12345}`},
					},
				},
			}
			data, _ := json.Marshal(result)
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}))
	t.Cleanup(srv.Close)

	// Override resolveHost to use our test server
	t.Setenv("AGENT_TOKEN_TESTHOST", "test-token")

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Host: srv.Listener.Addr().String(),
		Repo: "go-io",
		Task: "Fix tests",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "go-io", out.Repo)
}

func TestDispatchRemote_Bad_InitFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, _, err := s.dispatchRemote(context.Background(), nil, RemoteDispatchInput{
		Host: srv.Listener.Addr().String(),
		Repo: "go-io",
		Task: "test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP initialize failed")
}

// --- statusRemote ---

func TestStatusRemote_Bad_MissingHost(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, _, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestStatusRemote_Good_Unreachable(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Use a port that's not listening
	_, out, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{
		Host: "127.0.0.1:1",
	})
	assert.NoError(t, err) // returns output, not error
	assert.Contains(t, out.Error, "unreachable")
}

func TestStatusRemote_Good_FullRoundtrip(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Mcp-Session-Id", "test-session")
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

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.statusRemote(context.Background(), nil, RemoteStatusInput{
		Host: srv.Listener.Addr().String(),
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 5, out.Stats.Total)
	assert.Equal(t, 2, out.Stats.Running)
}
