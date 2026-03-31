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

func testPrepWithPlatformServer(t *testing.T, srv *httptest.Server, token string) *PrepSubsystem {
	t.Helper()

	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_API_KEY", token)
	t.Setenv("CORE_BRAIN_KEY", "")

	c := core.New()
	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		brainKey:       token,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	if srv != nil {
		subsystem.brainURL = srv.URL
	}

	return subsystem
}

func TestPlatform_HandleFleetRegister_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/fleet/register", r.URL.Path)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "charon", payload["agent_id"])
		require.Equal(t, "linux", payload["platform"])

		_, _ = w.Write([]byte(`{"data":{"id":1,"agent_id":"charon","platform":"linux","models":["codex"],"capabilities":["go","review"],"status":"online"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetRegister(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "platform", Value: "linux"},
		core.Option{Key: "models", Value: "codex"},
		core.Option{Key: "capabilities", Value: "go,review"},
	))
	require.True(t, result.OK)

	node, ok := result.Value.(FleetNode)
	require.True(t, ok)
	assert.Equal(t, "charon", node.AgentID)
	assert.Equal(t, "linux", node.Platform)
	assert.Equal(t, []string{"codex"}, node.Models)
	assert.Equal(t, []string{"go", "review"}, node.Capabilities)
}

func TestPlatform_HandleFleetRegister_Bad(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")

	result := subsystem.handleFleetRegister(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "platform", Value: "linux"},
	))
	assert.False(t, result.OK)
}

func TestPlatform_HandleFleetRegister_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{broken json`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetRegister(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "platform", Value: "linux"},
	))
	assert.False(t, result.OK)
}

func TestPlatform_HandleFleetNextTask_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/fleet/task/next", r.URL.Path)
		require.Equal(t, "charon", r.URL.Query().Get("agent_id"))
		require.Equal(t, []string{"go", "review"}, r.URL.Query()["capabilities[]"])
		_, _ = w.Write([]byte(`{"data":{"id":9,"repo":"core/go-io","branch":"dev","task":"Fix tests","template":"coding","agent_model":"codex:gpt-5.4","status":"in_progress","findings":[{"file":"main.go"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNextTask(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "capabilities", Value: `["go","review"]`},
	))
	require.True(t, result.OK)

	task, ok := result.Value.(*FleetTask)
	require.True(t, ok)
	require.NotNil(t, task)
	assert.Equal(t, 9, task.ID)
	assert.Equal(t, "core/go-io", task.Repo)
	assert.Len(t, task.Findings, 1)
}

func TestPlatform_HandleFleetNextTask_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"queue unavailable"}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNextTask(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	assert.False(t, result.OK)
}

func TestPlatform_HandleFleetNextTask_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":null}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNextTask(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	require.True(t, result.OK)

	task, ok := result.Value.(*FleetTask)
	require.True(t, ok)
	assert.Nil(t, task)
}

func TestPlatform_HandleFleetEvents_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/fleet/events", r.URL.Path)
		require.Equal(t, "charon", r.URL.Query().Get("agent_id"))
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte("data: {\"event\":\"task.assigned\",\"type\":\"task.assigned\",\"agent_id\":\"charon\",\"task_id\":9,\"repo\":\"core/go-io\",\"branch\":\"dev\",\"status\":\"assigned\"}\n\n"))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetEvents(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(FleetEventOutput)
	require.True(t, ok)
	assert.Equal(t, "task.assigned", output.Event.Event)
	assert.Equal(t, "charon", output.Event.AgentID)
	assert.Equal(t, 9, output.Event.TaskID)
	assert.Equal(t, "core/go-io", output.Event.Repo)
	assert.Equal(t, "dev", output.Event.Branch)
}

func TestPlatform_HandleFleetEvents_Bad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"event stream unavailable"}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetEvents(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	assert.False(t, result.OK)
}

func TestPlatform_HandleFleetEvents_Ugly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data: not-json\n\n"))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetEvents(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	assert.False(t, result.OK)
}

func TestPlatform_HandleSyncStatus_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/status", r.URL.Path)
		require.Equal(t, "charon", r.URL.Query().Get("agent_id"))
		_, _ = w.Write([]byte(`{"data":{"agent_id":"charon","status":"online","last_push_at":"2026-03-31T08:00:00Z","last_pull_at":"2026-03-31T08:05:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSyncStatus(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	require.True(t, result.OK)

	status, ok := result.Value.(SyncStatusOutput)
	require.True(t, ok)
	assert.Equal(t, "charon", status.AgentID)
	assert.Equal(t, "online", status.Status)
	assert.Equal(t, "2026-03-31T08:00:00Z", status.LastPushAt)
	assert.Empty(t, status.RemoteError)
}

func TestPlatform_HandleCreditsHistory_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/credits/history/charon", r.URL.Path)
		require.Equal(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":[{"id":1,"task_type":"fleet-task","amount":2,"balance_after":7,"description":"Fleet task completed"}],"total":1}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleCreditsHistory(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "limit", Value: 5},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(CreditsHistoryOutput)
	require.True(t, ok)
	assert.Equal(t, 1, output.Total)
	require.Len(t, output.Entries, 1)
	assert.Equal(t, 7, output.Entries[0].BalanceAfter)
}

func TestPlatform_HandleFleetNodes_Good_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"nodes":[{"id":1,"workspace_id":7,"agent_id":"charon","platform":"linux","models":["codex"],"status":"online"}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNodes(context.Background(), core.NewOptions())
	require.True(t, result.OK)

	output, ok := result.Value.(FleetNodesOutput)
	require.True(t, ok)
	require.Len(t, output.Nodes, 1)
	assert.Equal(t, 1, output.Total)
	assert.Equal(t, 7, output.Nodes[0].WorkspaceID)
}

func TestPlatform_HandleCreditsHistory_Good_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"entries":[{"id":1,"workspace_id":3,"fleet_node_id":9,"task_type":"fleet-task","amount":2,"balance_after":7}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleCreditsHistory(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(CreditsHistoryOutput)
	require.True(t, ok)
	require.Len(t, output.Entries, 1)
	assert.Equal(t, 1, output.Total)
	assert.Equal(t, 3, output.Entries[0].WorkspaceID)
	assert.Equal(t, 9, output.Entries[0].FleetNodeID)
}

func TestPlatform_HandleSubscriptionDetect_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/subscription/detect", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		apiKeys, ok := payload["api_keys"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "k", apiKeys["openai"])

		_, _ = w.Write([]byte(`{"data":{"providers":{"claude":true,"openai":true},"available":["claude","openai"]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSubscriptionDetect(context.Background(), core.NewOptions(
		core.Option{Key: "api_keys", Value: `{"openai":"k"}`},
	))
	require.True(t, result.OK)

	capabilities, ok := result.Value.(SubscriptionCapabilities)
	require.True(t, ok)
	assert.True(t, capabilities.Providers["claude"])
	assert.Equal(t, []string{"claude", "openai"}, capabilities.Available)
}

func TestPlatform_HandleSubscriptionDetect_Good_ProvidersOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"providers":{"claude":true,"openai":false,"gemini":true}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSubscriptionDetect(context.Background(), core.NewOptions())
	require.True(t, result.OK)

	capabilities, ok := result.Value.(SubscriptionCapabilities)
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"claude", "gemini"}, capabilities.Available)
}

func TestPlatform_HandleSyncStatus_Good_LocalStateFallback(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	t.Setenv("CORE_AGENT_API_KEY", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	recordSyncPush(time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC))
	recordSyncPull(time.Date(2026, 3, 31, 8, 5, 0, 0, time.UTC))

	c := core.New()
	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	result := subsystem.handleSyncStatus(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	require.True(t, result.OK)

	status, ok := result.Value.(SyncStatusOutput)
	require.True(t, ok)
	assert.Equal(t, "2026-03-31T08:00:00Z", status.LastPushAt)
	assert.Equal(t, "2026-03-31T08:05:00Z", status.LastPullAt)
}
