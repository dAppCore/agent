// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func listedResourceURIs(t *testing.T, register func(*coremcp.Service)) []string {
	t.Helper()

	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)
	register(svc)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := svc.Server().Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListResources(context.Background(), nil)
	core.RequireNoError(t, err)

	uris := make([]string, 0, len(result.Resources))
	for _, resource := range result.Resources {
		uris = append(uris, core.Sprint(resource.URI))
	}
	return uris
}

func TestDefaults_New_Good(t *testing.T) {
	t.Setenv("MONITOR_INTERVAL", "")
	mon := New()
	core.AssertEqual(t, 2*time.Minute, mon.interval)
	core.AssertNotNil(t, mon.poke)
}

func TestZeroInterval_New_Bad(t *testing.T) {
	t.Setenv("MONITOR_INTERVAL", "")
	mon := New(Options{Interval: 0})
	core.AssertEqual(t, 2*time.Minute, mon.interval)
	core.AssertNotNil(t, mon.poke)
}

func TestEnvOverride_New_Ugly(t *testing.T) {
	t.Setenv("MONITOR_INTERVAL", "45s")
	mon := New()
	core.AssertEqual(t, 45*time.Second, mon.interval)
	core.AssertNotNil(t, mon.poke)
}

func TestCoreRegister_Register_Good(t *testing.T) {
	c := core.New()
	result := Register(c)
	core.AssertTrue(t, result.OK)
	mon, ok := result.Value.(*Subsystem)
	core.RequireTrue(t, ok)
	core.AssertNotNil(t, mon.Core())
}

func TestNilCore_Register_Bad(t *testing.T) {
	result := Register(nil)
	core.AssertTrue(t, result.OK)
	mon, ok := result.Value.(*Subsystem)
	core.RequireTrue(t, ok)
	core.AssertNil(t, mon.Core())
}

func TestFreshInstance_Register_Ugly(t *testing.T) {
	c := core.New()
	first := Register(c)
	second := Register(c)
	core.RequireTrue(t, first.OK)
	core.RequireTrue(t, second.OK)
	core.AssertTrue(t, first.Value != second.Value)
}

func TestUnknownMessage_Subsystem_HandleIPCEvents_Good(t *testing.T) {
	mon := New()
	result := mon.HandleIPCEvents(nil, "unknown")
	core.AssertTrue(t, result.OK)
	core.AssertLen(t, mon.seenRunning, 0)
}

func TestEmptyStartedEvent_Subsystem_HandleIPCEvents_Bad(t *testing.T) {
	mon := New()
	result := mon.HandleIPCEvents(nil, messages.AgentStarted{})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, mon.seenRunning[""])
}

func TestCompletedEvent_Subsystem_HandleIPCEvents_Ugly(t *testing.T) {
	mon := New()
	result := mon.HandleIPCEvents(nil, messages.AgentCompleted{Workspace: "ws-1", Status: "completed"})
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, mon.seenCompleted["ws-1"])
}

func TestConstructedMonitor_Subsystem_Name_Good(t *testing.T) {
	got := New().Name()
	core.AssertEqual(t, "monitor", got)
	core.AssertNotEmpty(t, got)
}

func TestZeroValueMonitor_Subsystem_Name_Bad(t *testing.T) {
	got := (&Subsystem{}).Name()
	core.AssertEqual(t, "monitor", got)
	core.AssertContains(t, got, "monitor")
}

func TestNilMonitor_Subsystem_Name_Ugly(t *testing.T) {
	var mon *Subsystem
	got := mon.Name()
	core.AssertEqual(t, "monitor", got)
	core.AssertNotContains(t, got, "/")
}

func TestStartedLoop_Subsystem_OnStartup_Good(t *testing.T) {
	mon := New(Options{Interval: time.Hour})
	result := mon.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, mon.done)
	core.RequireNoError(t, mon.Shutdown(context.Background()))
}

func TestZeroValueMonitor_Subsystem_OnStartup_Bad(t *testing.T) {
	mon := &Subsystem{}
	result := mon.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, mon.done)
	core.RequireNoError(t, mon.Shutdown(context.Background()))
}

func TestCancelledContext_Subsystem_OnStartup_Ugly(t *testing.T) {
	mon := New(Options{Interval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := mon.OnStartup(ctx)
	core.AssertTrue(t, result.OK)
	core.RequireNoError(t, mon.Shutdown(context.Background()))
}

func TestStartedLoop_Subsystem_OnShutdown_Good(t *testing.T) {
	mon := New(Options{Interval: time.Hour})
	mon.Start(context.Background())
	result := mon.OnShutdown(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestZeroValueMonitor_Subsystem_OnShutdown_Bad(t *testing.T) {
	result := (&Subsystem{}).OnShutdown(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestRepeatedShutdown_Subsystem_OnShutdown_Ugly(t *testing.T) {
	mon := New(Options{Interval: time.Hour})
	mon.Start(context.Background())
	core.RequireTrue(t, mon.OnShutdown(context.Background()).OK)
	core.AssertTrue(t, mon.OnShutdown(context.Background()).OK)
}

func TestBufferedChannel_Subsystem_Poke_Good(t *testing.T) {
	mon := New()
	mon.Poke()
	core.AssertLen(t, mon.poke, 1)
	core.AssertNotNil(t, mon.poke)
}

func TestZeroValueMonitor_Subsystem_Poke_Bad(t *testing.T) {
	core.AssertNotPanics(t, func() {
		(&Subsystem{}).Poke()
	})
}

func TestDoublePoke_Subsystem_Poke_Ugly(t *testing.T) {
	mon := New()
	mon.Poke()
	mon.Poke()
	core.AssertLen(t, mon.poke, 1)
	core.AssertNotNil(t, mon.poke)
}

func TestAgentStatusResource_Subsystem_RegisterTools_Good(t *testing.T) {
	names := listedResourceURIs(t, New().RegisterTools)
	core.AssertContains(t, names, "status://agents")
	core.AssertLen(t, names, 1)
}

func TestZeroValueMonitor_Subsystem_RegisterTools_Bad(t *testing.T) {
	names := listedResourceURIs(t, (&Subsystem{}).RegisterTools)
	core.AssertContains(t, names, "status://agents")
	core.AssertLen(t, names, 1)
}

func TestRepeatedRegister_Subsystem_RegisterTools_Ugly(t *testing.T) {
	names := listedResourceURIs(t, func(svc *coremcp.Service) {
		mon := New()
		mon.RegisterTools(svc)
		mon.RegisterTools(svc)
	})
	core.AssertContains(t, names, "status://agents")
	core.AssertGreaterOrEqual(t, len(names), 1)
}

func TestStartedLoop_Subsystem_Shutdown_Good(t *testing.T) {
	mon := New(Options{Interval: time.Hour})
	mon.Start(context.Background())
	err := mon.Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestZeroValueMonitor_Subsystem_Shutdown_Bad(t *testing.T) {
	err := (&Subsystem{}).Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestNilMonitor_Subsystem_Shutdown_Ugly(t *testing.T) {
	var mon *Subsystem
	core.AssertPanics(t, func() {
		_ = mon.Shutdown(context.Background())
	})
}

func TestStartLoop_Subsystem_Start_Good(t *testing.T) {
	mon := New(Options{Interval: time.Hour})
	mon.Start(context.Background())
	core.AssertNotNil(t, mon.done)
	core.RequireNoError(t, mon.Shutdown(context.Background()))
}

func TestZeroValueMonitor_Subsystem_Start_Bad(t *testing.T) {
	mon := &Subsystem{}
	core.AssertNotPanics(t, func() {
		mon.Start(context.Background())
	})
	core.AssertNotNil(t, mon.done)
	core.RequireNoError(t, mon.Shutdown(context.Background()))
}

func TestCancelledContext_Subsystem_Start_Ugly(t *testing.T) {
	mon := New(Options{Interval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	core.AssertNotPanics(t, func() {
		mon.Start(ctx)
	})
	core.RequireNoError(t, mon.Shutdown(context.Background()))
}
