// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestFleetModeCov_CmdFleet_Good_RoutesToNodes — "fleet nodes" (action via the
// _arg positional) routes to the nodes lister and prints the node row.
func TestFleetModeCov_CmdFleet_Good_RoutesToNodes(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":1,"agent_id":"charon","platform":"linux","models":["codex"],"status":"online"}],"total":1}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleet(core.NewOptions(core.Option{Key: "_arg", Value: "nodes"}))
		core.RequireTrue(t, result.OK)
	})
	core.AssertContains(t, output, "charon")
	core.AssertContains(t, output, "total: 1")
}

// TestFleetModeCov_CmdFleet_Good_RoutesToStatus — "fleet status" routes to the
// status printer.
func TestFleetModeCov_CmdFleet_Good_RoutesToStatus(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleet(core.NewOptions(core.Option{Key: "_arg", Value: "status"}))
		core.RequireTrue(t, result.OK)
	})
	core.AssertContains(t, output, "state:")
	core.AssertContains(t, output, "transport:")
}

// TestFleetModeCov_CmdFleet_Good_HelpPrintsUsage — an empty action with no
// agent-id prints usage and returns OK.
func TestFleetModeCov_CmdFleet_Good_HelpPrintsUsage(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	output := captureStdout(t, func() {
		result := subsystem.cmdFleet(core.NewOptions(core.Option{Key: "help", Value: true}))
		core.RequireTrue(t, result.OK)
	})
	core.AssertContains(t, output, "usage: core-agent fleet")
}

// TestFleetModeCov_CmdFleet_Bad_UnknownAction — an unrecognised action prints
// usage and returns a failure carrying the unknown-command error.
func TestFleetModeCov_CmdFleet_Bad_UnknownAction(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	var result core.Result
	output := captureStdout(t, func() {
		result = subsystem.cmdFleet(core.NewOptions(core.Option{Key: "_arg", Value: "frobnicate"}))
	})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "usage: core-agent fleet")

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "unknown fleet command: frobnicate")
}

// TestFleetModeCov_CmdFleet_Ugly_ConnectValidationFails — with an agent-id set
// but no fleet api key, Connect fails config validation and cmdFleet prints the
// error and returns a failure (the connect-failure branch). No network loop is
// entered because validation rejects before the connect loop.
func TestFleetModeCov_CmdFleet_Ugly_ConnectValidationFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	// Clear any inherited fleet key env so the token requirement fails.
	t.Setenv("CORE_AGENT_API_KEY", "")
	t.Setenv("CORE_FLEET_API_KEY", "")

	var result core.Result
	output := captureStdout(t, func() {
		result = subsystem.cmdFleet(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
			core.Option{Key: "api", Value: "https://api.lthn.ai"},
		))
	})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "no fleet api key configured")
}

// TestFleetModeCov_CmdFleetNodesCommand_Good_EmptyNodes — an empty node list
// prints "no fleet nodes" and returns the (empty) output.
func TestFleetModeCov_CmdFleetNodesCommand_Good_EmptyNodes(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"total":0}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetNodesCommand(core.NewOptions())
		core.RequireTrue(t, result.OK)

		out, ok := result.Value.(FleetNodesOutput)
		core.RequireTrue(t, ok)
		core.AssertLen(t, out.Nodes, 0)
	})
	core.AssertContains(t, output, "no fleet nodes")
}

// TestFleetModeCov_CmdFleetNodesCommand_Bad_Unreachable — an unreachable API
// makes the underlying lister fail; cmdFleetNodesCommand prints the error and
// returns a failure.
func TestFleetModeCov_CmdFleetNodesCommand_Bad_Unreachable(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	var result core.Result
	output := captureStdout(t, func() {
		result = subsystem.cmdFleetNodesCommand(core.NewOptions(
			core.Option{Key: "api", Value: "http://127.0.0.1:1"},
		))
	})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
}

// TestFleetModeCov_CmdFleetStatus_Good_OfflineDefaults — with no remembered
// runtime state the status prints the offline/none/never defaults and "last
// task: none".
func TestFleetModeCov_CmdFleetStatus_Good_OfflineDefaults(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetStatus(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
		))
		core.RequireTrue(t, result.OK)
	})
	core.AssertContains(t, output, "state:           offline")
	core.AssertContains(t, output, "transport:       none")
	core.AssertContains(t, output, "last heartbeat:  never")
	core.AssertContains(t, output, "last task:       none")
}

// TestFleetModeCov_CmdFleetStatus_Ugly_AllOptionalFields — with every optional
// timestamp + task + error remembered, the status prints each conditional line.
func TestFleetModeCov_CmdFleetStatus_Ugly_AllOptionalFields(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	fleetRememberBase(fleetClientConfig{APIURL: "https://api.lthn.ai", AgentID: "charon"})
	fleetRememberState("connected", "sse", "")
	fleetRememberConnected()
	fleetRememberHeartbeat()
	fleetRememberEvent(FleetEvent{Event: "task.assigned", TaskID: 7, Repo: "core/go-io"})
	fleetRememberTask(FleetTask{ID: 7, Repo: "core/go-io", Status: "assigned", Task: "Fix tests"})
	// Set the error last — heartbeat/event/task all clear LastError.
	fleetRememberState("disconnected", "sse", "stream dropped")

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetStatus(core.NewOptions())
		core.RequireTrue(t, result.OK)
	})
	core.AssertContains(t, output, "last connected:")
	core.AssertContains(t, output, "last heartbeat:")
	core.AssertContains(t, output, "last event:")
	core.AssertContains(t, output, "task received:")
	core.AssertContains(t, output, "last error:      stream dropped")
}

// TestFleetModeCov_ListFleetNodes_Ugly_UnparseableBody — a non-JSON body makes
// the platform request fail to parse, so listFleetNodes returns the request
// error and cmdFleetNodesCommand surfaces it. (The "invalid fleet nodes
// payload" type-assert arm is unreachable from HTTP: fleetJSONRequest always
// returns a map[string]any on OK.)
func TestFleetModeCov_ListFleetNodes_Ugly_UnparseableBody(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	var result core.Result
	output := captureStdout(t, func() {
		result = subsystem.cmdFleetNodesCommand(core.NewOptions())
	})
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "agentic.fleet.nodes")
}

// TestFleetModeCov_FleetTaskSummary_Good_VariousShapes — the summary builds from
// whichever of id/repo/task is present (the early-return guard keys off
// id/repo/task, so a status-only task is treated as empty), and the status
// segment only appears once another field is present.
func TestFleetModeCov_FleetTaskSummary_Good_VariousShapes(t *testing.T) {
	core.AssertEqual(t, "", fleetTaskSummary(FleetTask{}))
	// Status alone does not satisfy the non-empty guard.
	core.AssertEqual(t, "", fleetTaskSummary(FleetTask{Status: "assigned"}))
	core.AssertEqual(t, "#5", fleetTaskSummary(FleetTask{ID: 5}))
	core.AssertEqual(t, "core/go-io", fleetTaskSummary(FleetTask{Repo: "core/go-io"}))
	core.AssertEqual(t, "Fix tests", fleetTaskSummary(FleetTask{Task: "Fix tests"}))
	// Repo present → the status segment is appended after it.
	core.AssertEqual(t, "core/go-io assigned", fleetTaskSummary(FleetTask{Repo: "core/go-io", Status: "assigned"}))
	core.AssertEqual(t, "#5 core/go-io assigned Fix tests", fleetTaskSummary(FleetTask{
		ID:     5,
		Repo:   "core/go-io",
		Status: "assigned",
		Task:   "Fix tests",
	}))
}
