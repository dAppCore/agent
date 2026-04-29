// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"syscall"
	"time"

	core "dappco.re/go"
)

func fleetExampleSubsystem(serverURL, token string) *PrepSubsystem {
	c := core.New()
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		brainURL:       serverURL,
		brainKey:       token,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

func ExamplePrepSubsystem_Connect() {
	fsys := (&core.Fs{}).NewUnrestricted()
	home := "/tmp/core-agent-fleet-example-connect"
	defer fsys.DeleteAll(home)

	if err := syscall.Setenv("CORE_HOME", home); err != nil {
		panic(err)
	}
	defer func() {
		if err := syscall.Unsetenv("CORE_HOME"); err != nil {
			panic(err)
		}
	}()

	resetFleetRuntimeState()
	defer resetFleetRuntimeState()

	originalSchedule := fleetBackoffSchedule
	originalThreshold := fleetPollingFailureThreshold
	originalHeartbeat := fleetHeartbeatInterval
	originalSleep := fleetSleep
	defer func() {
		fleetBackoffSchedule = originalSchedule
		fleetPollingFailureThreshold = originalThreshold
		fleetHeartbeatInterval = originalHeartbeat
		fleetSleep = originalSleep
	}()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"stream unavailable"}`))
	}))
	defer srv.Close()

	subsystem := fleetExampleSubsystem(srv.URL, "secret-token")
	fleetHeartbeatInterval = 0
	fleetPollingFailureThreshold = 99
	fleetBackoffSchedule = []time.Duration{time.Millisecond}

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0
	fleetSleep = func(_ context.Context, delay time.Duration) bool {
		attempts++
		cancel()
		return false
	}

	result := subsystem.Connect(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	core.Println(result.OK)
	core.Println(attempts)
	// Output:
	// true
	// 1
}

func ExamplePrepSubsystem_PollFallback() {
	fsys := (&core.Fs{}).NewUnrestricted()
	home := "/tmp/core-agent-fleet-example-poll"
	defer fsys.DeleteAll(home)

	if err := syscall.Setenv("CORE_HOME", home); err != nil {
		panic(err)
	}
	defer func() {
		if err := syscall.Unsetenv("CORE_HOME"); err != nil {
			panic(err)
		}
	}()

	resetFleetRuntimeState()
	defer resetFleetRuntimeState()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"task":{"id":7,"repo":"core/go-io","branch":"dev","task":"Fix tests","status":"assigned"}}}`))
	}))
	defer srv.Close()

	result := fleetExampleSubsystem(srv.URL, "secret-token").PollFallback(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	task := result.Value.(*FleetTask)
	core.Println(result.OK)
	core.Println(task.ID)
	// Output:
	// true
	// 7
}

func ExamplePrepSubsystem_Heartbeat() {
	fsys := (&core.Fs{}).NewUnrestricted()
	home := "/tmp/core-agent-fleet-example-heartbeat"
	defer fsys.DeleteAll(home)

	if err := syscall.Setenv("CORE_HOME", home); err != nil {
		panic(err)
	}
	defer func() {
		if err := syscall.Unsetenv("CORE_HOME"); err != nil {
			panic(err)
		}
	}()

	resetFleetRuntimeState()
	defer resetFleetRuntimeState()

	originalInterval := fleetHeartbeatInterval
	originalSleep := fleetSleep
	defer func() {
		fleetHeartbeatInterval = originalInterval
		fleetSleep = originalSleep
	}()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer srv.Close()

	fleetHeartbeatInterval = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	fleetSleep = func(_ context.Context, delay time.Duration) bool {
		if delay > 0 {
			cancel()
			return false
		}
		return true
	}

	result := fleetExampleSubsystem(srv.URL, "secret-token").Heartbeat(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	core.Println(result.OK)
	core.Println(requests)
	// Output:
	// true
	// 1
}
