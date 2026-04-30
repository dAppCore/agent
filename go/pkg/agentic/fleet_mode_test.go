// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestFleetMode_RegisterFleetCommands_Good_Case(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	c := core.New(core.WithOption("name", "test"))
	subsystem := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	subsystem.registerFleetCommands()

	core.AssertContains(t, c.Commands(), "login")
	core.AssertContains(t, c.Commands(), "fleet")
	core.AssertContains(t, c.Commands(), "fleet/nodes")
	core.AssertContains(t, c.Commands(), "fleet/status")
}

func TestFleetMode_CmdFleetNodes_Good_Case(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":1,"agent_id":"charon","platform":"linux","models":["codex"],"status":"online"}],"total":1}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetNodesCommand(core.NewOptions())
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "charon")
	core.AssertContains(t, output, "total: 1")
}

func TestFleetMode_CmdFleetStatus_Good_Case(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	fleetRememberBase(fleetClientConfig{APIURL: "https://api.lthn.ai", AgentID: "charon"})
	fleetRememberState("connected", "sse", "")
	fleetRememberTask(FleetTask{ID: 9, Repo: "core/go-io", Status: "assigned", Task: "Fix tests"})

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	output := captureStdout(t, func() {
		result := subsystem.cmdFleetStatus(core.NewOptions())
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "state:           connected")
	core.AssertContains(t, output, "last task:       #9 core/go-io assigned Fix tests")
}

func TestFleetMode_ListFleetNodes_Bad_Unreachable(t *testing.T) {
	t.Setenv("CORE_HOME", t.TempDir())
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	result := subsystem.listFleetNodes(context.Background(), core.NewOptions(
		core.Option{Key: "api", Value: "http://127.0.0.1:1"},
	))
	core.AssertFalse(t, result.OK)
}
