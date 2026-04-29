// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	core "dappco.re/go"
)

func testDefaultClientWithTrustedServerCert(t *testing.T, srv *httptest.Server) *http.Client {
	t.Helper()

	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots}
	core.AssertFalse(t, transport.TLSClientConfig.InsecureSkipVerify)

	return &http.Client{
		Timeout:   defaultClient.Timeout,
		Transport: transport,
	}
}

func testUseDefaultClient(t *testing.T, client *http.Client) {
	t.Helper()

	original := defaultClient
	defaultClient = client
	t.Cleanup(func() {
		defaultClient = original
	})
}

func testPrepWithPlatformServer(t *testing.T, srv *httptest.Server, token string) *PrepSubsystem {
	t.Helper()

	root := t.TempDir()
	setTestWorkspace(t, root)
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

func TestPlatform_HandleFleetRegister_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/fleet/register", r.URL.Path)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "charon", payload["agent_id"])
		core.AssertEqual(t, "linux", payload["platform"])

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
	core.RequireTrue(t, result.OK)

	node, ok := result.Value.(FleetNode)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "charon", node.AgentID)
	core.AssertEqual(t, "linux", node.Platform)
	core.AssertEqual(t, []string{"codex"}, node.Models)
	core.AssertEqual(t, []string{"go", "review"}, node.Capabilities)
	core.AssertNil(t, node.CurrentTaskID)
}

func TestPlatform_HandleFleetRegister_Good_TrustedTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertNotNil(t, r.TLS)
		core.AssertEqual(t, "/v1/fleet/register", r.URL.Path)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		_, _ = w.Write([]byte(`{"data":{"id":2,"agent_id":"charon","platform":"linux","status":"online"}}`))
	}))
	defer server.Close()

	testUseDefaultClient(t, testDefaultClientWithTrustedServerCert(t, server))

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetRegister(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "platform", Value: "linux"},
	))
	core.RequireTrue(t, result.OK)

	node, ok := result.Value.(FleetNode)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 2, node.ID)
	core.AssertEqual(t, "charon", node.AgentID)
	core.AssertEqual(t, "linux", node.Platform)
	core.AssertEqual(t, "online", node.Status)
}

func TestPlatform_HandleFleetRegister_Bad_UntrustedTLSCert(t *testing.T) {
	var called atomic.Bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		_, _ = w.Write([]byte(`{"data":{"id":3,"agent_id":"charon","platform":"linux","status":"online"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetRegister(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "platform", Value: "linux"},
	))
	core.AssertFalse(t, result.OK)
	core.AssertFalse(t, called.Load())

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "platform request failed")
	core.AssertTrue(t,
		core.Contains(err.Error(), "certificate") ||
			core.Contains(err.Error(), "x509") ||
			core.Contains(err.Error(), "tls"),
	)
}

func TestPlatform_HandleFleetHeartbeat_Good_ComputeBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/fleet/heartbeat", r.URL.Path)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)

		budget, ok := payload["compute_budget"].(map[string]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, 2.5, budget["max_daily_hours"])
		core.AssertEqual(t, 12.5, budget["max_weekly_cost_usd"])
		core.AssertEqual(t, "22:00", budget["quiet_start"])
		core.AssertEqual(t, "06:00", budget["quiet_end"])
		core.AssertEqual(t, []any{"codex", "gpt-5.4"}, budget["prefer_models"])
		core.AssertEqual(t, []any{"gpt-4.1"}, budget["avoid_models"])

		_, _ = w.Write([]byte(`{"data":{"id":1,"agent_id":"charon","platform":"linux","status":"online","compute_budget":{"max_daily_hours":2.5,"max_weekly_cost_usd":12.5,"quiet_start":"22:00","quiet_end":"06:00","prefer_models":["codex","gpt-5.4"],"avoid_models":["gpt-4.1"]}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetHeartbeat(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "status", Value: "online"},
		core.Option{Key: "compute_budget", Value: `{"max_daily_hours":2.5,"max_weekly_cost_usd":12.5,"quiet_start":"22:00","quiet_end":"06:00","prefer_models":["codex","gpt-5.4"],"avoid_models":["gpt-4.1"]}`},
	))
	core.RequireTrue(t, result.OK)

	node, ok := result.Value.(FleetNode)
	core.RequireTrue(t, ok)
	core.AssertNotNil(t, node.ComputeBudget)
	core.AssertEqual(t, 2.5, node.ComputeBudget.MaxDailyHours)
	core.AssertEqual(t, 12.5, node.ComputeBudget.MaxWeeklyCostUSD)
	core.AssertEqual(t, "22:00", node.ComputeBudget.QuietStart)
	core.AssertEqual(t, "06:00", node.ComputeBudget.QuietEnd)
	core.AssertEqual(t, []string{"codex", "gpt-5.4"}, node.ComputeBudget.PreferModels)
	core.AssertEqual(t, []string{"gpt-4.1"}, node.ComputeBudget.AvoidModels)
}

func TestPlatform_HandleFleetRegister_Bad_Case(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "")

	result := subsystem.handleFleetRegister(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "platform", Value: "linux"},
	))
	core.AssertFalse(t, result.OK)
}

func TestPlatform_HandleFleetRegister_Ugly_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{broken json`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetRegister(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "platform", Value: "linux"},
	))
	core.AssertFalse(t, result.OK)
}

func TestPlatform_HandleFleetNextTask_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/fleet/task/next", r.URL.Path)
		core.AssertEqual(t, "charon", r.URL.Query().Get("agent_id"))
		core.AssertEqual(t, []string{"go", "review"}, r.URL.Query()["capabilities[]"])
		_, _ = w.Write([]byte(`{"data":{"id":9,"repo":"core/go-io","branch":"dev","task":"Fix tests","template":"coding","agent_model":"codex:gpt-5.4","status":"in_progress","findings":[{"file":"main.go"}]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNextTask(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "capabilities", Value: `["go","review"]`},
	))
	core.RequireTrue(t, result.OK)

	task, ok := result.Value.(*FleetTask)
	core.RequireTrue(t, ok)
	core.AssertNotNil(t, task)
	core.AssertEqual(t, 9, task.ID)
	core.AssertEqual(t, "core/go-io", task.Repo)
	core.AssertLen(t, task.Findings, 1)
}

func TestPlatform_HandleFleetNextTask_Bad_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"queue unavailable"}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNextTask(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	core.AssertFalse(t, result.OK)
}

func TestPlatform_HandleFleetNextTask_Ugly_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":null}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNextTask(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	core.RequireTrue(t, result.OK)

	task, ok := result.Value.(*FleetTask)
	core.RequireTrue(t, ok)
	core.AssertNil(t, task)
}

func TestPlatform_ParseFleetNode_Good_CurrentTaskID(t *testing.T) {
	node := parseFleetNode(map[string]any{
		"agent_id":        "charon",
		"platform":        "linux",
		"status":          "online",
		"current_task_id": 17,
	})

	core.AssertNotNil(t, node.CurrentTaskID)
	core.AssertEqual(t, 17, *node.CurrentTaskID)
}

func TestPlatform_ParseFleetNode_Bad_CurrentTaskIDMissing(t *testing.T) {
	node := parseFleetNode(map[string]any{
		"agent_id": "charon",
		"platform": "linux",
		"status":   "online",
	})

	core.AssertNil(t, node.CurrentTaskID)
}

func TestPlatform_ParseFleetNode_Ugly_CurrentTaskIDNull(t *testing.T) {
	node := parseFleetNode(map[string]any{
		"agent_id":        "charon",
		"platform":        "linux",
		"status":          "online",
		"current_task_id": nil,
	})

	core.AssertNil(t, node.CurrentTaskID)
}

func TestPlatform_HandleFleetEvents_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/fleet/events", r.URL.Path)
		core.AssertEqual(t, "charon", r.URL.Query().Get("agent_id"))
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte("event: task.assigned\ndata: {\"agent_id\":\"charon\",\"task_id\":9,\"repo\":\"core/go-io\",\"branch\":\"dev\",\"status\":\"assigned\"}\n\n"))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetEvents(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(FleetEventOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "task.assigned", output.Event.Event)
	core.AssertEqual(t, "charon", output.Event.AgentID)
	core.AssertEqual(t, 9, output.Event.TaskID)
	core.AssertEqual(t, "core/go-io", output.Event.Repo)
	core.AssertEqual(t, "dev", output.Event.Branch)
}

func TestPlatform_EventPayloadValue_Good_EventAndData(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	payload := subsystem.eventPayloadValue("event: task.assigned\ndata: {\"agent_id\":\"charon\",\"task_id\":9,\"repo\":\"core/go-io\"}")

	core.AssertNotNil(t, payload)
	core.AssertEqual(t, "task.assigned", payload["event"])
	core.AssertEqual(t, "task.assigned", payload["type"])
	core.AssertEqual(t, "charon", payload["agent_id"])
	core.AssertEqual(t, 9.0, payload["task_id"])
	core.AssertEqual(t, "core/go-io", payload["repo"])
}

func TestPlatform_EventPayloadValue_Bad_Empty(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	payload := subsystem.eventPayloadValue("")

	core.AssertNil(t, payload)
}

func TestPlatform_EventPayloadValue_Ugly_InvalidData(t *testing.T) {
	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")

	payload := subsystem.eventPayloadValue("event: task.assigned\ndata: not-json")

	core.AssertNotNil(t, payload)
	core.AssertEqual(t, "task.assigned", payload["event"])
	core.AssertEqual(t, "task.assigned", payload["type"])
	core.AssertEqual(t, "not-json", payload["data"])
}

func TestPlatform_HandleFleetEvents_Good_FallbackToTaskNext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/fleet/events":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"event stream unavailable"}`))
		case "/v1/fleet/task/next":
			core.AssertEqual(t, "charon", r.URL.Query().Get("agent_id"))
			_, _ = w.Write([]byte(`{"data":{"id":12,"repo":"core/go-io","branch":"dev","task":"Fix tests","template":"coding","agent_model":"codex","status":"assigned"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetEvents(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(FleetEventOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "task.assigned", output.Event.Event)
	core.AssertEqual(t, "charon", output.Event.AgentID)
	core.AssertEqual(t, 12, output.Event.TaskID)
	core.AssertEqual(t, "core/go-io", output.Event.Repo)
	core.AssertEqual(t, "dev", output.Event.Branch)
}

func TestPlatform_FleetEventsTool_Good_ForwardsCapabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/fleet/events":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"event stream unavailable"}`))
		case "/v1/fleet/task/next":
			core.AssertEqual(t, "charon", r.URL.Query().Get("agent_id"))
			core.AssertEqual(t, []string{"go", "review"}, r.URL.Query()["capabilities[]"])
			_, _ = w.Write([]byte(`{"data":{"id":13,"repo":"core/go-io","branch":"dev","task":"Fix tests","template":"coding","agent_model":"codex","status":"assigned"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	_, output, err := subsystem.fleetEventsTool(context.Background(), nil, FleetEventsInput{
		AgentID:      "charon",
		Capabilities: []string{"go", "review"},
	})
	core.RequireNoError(t, err)

	core.AssertEqual(t, "task.assigned", output.Event.Event)
	core.AssertEqual(t, "charon", output.Event.AgentID)
	core.AssertEqual(t, 13, output.Event.TaskID)
	core.AssertEqual(t, "core/go-io", output.Event.Repo)
	core.AssertEqual(t, "dev", output.Event.Branch)
}

func TestPlatform_HandleFleetEvents_Bad_NoTaskAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/fleet/events":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"event stream unavailable"}`))
		case "/v1/fleet/task/next":
			_, _ = w.Write([]byte(`{"data":null}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetEvents(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	core.AssertFalse(t, result.OK)
}

func TestPlatform_HandleFleetEvents_Ugly_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/fleet/events":
			_, _ = w.Write([]byte("data: not-json\n\n"))
		case "/v1/fleet/task/next":
			_, _ = w.Write([]byte(`{"data":null}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetEvents(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	core.AssertFalse(t, result.OK)
}

func TestPlatform_HandleFleetCompleteTask_Good_AwardsCredits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/fleet/task/complete":
			core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

			bodyResult := core.ReadAll(r.Body)
			core.RequireTrue(t, bodyResult.OK)

			var payload map[string]any
			parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
			core.RequireTrue(t, parseResult.OK)
			core.AssertEqual(t, "charon", payload["agent_id"])
			core.AssertEqual(t, 9, int(payload["task_id"].(float64)))

			_, _ = w.Write([]byte(`{"data":{"task":{"id":9,"fleet_node_id":4,"repo":"core/go-io","task":"Fix tests","status":"completed"}}}`))
		case "/v1/credits/award":
			core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))

			bodyResult := core.ReadAll(r.Body)
			core.RequireTrue(t, bodyResult.OK)

			var payload map[string]any
			parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
			core.RequireTrue(t, parseResult.OK)
			core.AssertEqual(t, "charon", payload["agent_id"])
			core.AssertEqual(t, "fleet-task", payload["task_type"])
			core.AssertEqual(t, 2, int(payload["amount"].(float64)))
			core.AssertEqual(t, 4, int(payload["fleet_node_id"].(float64)))
			core.AssertEqual(t, "Fleet task completed", payload["description"])

			_, _ = w.Write([]byte(`{"data":{"entry":{"id":7,"task_type":"fleet-task","amount":2,"balance_after":11}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetCompleteTask(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "task_id", Value: 9},
	))
	core.RequireTrue(t, result.OK)

	task, ok := result.Value.(FleetTask)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 9, task.ID)
	core.AssertEqual(t, 4, task.FleetNodeID)
	core.AssertEqual(t, "completed", task.Status)
}

func TestPlatform_HandleSyncStatus_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/status", r.URL.Path)
		core.AssertEqual(t, "charon", r.URL.Query().Get("agent_id"))
		_, _ = w.Write([]byte(`{"data":{"agent_id":"charon","status":"online","last_push_at":"2026-03-31T08:00:00Z","last_pull_at":"2026-03-31T08:05:00Z"}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSyncStatus(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
	))
	core.RequireTrue(t, result.OK)

	status, ok := result.Value.(SyncStatusOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "charon", status.AgentID)
	core.AssertEqual(t, "online", status.Status)
	core.AssertEqual(t, "2026-03-31T08:00:00Z", status.LastPushAt)
	core.AssertEmpty(t, status.RemoteError)
}

func TestPlatform_HandleCreditsHistory_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/credits/history/charon", r.URL.Path)
		core.AssertEqual(t, "5", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"data":[{"id":1,"task_type":"fleet-task","amount":2,"balance_after":7,"description":"Fleet task completed"}],"total":1}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleCreditsHistory(context.Background(), core.NewOptions(
		core.Option{Key: "agent_id", Value: "charon"},
		core.Option{Key: "limit", Value: 5},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(CreditsHistoryOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, output.Total)
	core.AssertLen(t, output.Entries, 1)
	core.AssertEqual(t, 7, output.Entries[0].BalanceAfter)
}

func TestPlatform_HandleFleetNodes_Good_NestedEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"nodes":[{"id":1,"workspace_id":7,"agent_id":"charon","platform":"linux","models":["codex"],"status":"online","compute_budget":{"max_daily_hours":3,"quiet_start":"22:00","prefer_models":["codex"]}}],"total":1}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleFleetNodes(context.Background(), core.NewOptions())
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(FleetNodesOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, output.Nodes, 1)
	core.AssertEqual(t, 1, output.Total)
	core.AssertEqual(t, 7, output.Nodes[0].WorkspaceID)
	core.AssertNotNil(t, output.Nodes[0].ComputeBudget)
	core.AssertEqual(t, 3.0, output.Nodes[0].ComputeBudget.MaxDailyHours)
	core.AssertEqual(t, "22:00", output.Nodes[0].ComputeBudget.QuietStart)
	core.AssertEqual(t, []string{"codex"}, output.Nodes[0].ComputeBudget.PreferModels)
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
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(CreditsHistoryOutput)
	core.RequireTrue(t, ok)
	core.AssertLen(t, output.Entries, 1)
	core.AssertEqual(t, 1, output.Total)
	core.AssertEqual(t, 3, output.Entries[0].WorkspaceID)
	core.AssertEqual(t, 9, output.Entries[0].FleetNodeID)
}

func TestPlatform_HandleSubscriptionDetect_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/subscription/detect", r.URL.Path)
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		apiKeys, ok := payload["api_keys"].(map[string]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, "k", apiKeys["openai"])

		_, _ = w.Write([]byte(`{"data":{"providers":{"claude":true,"openai":true},"available":["claude","openai"]}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSubscriptionDetect(context.Background(), core.NewOptions(
		core.Option{Key: "api_keys", Value: `{"openai":"k"}`},
	))
	core.RequireTrue(t, result.OK)

	capabilities, ok := result.Value.(SubscriptionCapabilities)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, capabilities.Providers["claude"])
	core.AssertEqual(t, []string{"claude", "openai"}, capabilities.Available)
}

func TestPlatform_HandleSubscriptionDetect_Good_ProvidersOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"providers":{"claude":true,"openai":false,"gemini":true}}}`))
	}))
	defer server.Close()

	subsystem := testPrepWithPlatformServer(t, server, "secret-token")
	result := subsystem.handleSubscriptionDetect(context.Background(), core.NewOptions())
	core.RequireTrue(t, result.OK)

	capabilities, ok := result.Value.(SubscriptionCapabilities)
	core.RequireTrue(t, ok)
	core.AssertElementsMatch(t, []string{"claude", "gemini"}, capabilities.Available)
}

func TestPlatform_HandleSyncStatus_Good_LocalStateFallback(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
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
	core.RequireTrue(t, result.OK)

	status, ok := result.Value.(SyncStatusOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "2026-03-31T08:00:00Z", status.LastPushAt)
	core.AssertEqual(t, "2026-03-31T08:05:00Z", status.LastPullAt)
}
