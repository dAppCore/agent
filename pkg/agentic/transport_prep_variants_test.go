// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

func registerToolNames(t *testing.T, svc *coremcp.Service) []string {
	t.Helper()

	server := svc.Server()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	core.RequireNoError(t, err)

	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func TestServerError_HTTPGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"get upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPGet(context.Background(), srv.URL, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"get upstream"}`, result.Value.(string))
}

func TestInvalidURL_HTTPPost_Bad(t *testing.T) {
	result := HTTPPost(context.Background(), "://bad", `{"title":"Fix tests"}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestServerError_HTTPPost_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"post upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPost(context.Background(), srv.URL, `{"title":"Fix tests"}`, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"post upstream"}`, result.Value.(string))
}

func TestInvalidURL_HTTPPatch_Bad(t *testing.T) {
	result := HTTPPatch(context.Background(), "://bad", `{"status":"done"}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestServerError_HTTPPatch_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"patch upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPatch(context.Background(), srv.URL, `{"status":"done"}`, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"patch upstream"}`, result.Value.(string))
}

func TestInvalidURL_HTTPDelete_Bad(t *testing.T) {
	result := HTTPDelete(context.Background(), "://bad", `{"reason":"stale"}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestInvalidURL_HTTPDo_Bad(t *testing.T) {
	result := HTTPDo(context.Background(), http.MethodPut, "://bad", `{"value":7}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestServerError_HTTPDo_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`{"error":"put upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPDo(context.Background(), http.MethodPut, srv.URL, `{"value":7}`, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"put upstream"}`, result.Value.(string))
}

func TestServerError_DriveGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"drive get upstream"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveGet(c, "forge", "/repos/core/go-io", "Bearer")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"drive get upstream"}`, result.Value.(string))
}

func TestMissingDrive_DrivePost_Bad(t *testing.T) {
	result := DrivePost(core.New(), "missing", "/issues", `{"title":"Follow up"}`, "Bearer")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "drive not found")
}

func TestServerError_DrivePost_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"drive post upstream"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DrivePost(c, "forge", "/issues", `{"title":"Follow up"}`, "Bearer")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"drive post upstream"}`, result.Value.(string))
}

func TestServerError_DriveDo_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"drive do upstream"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveDo(c, "forge", http.MethodPatch, "/pulls/3", `{"state":"closed"}`, "token")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"drive do upstream"}`, result.Value.(string))
}

func TestInvalidURL_Stream_Send_Bad(t *testing.T) {
	stream := &httpStream{client: defaultClient, url: "://bad", method: http.MethodPost}
	sendErr := stream.Send([]byte(`{"ping":1}`))

	core.AssertError(t, sendErr)
	core.AssertContains(t, sendErr.Error(), "missing protocol scheme")
}

func TestNilClient_Stream_Send_Ugly(t *testing.T) {
	stream := &httpStream{url: "http://example.com", method: http.MethodPost}
	core.AssertPanics(t, func() {
		_ = stream.Send([]byte(`{"ping":1}`))
	})
}

func TestEmptyBuffer_Stream_Receive_Bad(t *testing.T) {
	stream := &httpStream{}
	response, err := stream.Receive()

	core.RequireNoError(t, err)
	core.AssertNil(t, response)
}

func TestNilReceiver_Stream_Receive_Ugly(t *testing.T) {
	var stream *httpStream
	core.AssertPanics(t, func() {
		_, _ = stream.Receive()
	})
}

func TestZeroValue_Stream_Close_Bad(t *testing.T) {
	stream := &httpStream{}
	err := stream.Close()

	core.RequireNoError(t, err)
	core.AssertNil(t, stream.client)
}

func TestNilReceiver_Stream_Close_Ugly(t *testing.T) {
	var stream *httpStream
	err := stream.Close()
	core.AssertNoError(t, err)
	core.AssertNil(t, stream)
}

func TestNonZero_SyncRealClock_Now_Bad(t *testing.T) {
	clock := remoteSyncRealClock{}
	now := clock.Now()

	core.AssertFalse(t, now.IsZero())
	core.AssertEqual(t, now.Location(), time.Now().Location())
}

func TestOrderedCalls_SyncRealClock_Now_Ugly(t *testing.T) {
	clock := remoteSyncRealClock{}
	first := clock.Now()
	second := clock.Now()

	core.AssertFalse(t, second.Before(first))
	core.AssertFalse(t, first.IsZero())
}

func TestImmediate_SyncRealClock_After_Bad(t *testing.T) {
	clock := remoteSyncRealClock{}
	ch := clock.After(0)

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected immediate After(0) notification")
	}
}

func TestNegativeDelay_SyncRealClock_After_Ugly(t *testing.T) {
	clock := remoteSyncRealClock{}
	ch := clock.After(-time.Millisecond)

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected After on negative delay to fire")
	}
}

func TestRoot_ClusterUnion_Find_Bad(t *testing.T) {
	union := newQAClusterUnion(2)
	root := union.Find(0)

	core.AssertEqual(t, 0, root)
	core.AssertEqual(t, 0, union.parent[0])
}

func TestCompression_ClusterUnion_Find_Ugly(t *testing.T) {
	union := newQAClusterUnion(4)
	union.parent[3] = 2
	union.parent[2] = 1
	union.parent[1] = 0
	root := union.Find(3)

	core.AssertEqual(t, 0, root)
	core.AssertEqual(t, 0, union.parent[3])
}

func TestDuplicate_ClusterUnion_Union_Bad(t *testing.T) {
	union := newQAClusterUnion(2)
	union.Union(0, 0)

	core.AssertEqual(t, 0, union.Find(0))
	core.AssertEqual(t, 1, union.size[0])
}

func TestConnected_ClusterUnion_Union_Ugly(t *testing.T) {
	union := newQAClusterUnion(3)
	union.Union(0, 1)
	union.Union(1, 2)
	union.Union(0, 2)

	root := union.Find(0)
	core.AssertEqual(t, root, union.Find(2))
	core.AssertEqual(t, 3, union.size[root])
}

func TestScalar_ConcurrencyLimit_UnmarshalYAML_Bad(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("2\n"), &limit)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 2, limit.Total)
	core.AssertNil(t, limit.Models)
}

func TestInvalid_ConcurrencyLimit_UnmarshalYAML_Ugly(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("total: nope\n"), &limit)

	core.AssertError(t, err)
	core.AssertEqual(t, 0, limit.Total)
}

func TestNilRegistry_PrepSubsystem_Workspaces_Bad(t *testing.T) {
	subsystem := &PrepSubsystem{}
	workspaces := subsystem.Workspaces()

	core.AssertNil(t, workspaces)
	core.AssertEqual(t, (*core.Registry[*WorkspaceStatus])(nil), workspaces)
}

func TestLiveRegistry_PrepSubsystem_Workspaces_Ugly(t *testing.T) {
	subsystem := &PrepSubsystem{workspaces: core.NewRegistry[*WorkspaceStatus]()}
	workspaces := subsystem.Workspaces()
	workspaces.Set("core/go-store/task-2", &WorkspaceStatus{Status: "running"})

	result := subsystem.workspaces.Get("core/go-store/task-2")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "running", result.Value.(*WorkspaceStatus).Status)
}

func TestNilStatus_PrepSubsystem_TrackWorkspace_Bad(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{workspaces: core.NewRegistry[*WorkspaceStatus]()}
	defer subsystem.closeStateStore()
	subsystem.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{Status: "queued", Agent: "codex", Repo: "go-io", Branch: "agent/fix"})
	subsystem.TrackWorkspace("core/go-io/task-5", nil)

	core.AssertEqual(t, 0, subsystem.stateStoreCount(stateQueueGroup))
	core.AssertFalse(t, subsystem.Workspaces().Get("core/go-io/task-5").OK)
}

func TestConcurrencySnapshot_PrepSubsystem_TrackWorkspace_Ugly(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{workspaces: core.NewRegistry[*WorkspaceStatus]()}
	defer subsystem.closeStateStore()
	subsystem.TrackWorkspace("core/go-io/task-5", &WorkspaceStatus{Status: "running", Agent: "codex:gpt-5.4", Repo: "go-io"})

	value, ok := subsystem.stateStoreGet(stateConcurrencyGroup, "codex")
	core.RequireTrue(t, ok)
	core.AssertContains(t, value, "\"running\":1")
}

func TestBaseOnly_PrepSubsystem_RegisterTools_Bad(t *testing.T) {
	t.Setenv("CORE_MCP_FULL", "")
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(svc)
	toolNames := registerToolNames(t, svc)

	core.AssertContains(t, toolNames, "agentic_prep_workspace")
	core.AssertNotContains(t, toolNames, "agentic_session_start")
}

func TestRepeatedCall_PrepSubsystem_RegisterTools_Ugly(t *testing.T) {
	t.Setenv("CORE_MCP_FULL", "1")
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(svc)
	subsystem.RegisterTools(svc)
	toolNames := registerToolNames(t, svc)

	core.AssertContains(t, toolNames, "agentic_prep_workspace")
	core.AssertContains(t, toolNames, "agentic_session_start")
}

func TestUnknownMessage_PrepSubsystem_HandleIPCEvents_Bad(t *testing.T) {
	c, subsystem := newCoreForHandlerTests(t)
	result := subsystem.HandleIPCEvents(c, messages.PokeQueue{})

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestMissingWorkspace_PrepSubsystem_HandleIPCEvents_Ugly(t *testing.T) {
	c, subsystem := newCoreForHandlerTests(t)
	result := subsystem.HandleIPCEvents(c, messages.SpawnQueued{Workspace: "missing", Agent: "claude", Task: "Write docs"})

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestMissingToken_PrepSubsystem_Connect_Bad(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.Connect(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestCancelled_PrepSubsystem_Connect_Ugly(t *testing.T) {
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

func TestMissingToken_PrepSubsystem_PollFallback_Bad(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.PollFallback(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestCancelled_PrepSubsystem_PollFallback_Ugly(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "secret-token")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := subsystem.PollFallback(ctx, core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestMissingToken_PrepSubsystem_Heartbeat_Bad(t *testing.T) {
	resetFleetRuntimeState()
	t.Cleanup(resetFleetRuntimeState)

	subsystem := testPrepWithPlatformServer(t, nil, "")
	result := subsystem.Heartbeat(context.Background(), core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestDisabledInterval_PrepSubsystem_Heartbeat_Ugly(t *testing.T) {
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

func TestPrepFailure_PrepSubsystem_DispatchSync_Bad(t *testing.T) {
	subsystem := &PrepSubsystem{}
	subsystem.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		return nil, PrepOutput{}, core.E("prepWorkspace", "boom", nil)
	}

	result := subsystem.DispatchSync(context.Background(), DispatchSyncInput{Repo: "go-io", Task: "Fix tests"})
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Error)
	core.AssertContains(t, result.Error.Error(), "prep workspace failed")
}

func TestSpawnFailure_PrepSubsystem_DispatchSync_Ugly(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-10")
	subsystem := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}
	subsystem.dispatchSyncPrep = func(context.Context, *mcpsdk.CallToolRequest, PrepInput) (*mcpsdk.CallToolResult, PrepOutput, error) {
		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Status: "running"})).OK)
		return nil, PrepOutput{Success: true, WorkspaceDir: workspaceDir, Prompt: "prompt"}, nil
	}
	subsystem.dispatchSyncSpawn = func(string, string, string) (int, string, string, error) {
		return 0, "", "", core.E("spawn", "boom", nil)
	}

	result := subsystem.DispatchSync(context.Background(), DispatchSyncInput{Repo: "go-io", Agent: "codex", Task: "Fix tests"})
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Error)
	core.AssertContains(t, result.Error.Error(), "spawn agent failed")
}

func TestUnknownAgent_PrepSubsystem_SpawnFromQueue_Bad(t *testing.T) {
	subsystem := newPrepWithProcess()
	result := subsystem.SpawnFromQueue("robot-from-the-future", "Write docs", t.TempDir())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "unknown agent")
}

func TestMissingProcess_PrepSubsystem_SpawnFromQueue_Ugly(t *testing.T) {
	writeFakeAgentBinary(t, "claude")

	c := core.New()
	subsystem := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	result := subsystem.SpawnFromQueue("claude", "Write docs", t.TempDir())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "process service not registered")
}

func TestNilArgs_RegisterHandlers_Bad(t *testing.T) {
	core.AssertNotPanics(t, func() {
		RegisterHandlers(nil, nil)
	})
}

func TestRepeated_RegisterHandlers_Ugly(t *testing.T) {
	c := core.New()
	c.Action("runner.poke", func(context.Context, core.Options) core.Result { return core.Result{OK: true} })

	RegisterHandlers(c, &PrepSubsystem{})
	RegisterHandlers(c, &PrepSubsystem{})
	core.AssertNotPanics(t, func() {
		c.ACTION(messages.AgentCompleted{})
	})
}

func TestMissingIssue_ForgeMetaReader_GetIssueState_Bad(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetIssueState(context.Background(), "go-io", 77)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "issue")
}

func TestDefaultState_ForgeMetaReader_GetIssueState_Ugly(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[8] = &pipelineTestIssue{Number: 8, Title: "Untitled", State: ""}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	state, err := reader.GetIssueState(context.Background(), "go-io", 8)

	core.RequireNoError(t, err)
	core.AssertEqual(t, "open", state.State)
	core.AssertEqual(t, "Untitled", state.Title)
}

func TestMissingPR_ForgeMetaReader_GetPRMeta_Bad(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetPRMeta(context.Background(), "go-io", 44)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read PR")
}

func TestInvalidStatusPayload_ForgeMetaReader_GetPRMeta_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/core/go-io/pulls/12":
			_, _ = w.Write([]byte(`{"number":12,"state":"open","head":{"ref":"agent/fix","sha":"sha-12","repo":{"updated_at":"2026-04-25T12:00:00Z","pushed_at":"2026-04-25T12:00:00Z"}},"base":{"ref":"dev"},"reactions":{"eyes":0}}`))
		case "/api/v1/repos/core/go-io/commits/sha-12/status":
			_, _ = w.Write([]byte(`not-json`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	meta, err := reader.GetPRMeta(context.Background(), "go-io", 12)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 12, meta.Number)
	core.AssertLen(t, meta.Checks, 0)
}

func TestMissingEpic_ForgeMetaReader_GetEpicMeta_Bad(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetEpicMeta(context.Background(), "go-io", 1)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "issue")
}

func TestNoChildren_ForgeMetaReader_GetEpicMeta_Ugly(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{Number: 1, Title: "Epic", State: "open", Body: "plain body"}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	meta, err := reader.GetEpicMeta(context.Background(), "go-io", 1)

	core.RequireNoError(t, err)
	core.AssertEmpty(t, meta.Branch)
	core.AssertLen(t, meta.Children, 0)
}

func TestMissingComment_ForgeMetaReader_GetCommentReactions_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetCommentReactions(context.Background(), "go-io", 55)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read comment reactions")
}

func TestInvalidPayload_ForgeMetaReader_GetCommentReactions_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := &pipelineForgeMetaReader{subsystem: subsystem, org: "core"}
	_, err := reader.GetCommentReactions(context.Background(), "go-io", 55)

	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to decode reactions")
}
