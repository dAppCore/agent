// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

func writeFakeAgentBinary(t *testing.T, name string) {
	t.Helper()

	binDir := t.TempDir()
	binaryPath := core.JoinPath(binDir, name)
	script := "#!/bin/sh\nprintf 'done'\n"
	core.RequireTrue(t, fs.Write(binaryPath, script).OK)

	chmodResult := testCore.Process().Run(context.Background(), "chmod", "+x", binaryPath)
	core.RequireTrue(t, chmodResult.OK)
	t.Setenv("PATH", core.Concat(binDir, ":", core.Env("PATH")))
}

func TestCallBody_RemoteClient_ToolCallBody_Bad(t *testing.T) {
	client := NewRemoteClient("local")
	body := client.ToolCallBody(-1, "agentic_dispatch", map[string]any{})

	var payload map[string]any
	result := core.JSONUnmarshal(body, &payload)
	core.RequireTrue(t, result.OK)

	params, ok := payload["params"].(map[string]any)
	core.RequireTrue(t, ok)
	arguments, ok := params["arguments"].(map[string]any)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, float64(-1), payload["id"])
	core.AssertLen(t, arguments, 0)
}

func TestDuplicateRegistration_RegisterHTTPTransport_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEmpty(t, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	RegisterHTTPTransport(c)
	RegisterHTTPTransport(c)
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "remote"},
		core.Option{Key: "transport", Value: srv.URL},
	))

	streamResult := c.API().Stream("remote")
	core.RequireTrue(t, streamResult.OK)

	stream := streamResult.Value.(core.Stream)
	sendErr := stream.Send([]byte(`{"ping":1}`))
	core.RequireNoError(t, sendErr)

	response, receiveErr := stream.Receive()
	core.RequireNoError(t, receiveErr)
	core.AssertEqual(t, `{"status":"ok"}`, string(response))
}

func TestTransport_HTTPPost_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer post-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"title":"Fix tests"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPost(context.Background(), srv.URL, `{"title":"Fix tests"}`, "post-token", "Bearer")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"created":true}`, result.Value.(string))
}

func TestTransport_HTTPPatch_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "token patch-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"status":"done"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"updated":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPatch(context.Background(), srv.URL, `{"status":"done"}`, "patch-token", "token")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"updated":true}`, result.Value.(string))
}

func TestTransport_HTTPDelete_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodDelete, r.Method)
		core.AssertEqual(t, "Bearer delete-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"reason":"stale"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"deleted":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPDelete(context.Background(), srv.URL, `{"reason":"stale"}`, "delete-token", "Bearer")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"deleted":true}`, result.Value.(string))
}

func TestTransport_HTTPDo_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPut, r.Method)
		core.AssertEqual(t, "token do-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"value":7}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPDo(context.Background(), http.MethodPut, srv.URL, `{"value":7}`, "do-token", "token")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"ok":true}`, result.Value.(string))
}

func TestTransport_DrivePost_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/issues", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer drive-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"title":"Follow up"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"number":9}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DrivePost(c, "forge", "/issues", `{"title":"Follow up"}`, "Bearer")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"number":9}`, result.Value.(string))
}

func TestTransport_DriveDo_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/pulls/3", r.URL.Path)
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "token drive-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"state":"closed"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"closed":true}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveDo(c, "forge", http.MethodPatch, "/pulls/3", `{"state":"closed"}`, "token")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"closed":true}`, result.Value.(string))
}

func TestMissingDrive_DriveGet_Bad(t *testing.T) {
	c := core.New()
	result := DriveGet(c, "missing", "/repos/core/go-io", "Bearer")

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "drive not found")
}

func TestTransport_Stream_Send_Good_Case(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "token send-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"ping":1}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"pong":1}`))
	}))
	t.Cleanup(srv.Close)

	stream := &httpStream{Client: defaultClient, URL: srv.URL, Token: "send-token", Method: http.MethodPost}
	sendErr := stream.Send([]byte(`{"ping":1}`))
	core.RequireNoError(t, sendErr)
	core.AssertEqual(t, `{"pong":1}`, string(stream.Response))
}

func TestTransport_Stream_Receive_Good_Case(t *testing.T) {
	stream := &httpStream{Response: []byte(`{"cached":true}`)}
	response, err := stream.Receive()

	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"cached":true}`, string(response))
}

func TestTransport_Stream_Close_Good_Case(t *testing.T) {
	stream := &httpStream{}
	err := stream.Close()

	core.RequireNoError(t, err)
	core.AssertNil(t, stream.Response)
}

func TestClock_SyncRealClock_Now_Good(t *testing.T) {
	clock := remoteSyncRealClock{}
	before := time.Now()
	now := clock.Now()
	after := time.Now()

	core.AssertFalse(t, now.Before(before))
	core.AssertFalse(t, now.After(after))
}

func TestClock_SyncRealClock_After_Good(t *testing.T) {
	clock := remoteSyncRealClock{}
	start := time.Now()
	ch := clock.After(time.Millisecond)

	select {
	case firedAt := <-ch:
		core.AssertFalse(t, firedAt.Before(start))
	case <-time.After(time.Second):
		t.Fatal("expected remoteSyncRealClock.After to fire")
	}
}

func TestCluster_ClusterUnion_Find_Good(t *testing.T) {
	union := newQAClusterUnion(3)
	union.parent[1] = 0
	root := union.Find(1)

	core.AssertEqual(t, 0, root)
	core.AssertEqual(t, 0, union.parent[1])
}

func TestCluster_ClusterUnion_Union_Good(t *testing.T) {
	union := newQAClusterUnion(4)
	union.Union(0, 1)
	union.Union(1, 2)

	left := union.Find(0)
	right := union.Find(2)
	core.AssertEqual(t, left, right)
	core.AssertEqual(t, 3, union.size[left])
}

func TestYAML_ConcurrencyLimit_UnmarshalYAML_Good(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("total: 3\ngpt-5.4: 2\ngpt-5.3-codex-spark: 1\n"), &limit)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 3, limit.Total)
	core.AssertEqual(t, 2, limit.Models["gpt-5.4"])
	core.AssertEqual(t, 1, limit.Models["gpt-5.3-codex-spark"])
}

func TestRegistry_PrepSubsystem_Workspaces_Good(t *testing.T) {
	registry := core.NewRegistry[*WorkspaceStatus]()
	registry.Set("core/go-io/task-5", &WorkspaceStatus{Status: "queued"})

	subsystem := &PrepSubsystem{workspaces: registry}
	result := subsystem.Workspaces().Get("core/go-io/task-5")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "queued", result.Value.(*WorkspaceStatus).Status)
}

func TestMirrorsQueueState_PrepSubsystem_TrackWorkspace_Good(t *testing.T) {
	withStateStoreTempDir(t)

	subsystem := &PrepSubsystem{workspaces: core.NewRegistry[*WorkspaceStatus]()}
	defer subsystem.closeStateStore()

	status := &WorkspaceStatus{
		Status:    "queued",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
	}
	subsystem.TrackWorkspace("core/go-io/task-5", status)

	registryResult := subsystem.Workspaces().Get("core/go-io/task-5")
	core.RequireTrue(t, registryResult.OK)
	core.AssertEqual(t, "queued", registryResult.Value.(*WorkspaceStatus).Status)
	core.AssertEqual(t, 1, subsystem.stateStoreCount(stateQueueGroup))
}

func TestDispatchSync_PrepSubsystem_DispatchSync_Good_Case(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-9")
	subsystem := &PrepSubsystem{dispatchSyncTick: 10 * time.Millisecond}

	subsystem.dispatchSyncPrep = func(_ context.Context, _ *mcp.CallToolRequest, input PrepInput) (*mcp.CallToolResult, PrepOutput, error) {
		core.AssertEqual(t, "core", input.Org)
		core.AssertEqual(t, "go-io", input.Repo)
		core.AssertEqual(t, "codex", input.Agent)
		core.AssertEqual(t, "Fix tests", input.Task)
		core.AssertEqual(t, 9, input.Issue)

		core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
		core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
			Status: "completed",
			PRURL:  "https://forge.test/core/go-io/pulls/9",
		})).OK)

		return nil, PrepOutput{
			Success:      true,
			WorkspaceDir: workspaceDir,
			Branch:       "agent/fix-tests",
			Prompt:       "prompt",
		}, nil
	}
	subsystem.dispatchSyncSpawn = func(agent, prompt, workspaceDir string) (int, string, string, error) {
		core.AssertEqual(t, "codex", agent)
		core.AssertEqual(t, "prompt", prompt)
		core.AssertContains(t, workspaceDir, "task-9")
		return 42, "process-42", core.JoinPath(workspaceDir, ".meta", "agent.log"), nil
	}

	result := subsystem.DispatchSync(context.Background(), DispatchSyncInput{
		Org:   "core",
		Repo:  "go-io",
		Agent: "codex",
		Task:  "Fix tests",
		Issue: 9,
	})

	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "completed", result.Status)
	core.AssertEqual(t, "https://forge.test/core/go-io/pulls/9", result.PRURL)
}

func TestQueuedAgent_PrepSubsystem_SpawnFromQueue_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	writeFakeAgentBinary(t, "claude")

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-3")
	core.RequireTrue(t, fs.EnsureDir(WorkspaceRepoDir(workspaceDir)).OK)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceMetaDir(workspaceDir)).OK)

	subsystem := newPrepWithProcess()
	result := subsystem.SpawnFromQueue("claude", "Write docs", workspaceDir)
	core.RequireTrue(t, result.OK)

	pid, ok := result.Value.(int)
	core.RequireTrue(t, ok)
	if pid <= 0 {
		t.Fatalf("expected positive pid, got %d", pid)
	}

	outputPath := agentOutputFile(workspaceDir, "claude")
	requireEventually(t, func() bool { return fs.IsFile(outputPath).OK }, 5*time.Second, 10*time.Millisecond)
	outputResult := fs.Read(outputPath)
	core.RequireTrue(t, outputResult.OK)
	output := core.Trim(outputResult.Value.(string))
	core.AssertEqual(t, "done", output)
}

func TestMultipleRevisions_PrepSubsystem_ScheduleRevision_Ugly(t *testing.T) {
	withStateStoreTempDir(t)

	firstTime := time.Date(2026, time.April, 26, 12, 0, 0, 0, time.UTC)
	restoreContentSEONow(t, firstTime)
	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()

	first, firstErr := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "First copy")
	core.RequireNoError(t, firstErr)

	secondTime := firstTime.Add(time.Minute)
	restoreContentSEONow(t, secondTime)
	second, secondErr := subsystem.ScheduleRevision(context.Background(), "/help/hosting", "Second copy")
	core.RequireNoError(t, secondErr)

	pending, pendingErr := subsystem.GetPendingRevisions("/help/hosting")
	core.RequireNoError(t, pendingErr)
	core.AssertLen(t, pending, 2)
	core.AssertNotEqual(t, first.CreatedAt, second.CreatedAt)
	core.AssertContains(t, []string{pending[0].Content, pending[1].Content}, "First copy")
	core.AssertContains(t, []string{pending[0].Content, pending[1].Content}, "Second copy")
}

func TestMetaReader_ForgeMetaReader_GetIssueState_Good(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[7] = &pipelineTestIssue{
		Number: 7,
		Title:  "Fix the flaky tests",
		Body:   "Body",
		State:  "closed",
		Labels: []string{"bug", "agentic"},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	subsystem, _ := testPrepWithCore(t, srv)
	reader := newPipelineForgeMetaReader(subsystem, "core")
	state, err := reader.GetIssueState(context.Background(), "go-io", 7)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 7, state.Number)
	core.AssertEqual(t, "closed", state.State)
	core.AssertEqual(t, "Fix the flaky tests", state.Title)
	core.AssertContains(t, state.Labels, "bug")
}

func TestMetaReader_ForgeMetaReader_GetPRMeta_Good(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Pulls[12] = &pipelineTestPR{
		Number:                12,
		Title:                 "Stabilise pipeline",
		State:                 "open",
		Mergeable:             boolPtr(true),
		MergeableState:        "clean",
		HeadRef:               "agent/stabilise-pipeline",
		HeadSHA:               "sha-12",
		BaseRef:               "dev",
		ReviewThreadsTotal:    3,
		ReviewThreadsResolved: 2,
		Statuses: []map[string]any{
			{"context": "qa", "status": "success"},
			{"name": "build", "conclusion": "failure"},
		},
		Reactions: map[string]int{"eyes": 1},
	}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	subsystem, _ := testPrepWithCore(t, srv)
	reader := newPipelineForgeMetaReader(subsystem, "core")
	meta, err := reader.GetPRMeta(context.Background(), "go-io", 12)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 12, meta.Number)
	core.AssertEqual(t, "mergeable", meta.Mergeable)
	core.AssertEqual(t, "agent/stabilise-pipeline", meta.HeadBranch)
	core.AssertLen(t, meta.Checks, 2)
	core.AssertEqual(t, 3, meta.ThreadsTotal)
	core.AssertEqual(t, 2, meta.ThreadsResolved)
	core.AssertTrue(t, meta.HasEyesReaction)
}

func TestMetaReader_ForgeMetaReader_GetEpicMeta_Good(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[1] = &pipelineTestIssue{
		Number: 1,
		Title:  "Epic",
		State:  "open",
		Body:   "Epic branch: `agent/epic`\n- [x] #2 Done child\n- [ ] #3 Open child",
	}
	repo.Issues[2] = &pipelineTestIssue{Number: 2, Title: "Done child", State: "closed"}
	repo.Issues[3] = &pipelineTestIssue{Number: 3, Title: "Open child", State: "open"}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})

	subsystem, _ := testPrepWithCore(t, srv)
	reader := newPipelineForgeMetaReader(subsystem, "core")
	meta, err := reader.GetEpicMeta(context.Background(), "go-io", 1)

	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, meta.Number)
	core.AssertEqual(t, "agent/epic", meta.Branch)
	core.AssertLen(t, meta.Children, 2)
	core.AssertTrue(t, meta.Children[0].Checked)
	core.AssertEqual(t, "closed", meta.Children[0].State)
	core.AssertEqual(t, "open", meta.Children[1].State)
}

func TestMetaReader_ForgeMetaReader_GetCommentReactions_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/api/v1/repos/core/go-io/issues/comments/55/reactions", r.URL.Path)
		core.AssertEqual(t, "token test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`[{"content":"eyes"},{"content":"eyes"},{"content":"rocket"}]`))
	}))
	t.Cleanup(srv.Close)

	subsystem, _ := testPrepWithCore(t, srv)
	reader := newPipelineForgeMetaReader(subsystem, "core")
	reactions, err := reader.GetCommentReactions(context.Background(), "go-io", 55)

	core.RequireNoError(t, err)
	core.AssertLen(t, reactions, 2)

	counts := map[string]int{}
	for _, reaction := range reactions {
		counts[reaction.Content] = reaction.Count
	}
	core.AssertEqual(t, 2, counts["eyes"])
	core.AssertEqual(t, 1, counts["rocket"])
}
