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

func TestSync_HandleSyncPush_Good(t *testing.T) {
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
		require.Equal(t, "/v1/agent/sync", r.URL.Path)
		require.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, AgentName(), payload["agent_id"])
		dispatches, ok := payload["dispatches"].([]any)
		require.True(t, ok)
		require.Len(t, dispatches, 1)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	assert.False(t, readSyncStatusState().LastPushAt.IsZero())
}

func TestSync_HandleSyncPush_Good_UsesProvidedDispatches(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/sync", r.URL.Path)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		require.Equal(t, "charon", payload["agent_id"])

		dispatches, ok := payload["dispatches"].([]any)
		require.True(t, ok)
		require.Len(t, dispatches, 1)

		record, ok := dispatches[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "external-1", record["workspace"])
		require.Equal(t, "completed", record["status"])

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
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	assert.Empty(t, readSyncQueue())
}

func TestSync_HandleSyncPush_Bad(t *testing.T) {
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
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 0, output.Count)
	assert.Empty(t, readSyncQueue())
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
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 0, output.Count)

	queued := readSyncQueue()
	require.Len(t, queued, 1)
	assert.Equal(t, "charon", queued[0].AgentID)
	require.Len(t, queued[0].Dispatches, 1)
	assert.Equal(t, "external-1", queued[0].Dispatches[0]["workspace"])
}

func TestSync_HandleSyncPush_Ugly(t *testing.T) {
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
		require.Equal(t, "/v1/agent/sync", r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"unavailable"}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 0, output.Count)

	queued := readSyncQueue()
	require.Len(t, queued, 1)
	assert.Equal(t, AgentName(), queued[0].AgentID)
	require.Len(t, queued[0].Dispatches, 1)
}

func TestSync_HandleSyncPull_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/context", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"mem-1","content":"Known pattern"}]}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPull(context.Background(), "codex")
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	require.Len(t, output.Context, 1)
	assert.Equal(t, "mem-1", output.Context[0]["id"])

	cached := readSyncContext()
	require.Len(t, cached, 1)
	assert.Equal(t, "mem-1", cached[0]["id"])
	assert.False(t, readSyncStatusState().LastPullAt.IsZero())
}

func TestSync_HandleSyncPull_Good_SinceQuery(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/context", r.URL.Path)
		require.Equal(t, "codex", r.URL.Query().Get("agent_id"))
		require.Equal(t, "2026-03-30T00:00:00Z", r.URL.Query().Get("since"))
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
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	require.Len(t, output.Context, 1)
	assert.Equal(t, "mem-2", output.Context[0]["id"])
}

func TestSync_RecordSyncHistory_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	recordSyncHistory("push", "codex", 7, 256, 3, now)
	recordSyncHistory("pull", "codex", 7, 128, 1, now.Add(5*time.Minute))

	records := readSyncRecords()
	require.Len(t, records, 2)
	assert.Equal(t, "codex", records[0].AgentID)
	assert.Equal(t, 7, records[0].FleetNodeID)
	assert.Equal(t, "push", records[0].Direction)
	assert.Equal(t, 256, records[0].PayloadSize)
	assert.Equal(t, 3, records[0].ItemsCount)
	assert.Equal(t, "2026-03-31T12:00:00Z", records[0].SyncedAt)
	assert.Equal(t, "pull", records[1].Direction)
	assert.Equal(t, 1, records[1].ItemsCount)
}

func TestSync_RecordSyncHistory_Good_FleetNodeID(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	recordSyncHistory("push", "charon", 42, 512, 2, now)

	records := readSyncRecords()
	require.Len(t, records, 1)
	assert.Equal(t, 42, records[0].FleetNodeID)
	assert.Equal(t, "charon", records[0].AgentID)
}

func TestSync_RecordSyncHistory_Bad_MissingFile(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	records := readSyncRecords()
	require.Empty(t, records)

	recordSyncHistory("", "codex", 0, 64, 1, time.Now())
	records = readSyncRecords()
	require.Empty(t, records)
}

func TestSync_RecordSyncHistory_Ugly_CorruptFile(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	require.True(t, fs.WriteAtomic(syncRecordsPath(), "{not-json").OK)

	records := readSyncRecords()
	require.Empty(t, records)
}

func TestSync_HandleSyncPush_Good_ReportMetadata(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	fs.EnsureDir(WorkspaceMetaDir(workspaceDir))
	require.True(t, fs.Write(core.JoinPath(WorkspaceMetaDir(workspaceDir), "report.json"), `{"findings":[{"file":"main.go"}],"changes":{"files_changed":1}}`).OK)
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
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)

		dispatches, ok := payload["dispatches"].([]any)
		require.True(t, ok)
		require.Len(t, dispatches, 1)

		record, ok := dispatches[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "Which API version?", record["question"])
		require.Equal(t, float64(42), record["issue"])
		require.Equal(t, float64(2), record["runs"])
		require.Equal(t, "proc-1", record["process_id"])
		require.NotNil(t, record["report"])
		require.NotNil(t, record["findings"])
		require.NotNil(t, record["changes"])

		_, _ = w.Write([]byte(`{"data":{"synced":1}}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPush(context.Background(), "")
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
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
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	require.Len(t, output.Context, 1)
	assert.Equal(t, "ctx-1", output.Context[0]["id"])
}

func TestSync_HandleSyncPull_Bad(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")
	writeSyncContext([]map[string]any{
		{"id": "cached-1", "content": "Cached context"},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/context", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPull(context.Background(), "codex")
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	require.Len(t, output.Context, 1)
	assert.Equal(t, "cached-1", output.Context[0]["id"])
}

func TestSync_HandleSyncPull_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	t.Setenv("CORE_AGENT_API_KEY", "secret-token")
	writeSyncContext([]map[string]any{
		{"id": "cached-2", "content": "Fallback context"},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/agent/context", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{this is not json`))
	}))
	defer server.Close()

	subsystem := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       server.URL,
	}
	output, err := subsystem.syncPull(context.Background(), "codex")
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 1, output.Count)
	require.Len(t, output.Context, 1)
	assert.Equal(t, "cached-2", output.Context[0]["id"])
}

// schedule := syncBackoffSchedule(3) // 15s
func TestSync_SyncBackoffSchedule_Good(t *testing.T) {
	assert.Equal(t, time.Duration(0), syncBackoffSchedule(0))
	assert.Equal(t, time.Second, syncBackoffSchedule(1))
	assert.Equal(t, 5*time.Second, syncBackoffSchedule(2))
	assert.Equal(t, 15*time.Second, syncBackoffSchedule(3))
	assert.Equal(t, 60*time.Second, syncBackoffSchedule(4))
	assert.Equal(t, 5*time.Minute, syncBackoffSchedule(5))
	assert.Equal(t, 5*time.Minute, syncBackoffSchedule(100))
}

func TestSync_SyncBackoffSchedule_Bad_NegativeAttempts(t *testing.T) {
	assert.Equal(t, time.Duration(0), syncBackoffSchedule(-1))
	assert.Equal(t, time.Duration(0), syncBackoffSchedule(-5))
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
	require.NoError(t, err)
	queued := readSyncQueue()
	require.Len(t, queued, 1)
	assert.Equal(t, 1, queued[0].Attempts)
	assert.False(t, queued[0].NextAttempt.IsZero())
	assert.True(t, queued[0].NextAttempt.After(time.Now()))
	assert.True(t, queued[0].NextAttempt.Before(time.Now().Add(2*time.Second)))
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
		require.Equal(t, "/v1/agent/sync", r.URL.Path)
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
	require.Len(t, first, 1)

	// Mark as synced — next scan skips it.
	markDispatchesSynced(first)
	second := collectSyncDispatches()
	assert.Empty(t, second)

	// When the workspace gets a new run, fingerprint changes → rescan.
	writeStatusResult(workspaceDir, &WorkspaceStatus{
		Status:    "completed",
		Repo:      "go-io",
		Org:       "core",
		Runs:      2,
		UpdatedAt: updatedAt.Add(time.Hour),
	})
	third := collectSyncDispatches()
	assert.Len(t, third, 1)
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
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 0, output.Count)
	assert.Equal(t, 0, called)
	assert.Empty(t, readSyncQueue())
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
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, 0, output.Count)
	assert.Equal(t, 0, called, "backoff must skip the HTTP call")

	queued := readSyncQueue()
	require.Len(t, queued, 1)
	assert.Equal(t, 3, queued[0].Attempts)
}
