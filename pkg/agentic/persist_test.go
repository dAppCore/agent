// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersist_OnStartup_Good_RestoresQueue(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	workspaceName := "core/go-io/task-restore"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-restore")
	require.True(t, fs.EnsureDir(workspaceDir).OK)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()
	if subsystem.stateStoreInstance() == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	queuedAt := time.Now().UTC().Add(-5 * time.Minute)
	subsystem.stateStoreSet(stateQueueGroup, workspaceName, queueEntry{
		Repo:     "go-io",
		Org:      "core",
		Task:     "restore queue",
		Branch:   "agent/restore-queue",
		Agent:    "codex:gpt-5.4",
		Status:   "queued",
		QueuedAt: queuedAt,
	})

	result := subsystem.restorePersistedState(context.Background())
	require.True(t, result.OK)
	assert.True(t, fs.IsFile(core.JoinPath(root, "db.duckdb")))

	registryResult := subsystem.Workspaces().Get(workspaceName)
	require.True(t, registryResult.OK)
	workspaceStatus, ok := registryResult.Value.(*WorkspaceStatus)
	require.True(t, ok)
	assert.Equal(t, "queued", workspaceStatus.Status)
	assert.Equal(t, "go-io", workspaceStatus.Repo)
	assert.Equal(t, "core", workspaceStatus.Org)
	assert.Equal(t, "agent/restore-queue", workspaceStatus.Branch)

	statusResult := ReadStatusResult(workspaceDir)
	require.True(t, statusResult.OK)
	restoredStatus, ok := workspaceStatusValue(statusResult)
	require.True(t, ok)
	assert.Equal(t, "queued", restoredStatus.Status)
	assert.Equal(t, "go-io", restoredStatus.Repo)
}

func TestPersist_OnStartup_Good_MarksDeadWorkers(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	workspaceName := "core/go-io/task-dead"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-dead")
	require.True(t, fs.EnsureDir(workspaceDir).OK)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()
	if subsystem.stateStoreInstance() == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	subsystem.stateStoreSet(stateRegistryGroup, workspaceName, WorkspaceStatus{
		Status:    "running",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		Org:       "core",
		Task:      "reap ghost",
		Branch:    "agent/reap-ghost",
		PID:       999999,
		ProcessID: "process-dead",
		StartedAt: time.Now().UTC().Add(-10 * time.Minute),
		UpdatedAt: time.Now().UTC().Add(-9 * time.Minute),
		Runs:      1,
	})

	result := subsystem.restorePersistedState(context.Background())
	require.True(t, result.OK)

	registryResult := subsystem.Workspaces().Get(workspaceName)
	require.True(t, registryResult.OK)
	workspaceStatus, ok := registryResult.Value.(*WorkspaceStatus)
	require.True(t, ok)
	assert.Equal(t, "failed", workspaceStatus.Status)
	assert.Equal(t, deadWorkerOnRestartQuestion, workspaceStatus.Question)
	assert.Zero(t, workspaceStatus.PID)
	assert.Empty(t, workspaceStatus.ProcessID)

	statusResult := ReadStatusResult(workspaceDir)
	require.True(t, statusResult.OK)
	restoredStatus, ok := workspaceStatusValue(statusResult)
	require.True(t, ok)
	assert.Equal(t, "failed", restoredStatus.Status)
	assert.Equal(t, deadWorkerOnRestartQuestion, restoredStatus.Question)
}

func TestPersist_OnShutdown_Good_PersistsQueue(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	subsystem := &PrepSubsystem{
		workspaces: core.NewRegistry[*WorkspaceStatus](),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}
	if subsystem.stateStoreInstance() == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	now := time.Now().UTC()
	subsystem.workspaces.Set("core/go-io/task-queue", &WorkspaceStatus{
		Status:    "queued",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		Org:       "core",
		Task:      "persist queue",
		Branch:    "agent/persist-queue",
		StartedAt: now,
		UpdatedAt: now,
	})
	subsystem.workspaces.Set("core/go-store/task-running", &WorkspaceStatus{
		Status:    "running",
		Agent:     "codex:gpt-5.4-mini",
		Repo:      "go-store",
		Org:       "core",
		Task:      "persist registry",
		Branch:    "agent/persist-registry",
		PID:       4242,
		StartedAt: now,
		UpdatedAt: now,
	})

	result := subsystem.OnShutdown(context.Background())
	require.True(t, result.OK)

	replay := &PrepSubsystem{}
	defer replay.closeStateStore()

	queueValue, ok := replay.stateStoreGet(stateQueueGroup, "core/go-io/task-queue")
	require.True(t, ok)
	var entry queueEntry
	require.True(t, core.JSONUnmarshalString(queueValue, &entry).OK)
	assert.Equal(t, "go-io", entry.Repo)
	assert.Equal(t, "agent/persist-queue", entry.Branch)

	registryValue, ok := replay.stateStoreGet(stateRegistryGroup, "core/go-store/task-running")
	require.True(t, ok)
	var stored WorkspaceStatus
	require.True(t, core.JSONUnmarshalString(registryValue, &stored).OK)
	assert.Equal(t, "running", stored.Status)
	assert.Equal(t, "go-store", stored.Repo)
}

func TestPersist_OnStartup_Bad_IgnoresInvalidStorePayload(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	validWorkspace := "core/go-io/task-valid"
	validWorkspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-valid")
	require.True(t, fs.EnsureDir(validWorkspaceDir).OK)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()
	storeInstance := subsystem.stateStoreInstance()
	if storeInstance == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	require.NoError(t, storeInstance.Set(stateRegistryGroup, "broken", "{"))
	subsystem.stateStoreSet(stateQueueGroup, validWorkspace, queueEntry{
		Repo:     "go-io",
		Org:      "core",
		Task:     "valid queue",
		Branch:   "agent/valid-queue",
		Agent:    "codex:gpt-5.4",
		QueuedAt: time.Now().UTC(),
	})

	result := subsystem.restorePersistedState(context.Background())
	require.True(t, result.OK)
	assert.False(t, subsystem.Workspaces().Get("broken").OK)
	assert.True(t, subsystem.Workspaces().Get(validWorkspace).OK)
}

func TestPersist_OnStartup_Ugly_CleansCompletedOrphanedWorkspace(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	workspaceName := "core/go-io/task-completed"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-completed")
	require.True(t, fs.EnsureDir(workspaceDir).OK)
	require.True(t, fs.WriteAtomic(WorkspaceStatusPath(workspaceDir), core.JSONMarshalString(WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		Org:       "core",
		Task:      "cleanup orphan",
		Branch:    "agent/cleanup-orphan",
		StartedAt: time.Now().UTC().Add(-2 * time.Hour),
		UpdatedAt: time.Now().UTC().Add(-time.Hour),
		Runs:      1,
	})).OK)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()
	if subsystem.stateStoreInstance() == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	subsystem.stateStoreSet(stateRegistryGroup, workspaceName, WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex:gpt-5.4",
		Repo:      "go-io",
		Org:       "core",
		Task:      "cleanup orphan",
		Branch:    "agent/cleanup-orphan",
		StartedAt: time.Now().UTC().Add(-2 * time.Hour),
		UpdatedAt: time.Now().UTC().Add(-time.Hour),
		Runs:      1,
	})

	result := subsystem.restorePersistedState(context.Background())
	require.True(t, result.OK)
	assert.False(t, fs.IsDir(workspaceDir))
}

func setPersistTestWorkspace(t *testing.T, root string) {
	t.Helper()
	setTestWorkspace(t, root)
	t.Setenv("CORE_HOME", root)
	t.Setenv("DIR_HOME", root)
	t.Setenv("HOME", root)
}
