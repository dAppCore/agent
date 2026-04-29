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

func TestSync_HandleSyncPush_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/sync", r.URL.Path)
		core.AssertEqual(t, "Bearer secret-token", r.Header.Get("Authorization"))
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, AgentName(), payload["agent_id"])
		dispatches, ok := payload["dispatches"].([]any)
		core.RequireTrue(t, ok)
		core.AssertLen(t, dispatches, 1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertFalse(t, readSyncStatusState().LastPushAt.IsZero())
}

func TestSync_HandleSyncPush_Good_UsesProvidedDispatches(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/sync", r.URL.Path)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "charon", payload["agent_id"])

		dispatches, ok := payload["dispatches"].([]any)
		core.RequireTrue(t, ok)
		core.AssertLen(t, dispatches, 1)

		record, ok := dispatches[0].(map[string]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, "external-1", record["workspace"])
		core.AssertEqual(t, "completed", record["status"])

		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPushInput(context.Background(), SyncPushInput{
		AgentID: "charon",
		Dispatches: []map[string]any{
			{"workspace": "external-1", "status": "completed"},
		},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertEmpty(t, readSyncQueue())
}

func TestSync_HandleSyncPush_Bad_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "")

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
	}
	output, err := subsystem.syncPush(context.Background(), "")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 0, output.Count)
	core.AssertEmpty(t, readSyncQueue())
}

func TestSync_HandleSyncPush_Bad_QueuesProvidedDispatchesWhenOffline(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "")

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
	}
	output, err := subsystem.syncPushInput(context.Background(), SyncPushInput{
		AgentID: "charon",
		Dispatches: []map[string]any{
			{"workspace": "external-1", "status": "completed"},
		},
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 0, output.Count)

	queued := readSyncQueue()
	core.AssertLen(t, queued, 1)
	core.AssertEqual(t, "charon", queued[0].AgentID)
	core.AssertLen(t, queued[0].Dispatches, 1)
	core.AssertEqual(t, "external-1", queued[0].Dispatches[0]["workspace"])
}

func TestSync_HandleSyncPush_Ugly_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/sync", r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"unavailable"}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 0, output.Count)

	queued := readSyncQueue()
	core.AssertLen(t, queued, 1)
	core.AssertEqual(t, AgentName(), queued[0].AgentID)
	core.AssertLen(t, queued[0].Dispatches, 1)
}

func TestSync_HandleSyncPull_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/context", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"mem-1","content":"Known pattern"}]}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPull(context.Background(), "codex")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertLen(t, output.Context, 1)
	core.AssertEqual(t, "mem-1", output.Context[0]["id"])

	cached := readSyncContext()
	core.AssertLen(t, cached, 1)
	core.AssertEqual(t, "mem-1", cached[0]["id"])
	core.AssertFalse(t, readSyncStatusState().LastPullAt.IsZero())
}

func TestSync_HandleSyncPull_Good_SinceQuery(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/context", r.URL.Path)
		core.AssertEqual(t, "codex", r.URL.Query().Get("agent_id"))
		core.AssertEqual(t, "2026-03-30T00:00:00Z", r.URL.Query().Get("since"))
		_, _ = w.Write([]byte(`{"data":[{"id":"mem-2","content":"Recent pattern"}]}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPullInput(context.Background(), SyncPullInput{
		AgentID: "codex",
		Since:   "2026-03-30T00:00:00Z",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertLen(t, output.Context, 1)
	core.AssertEqual(t, "mem-2", output.Context[0]["id"])
}

func TestSync_RecordSyncHistory_Good_Case(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	recordSyncHistory("push", "codex", 7, 256, 3, now)
	recordSyncHistory("pull", "codex", 7, 128, 1, now.Add(5*time.Minute))

	records := readSyncRecords()
	core.AssertLen(t, records, 2)
	core.AssertEqual(t, "codex", records[0].AgentID)
	core.AssertEqual(t, 7, records[0].FleetNodeID)
	core.AssertEqual(t, "push", records[0].Direction)
	core.AssertEqual(t, 256, records[0].PayloadSize)
	core.AssertEqual(t, 3, records[0].ItemsCount)
	core.AssertEqual(t, "2026-03-31T12:00:00Z", records[0].SyncedAt)
	core.AssertEqual(t, "pull", records[1].Direction)
	core.AssertEqual(t, 1, records[1].ItemsCount)
}

func TestSync_RecordSyncHistory_Good_FleetNodeID(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	recordSyncHistory("push", "charon", 42, 512, 2, now)

	records := readSyncRecords()
	core.AssertLen(t, records, 1)
	core.AssertEqual(t, 42, records[0].FleetNodeID)
	core.AssertEqual(t, "charon", records[0].AgentID)
}

func TestSync_RecordSyncHistory_Bad_MissingFile(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	records := readSyncRecords()
	core.AssertEmpty(t, records)

	recordSyncHistory("", "codex", 0, 64, 1, time.Now())
	records = readSyncRecords()
	core.AssertEmpty(t, records)
}

func TestSync_RecordSyncHistory_Ugly_CorruptFile(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	core.RequireTrue(t, fs.WriteAtomic(syncRecordsPath(), "{not-json").OK)

	records := readSyncRecords()
	core.AssertEmpty(t, records)
}

func TestSync_HandleSyncPush_Good_ReportMetadata(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(WorkspaceMetaDir(workspaceDir))
	core.RequireTrue(t, fs.Write(core.JoinPath(WorkspaceMetaDir(workspaceDir), "report.json"), `{"findings":[{"file":"main.go"}],"changes":{"files_changed":1}}`).OK)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "blocked",
		Agent:     "codex",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		Issue:     42,
		Question:  "Which API version?",
		ProcessID: "proc-1",
		Runs:      2,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)

		dispatches, ok := payload["dispatches"].([]any)
		core.RequireTrue(t, ok)
		core.AssertLen(t, dispatches, 1)

		record, ok := dispatches[0].(map[string]any)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, "Which API version?", record["question"])
		core.AssertEqual(t, float64(42), record["issue"])
		core.AssertEqual(t, float64(2), record["runs"])
		core.AssertEqual(t, "proc-1", record["process_id"])
		core.AssertNotNil(t, record["report"])
		core.AssertNotNil(t, record["findings"])
		core.AssertNotNil(t, record["changes"])

		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
}

func TestSync_ReadSyncWorkspaceReport_Ugly_CorruptJSONPreservesArtifact(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	metaDir := WorkspaceMetaDir(workspaceDir)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)

	reportPath := core.JoinPath(metaDir, "report.json")
	core.RequireTrue(t, fs.Write(reportPath, `{"findings":[{"file":"main.go"}],"changes":`).OK)

	report := readSyncWorkspaceReport(workspaceDir)
	core.AssertNil(t, report)
	core.AssertFalse(t, fs.Exists(reportPath))

	entries := listDirNames(fs.List(metaDir))
	core.AssertLen(t, entries, 1)
	core.AssertTrue(t, core.HasPrefix(entries[0], "report.json.corrupt-"))
}

func TestSync_HandleSyncPull_Good_NestedEnvelope(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"context":[{"id":"ctx-1","content":"Known pattern"}]}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPull(context.Background(), "codex")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertLen(t, output.Context, 1)
	core.AssertEqual(t, "ctx-1", output.Context[0]["id"])
}

func TestSync_HandleSyncPull_Bad_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")
	writeSyncContext([]map[string]any{
		{"id": "cached-1", "content": "Cached context"},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/context", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPull(context.Background(), "codex")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertLen(t, output.Context, 1)
	core.AssertEqual(t, "cached-1", output.Context[0]["id"])
}

func TestSync_HandleSyncPull_Ugly_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")
	writeSyncContext([]map[string]any{
		{"id": "cached-2", "content": "Fallback context"},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/context", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{this is not json`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPull(context.Background(), "codex")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 1, output.Count)
	core.AssertLen(t, output.Context, 1)
	core.AssertEqual(t, "cached-2", output.Context[0]["id"])
}

// schedule := syncBackoffSchedule(3) // 15s
func TestSync_SyncBackoffSchedule_Good_Case(t *testing.T) {
	core.AssertEqual(t, time.Duration(0), syncBackoffSchedule(0))
	core.AssertEqual(t, time.Second, syncBackoffSchedule(1))
	core.AssertEqual(t, 5*time.Second, syncBackoffSchedule(2))
	core.AssertEqual(t, 15*time.Second, syncBackoffSchedule(3))
	core.AssertEqual(t, 60*time.Second, syncBackoffSchedule(4))
	core.AssertEqual(t, 5*time.Minute, syncBackoffSchedule(5))
	core.AssertEqual(t, 5*time.Minute, syncBackoffSchedule(100))
}

func TestSync_SyncBackoffSchedule_Bad_NegativeAttempts(t *testing.T) {
	first := syncBackoffSchedule(-1)
	second := syncBackoffSchedule(-5)
	core.AssertEqual(t, time.Duration(0), first)
	core.AssertEqual(t, time.Duration(0), second)
}

func TestSync_HandleSyncPush_Ugly_IncrementsBackoffOnFailure(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}

	// First failure — attempt 1, backoff 1s
	_, err := subsystem.syncPushInput(context.Background(), SyncPushInput{
		AgentID:    "charon",
		Dispatches: []map[string]any{{"workspace": "w-1", "status": "completed"}},
	})
	core.RequireNoError(t, err)
	queued := readSyncQueue()
	core.AssertLen(t, queued, 1)
	core.AssertEqual(t, 1, queued[0].Attempts)
	core.AssertFalse(t, queued[0].NextAttempt.IsZero())
	core.AssertTrue(t, queued[0].NextAttempt.After(time.Now()))
	core.AssertTrue(t, queued[0].NextAttempt.Before(time.Now().Add(2*time.Second)))
}

func TestSync_RunSyncFlushLoop_Good_DrainsQueuedPushes(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	writeSyncQueue([]syncQueuedPush{{
		AgentID:    "charon",
		Dispatches: []map[string]any{{"workspace": "w-1", "status": "completed"}},
		QueuedAt:   time.Now().Add(-1 * time.Minute),
	}})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/agent/sync", r.URL.Path)
		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go subsystem.runSyncFlushLoop(ctx, 10*time.Millisecond)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if len(readSyncQueue()) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("sync flush loop did not drain queue: %v", readSyncQueue())
}

func TestSync_CollectSyncDispatches_Good_SkipsAlreadySynced(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)
	updatedAt := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Repo:      "go-io",
		Org:       "core",
		Runs:      1,
		UpdatedAt: updatedAt,
	})

	// First scan picks it up.
	first := collectSyncDispatches()
	core.AssertLen(t, first, 1)

	// Mark as synced — next scan skips it.
	markDispatchesSynced(first)
	second := collectSyncDispatches()
	core.AssertEmpty(t, second)

	// When the workspace gets a new run, fingerprint changes → rescan.
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Repo:      "go-io",
		Org:       "core",
		Runs:      2,
		UpdatedAt: updatedAt.Add(time.Hour),
	})
	third := collectSyncDispatches()
	core.AssertLen(t, third, 1)
}

func TestSync_SyncPushInput_Good_QueueOnlySkipsWorkspaceScan(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	// Seed a completed workspace that would normally be picked up by scan.
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(workspaceDir)
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}

	// With an empty queue and no scan, nothing to push.
	output, err := subsystem.syncPushInput(context.Background(), SyncPushInput{QueueOnly: true})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 0, output.Count)
	core.AssertEqual(t, 0, called)
	core.AssertEmpty(t, readSyncQueue())
}

func TestSync_RunSyncFlushLoop_Bad_NoopWithoutToken(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "")

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Should return immediately, no goroutine leak.
	subsystem.runSyncFlushLoop(ctx, 10*time.Millisecond)
}

func TestSync_HandleSyncPush_Ugly_RespectsBackoffWindow(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	// Prime queue with a push that's still inside its backoff window
	writeSyncQueue([]syncQueuedPush{{
		AgentID:     "charon",
		Dispatches:  []map[string]any{{"workspace": "w-1", "status": "completed"}},
		QueuedAt:    time.Now().Add(-2 * time.Minute),
		Attempts:    3,
		NextAttempt: time.Now().Add(5 * time.Minute),
	}})

	called := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, 0, output.Count)
	core.AssertEqual(t, 0, called, "backoff must skip the HTTP call")

	queued := readSyncQueue()
	core.AssertLen(t, queued, 1)
	core.AssertEqual(t, 3, queued[0].Attempts)
}

func TestSync_syncBackoffSchedule_Good_Case(t *testing.T) {
	core.AssertEqual(t, time.Second, syncBackoffSchedule(1))
	core.AssertEqual(t, 5*time.Second, syncBackoffSchedule(2))
	core.AssertEqual(t, 15*time.Second, syncBackoffSchedule(3))
	core.AssertEqual(t, 60*time.Second, syncBackoffSchedule(4))
}

func TestSync_syncBackoffSchedule_Bad_Case(t *testing.T) {
	first := syncBackoffSchedule(-1)
	second := syncBackoffSchedule(-42)
	core.AssertEqual(t, time.Duration(0), first)
	core.AssertEqual(t, time.Duration(0), second)
}

func TestSync_syncBackoffSchedule_Ugly_Case(t *testing.T) {
	core.AssertEqual(t, time.Duration(0), syncBackoffSchedule(0))
	core.AssertEqual(t, 5*time.Minute, syncBackoffSchedule(5))
	core.AssertEqual(t, 5*time.Minute, syncBackoffSchedule(999))
}
