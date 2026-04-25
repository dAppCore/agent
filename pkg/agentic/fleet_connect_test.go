// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnect_BackoffOnFailure(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()

	originalSchedule := fleetBackoffSchedule
	originalThreshold := fleetPollingFailureThreshold
	originalHeartbeat := fleetHeartbeatInterval
	originalSleep := fleetSleep
	t.Cleanup(func() {
		fleetBackoffSchedule = originalSchedule
		fleetPollingFailureThreshold = originalThreshold
		fleetHeartbeatInterval = originalHeartbeat
		fleetSleep = originalSleep
		resetFleetRuntimeState()
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/fleet/events", r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"stream unavailable"}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	fleetHeartbeatInterval = 0
	fleetPollingFailureThreshold = 99
	fleetBackoffSchedule = []time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		4 * time.Millisecond,
		8 * time.Millisecond,
		16 * time.Millisecond,
		30 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	durations := []time.Duration{}
	fleetSleep = func(_ context.Context, delay time.Duration) bool {
		durations = append(durations, delay)
		if len(durations) >= 6 {
			cancel()
			return false
		}
		return true
	}

	result := subsystem.Connect(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	require.True(t, result.OK)
	assert.Equal(t, []time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		4 * time.Millisecond,
		8 * time.Millisecond,
		16 * time.Millisecond,
		30 * time.Millisecond,
	}, durations)
}

func TestPollFallback_Good(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/fleet/task/next", r.URL.Path)
		require.Equal(t, "agent_id=charon", r.URL.RawQuery)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":{"task":{"id":7,"repo":"core/go-io","branch":"dev","task":"Fix tests","status":"assigned"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.PollFallback(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	require.True(t, result.OK)

	task, ok := result.Value.(*FleetTask)
	require.True(t, ok)
	require.NotNil(t, task)
	assert.Equal(t, 7, task.ID)
	assert.Equal(t, "core/go-io", task.Repo)

	snapshot := fleetRuntimeSnapshotValue()
	assert.Equal(t, "polling", snapshot.State)
	assert.Equal(t, 7, snapshot.LastTask.ID)
}

func TestHeartbeat_Good(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()

	originalInterval := fleetHeartbeatInterval
	originalSleep := fleetSleep
	t.Cleanup(func() {
		fleetHeartbeatInterval = originalInterval
		fleetSleep = originalSleep
		resetFleetRuntimeState()
	})

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/fleet/heartbeat", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		assert.Equal(t, "charon", payload["agent_id"])
		assert.Equal(t, "online", payload["status"])

		requests++
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	fleetHeartbeatInterval = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	fleetSleep = func(_ context.Context, delay time.Duration) bool {
		if delay > 0 {
			cancel()
			return false
		}
		return true
	}

	result := subsystem.Heartbeat(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	require.True(t, result.OK)
	assert.Equal(t, 1, requests)
	assert.NotEmpty(t, fleetRuntimeSnapshotValue().LastHeartbeatAt)
}
