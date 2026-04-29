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

func TestBackoffOnFailure_PrepSubsystem_Connect_Good(t *testing.T) {
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
		core.AssertEqual(t, "/v1/fleet/events", r.URL.Path)
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
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, []time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		4 * time.Millisecond,
		8 * time.Millisecond,
		16 * time.Millisecond,
		30 * time.Millisecond,
	}, durations)
}

func TestPollFallback_PrepSubsystem_PollFallback_Good(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/fleet/task/next", r.URL.Path)
		core.AssertEqual(t, "agent_id=charon", r.URL.RawQuery)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":{"task":{"id":7,"repo":"core/go-io","branch":"dev","task":"Fix tests","status":"assigned"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.PollFallback(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	core.RequireTrue(t, result.OK)

	task, ok := result.Value.(*FleetTask)
	core.RequireTrue(t, ok)
	core.AssertNotNil(t, task)
	core.AssertEqual(t, 7, task.ID)
	core.AssertEqual(t, "core/go-io", task.Repo)

	snapshot := fleetRuntimeSnapshotValue()
	core.AssertEqual(t, "polling", snapshot.State)
	core.AssertEqual(t, 7, snapshot.LastTask.ID)
}

func TestHeartbeat_PrepSubsystem_Heartbeat_Good(t *testing.T) {
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
		core.AssertEqual(t, "/v1/fleet/heartbeat", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "charon", payload["agent_id"])
		core.AssertEqual(t, "online", payload["status"])

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
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, 1, requests)
	core.AssertNotEmpty(t, fleetRuntimeSnapshotValue().LastHeartbeatAt)
}

func TestFleetConnect_PrepSubsystem_Connect_Good(t *testing.T) {
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
		core.AssertEqual(t, "/v1/fleet/events", r.URL.Path)
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
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, []time.Duration{
		time.Millisecond,
		2 * time.Millisecond,
		4 * time.Millisecond,
		8 * time.Millisecond,
		16 * time.Millisecond,
		30 * time.Millisecond,
	}, durations)
}

func TestFleetConnect_PrepSubsystem_Connect_Bad(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.Connect(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestFleetConnect_PrepSubsystem_Connect_Ugly(t *testing.T) {
	resetFleetRuntimeState()
	originalHeartbeat := fleetHeartbeatInterval
	t.Cleanup(func() {
		fleetHeartbeatInterval = originalHeartbeat
		resetFleetRuntimeState()
	})

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	fleetHeartbeatInterval = 0
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := subsystem.Connect(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "offline", fleetRuntimeSnapshotValue().State)
}

func TestFleetConnect_PrepSubsystem_PollFallback_Good(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/fleet/task/next", r.URL.Path)
		core.AssertEqual(t, "agent_id=charon", r.URL.RawQuery)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":{"task":{"id":7,"repo":"core/go-io","branch":"dev","task":"Fix tests","status":"assigned"}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.PollFallback(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	core.RequireTrue(t, result.OK)

	task, ok := result.Value.(*FleetTask)
	core.RequireTrue(t, ok)
	core.AssertNotNil(t, task)
	core.AssertEqual(t, 7, task.ID)
	core.AssertEqual(t, "core/go-io", task.Repo)

	snapshot := fleetRuntimeSnapshotValue()
	core.AssertEqual(t, "polling", snapshot.State)
	core.AssertEqual(t, 7, snapshot.LastTask.ID)
}

func TestFleetConnect_PrepSubsystem_PollFallback_Bad(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.PollFallback(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestFleetConnect_PrepSubsystem_PollFallback_Ugly(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := subsystem.PollFallback(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestFleetConnect_PrepSubsystem_Heartbeat_Good(t *testing.T) {
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
		core.AssertEqual(t, "/v1/fleet/heartbeat", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "charon", payload["agent_id"])
		core.AssertEqual(t, "online", payload["status"])

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
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, 1, requests)
	core.AssertNotEmpty(t, fleetRuntimeSnapshotValue().LastHeartbeatAt)
}

func TestFleetConnect_PrepSubsystem_Heartbeat_Bad(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.Heartbeat(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestFleetConnect_PrepSubsystem_Heartbeat_Ugly(t *testing.T) {
	resetFleetRuntimeState()
	originalHeartbeat := fleetHeartbeatInterval
	t.Cleanup(func() {
		fleetHeartbeatInterval = originalHeartbeat
		resetFleetRuntimeState()
	})

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	fleetHeartbeatInterval = 0
	result := subsystem.Heartbeat(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}
