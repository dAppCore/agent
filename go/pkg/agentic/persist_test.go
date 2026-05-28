// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestPersist_OnStartup_Good_RestoresQueue(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	workspaceName := "core/go-io/task-restore"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-restore")
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)

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
	core.RequireTrue(t, result.OK)
	core.AssertTrue(t, fs.IsFile(core.JoinPath(root, "db.duckdb")))

	registryResult := subsystem.Workspaces().Get(workspaceName)
	core.RequireTrue(t, registryResult.OK)
	workspaceStatus, ok := registryResult.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "queued", workspaceStatus.Status)
	core.AssertEqual(t, "go-io", workspaceStatus.Repo)
	core.AssertEqual(t, "core", workspaceStatus.Org)
	core.AssertEqual(t, "agent/restore-queue", workspaceStatus.Branch)

	statusResult := ReadStatusResult(workspaceDir)
	core.RequireTrue(t, statusResult.OK)
	restoredStatus, ok := workspaceStatusValue(statusResult)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "queued", restoredStatus.Status)
	core.AssertEqual(t, "go-io", restoredStatus.Repo)
}

func TestPersist_OnStartup_Good_MarksDeadWorkers(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	workspaceName := "core/go-io/task-dead"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-dead")
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)

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
	core.RequireTrue(t, result.OK)

	registryResult := subsystem.Workspaces().Get(workspaceName)
	core.RequireTrue(t, registryResult.OK)
	workspaceStatus, ok := registryResult.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "failed", workspaceStatus.Status)
	core.AssertEqual(t, deadWorkerOnRestartQuestion, workspaceStatus.Question)
	assertZero(t, workspaceStatus.PID)
	core.AssertEmpty(t, workspaceStatus.ProcessID)

	statusResult := ReadStatusResult(workspaceDir)
	core.RequireTrue(t, statusResult.OK)
	restoredStatus, ok := workspaceStatusValue(statusResult)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "failed", restoredStatus.Status)
	core.AssertEqual(t, deadWorkerOnRestartQuestion, restoredStatus.Question)
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
	core.RequireTrue(t, result.OK)

	replay := &PrepSubsystem{}
	defer replay.closeStateStore()

	queueValue, ok := replay.stateStoreGet(stateQueueGroup, "core/go-io/task-queue")
	core.RequireTrue(t, ok)
	var entry queueEntry
	core.RequireTrue(t, core.JSONUnmarshalString(queueValue, &entry).OK)
	core.AssertEqual(t, "go-io", entry.Repo)
	core.AssertEqual(t, "agent/persist-queue", entry.Branch)

	registryValue, ok := replay.stateStoreGet(stateRegistryGroup, "core/go-store/task-running")
	core.RequireTrue(t, ok)
	var stored WorkspaceStatus
	core.RequireTrue(t, core.JSONUnmarshalString(registryValue, &stored).OK)
	core.AssertEqual(t, "running", stored.Status)
	core.AssertEqual(t, "go-store", stored.Repo)
}

func TestPersist_OnStartup_Bad_IgnoresInvalidStorePayload(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	validWorkspace := "core/go-io/task-valid"
	validWorkspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-valid")
	core.RequireTrue(t, fs.EnsureDir(validWorkspaceDir).OK)

	subsystem := &PrepSubsystem{}
	defer subsystem.closeStateStore()
	storeInstance := subsystem.stateStoreInstance()
	if storeInstance == nil {
		t.Skip("go-store unavailable on this platform — RFC §15.6 graceful degradation")
	}

	if result := storeInstance.Set(stateRegistryGroup, "broken", "{"); !result.OK {
		t.Fatalf("seed broken registry payload: %v", resultErrorValue("TestPersist_OnStartup_Bad_IgnoresInvalidStorePayload", result))
	}
	subsystem.stateStoreSet(stateQueueGroup, validWorkspace, queueEntry{
		Repo:     "go-io",
		Org:      "core",
		Task:     "valid queue",
		Branch:   "agent/valid-queue",
		Agent:    "codex:gpt-5.4",
		QueuedAt: time.Now().UTC(),
	})

	result := subsystem.restorePersistedState(context.Background())
	core.RequireTrue(t, result.OK)
	core.AssertFalse(t, subsystem.Workspaces().Get("broken").OK)
	core.AssertTrue(t, subsystem.Workspaces().Get(validWorkspace).OK)
}

func TestPersist_OnStartup_Ugly_CleansCompletedOrphanedWorkspace(t *testing.T) {
	root := t.TempDir()
	setPersistTestWorkspace(t, root)

	workspaceName := "core/go-io/task-completed"
	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-completed")
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
	core.RequireTrue(t, fs.WriteAtomic(WorkspaceStatusPath(workspaceDir), core.JSONMarshalString(WorkspaceStatus{
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
	core.RequireTrue(t, result.OK)
	core.AssertFalse(t, fs.IsDir(workspaceDir))
}

func setPersistTestWorkspace(t *testing.T, root string) {
	t.Helper()
	setTestWorkspace(t, root)
	t.Setenv("CORE_HOME", root)
	t.Setenv("DIR_HOME", root)
	t.Setenv("HOME", root)
}
