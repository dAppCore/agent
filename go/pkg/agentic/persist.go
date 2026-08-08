// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"syscall"
	"time"

	core "dappco.re/go"
)

const deadWorkerOnRestartQuestion = "dead worker on restart"

type concurrencySnapshot struct {
	Running    int       `json:"running,omitempty"`
	Tracked    int       `json:"tracked,omitempty"`
	SnapshotAt time.Time `json:"snapshot_at,omitempty"`
}

// result := s.restorePersistedState(context.Background())
func (s *PrepSubsystem) restorePersistedState(_ context.Context) core.Result {
	if s == nil {
		return core.Result{OK: true}
	}

	s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	if storeInstance := s.stateStoreInstance(); storeInstance == nil {
		if err := stateStoreErr(s); err != nil {
			core.Warn("agentic.restorePersistedState: state store unavailable", "reason", err)
		}
	}

	restored := map[string]*WorkspaceStatus{}
	s.restoreRegistrySnapshot(restored)
	s.restoreQueueSnapshot(restored)
	s.restoreConcurrencySnapshot()
	s.restoreWorkspaceSnapshot(restored)

	registryKeep := map[string]struct{}{}
	queueKeep := map[string]struct{}{}

	for name, workspaceStatus := range restored {
		if workspaceStatus == nil {
			continue
		}

		registryKeep[name] = struct{}{}
		workspaceDir := core.JoinPath(WorkspaceRoot(), name)
		changed := s.normaliseRestoredWorkspace(workspaceStatus)

		if workspaceStatus.Status == "completed" && fs.IsDir(workspaceDir).OK {
			_ = fs.DeleteAll(workspaceDir) // best-effort cleanup; may already be gone
		} else if fs.IsDir(workspaceDir).OK && (changed || !fs.IsFile(WorkspaceStatusPath(workspaceDir)).OK) {
			s.writePersistedWorkspaceStatus(workspaceDir, workspaceStatus)
		}

		// Reported: Set only fails on a locked or sealed registry, so a
		// failure here means this workspace silently stops being tracked.
		if r := s.workspaces.Set(name, cloneWorkspaceStatus(workspaceStatus)); !r.OK {
			core.Warn("agentic: failed to track workspace", "reason", r.Value)
		}
		s.stateStoreSet(stateRegistryGroup, name, workspaceStatus)

		if workspaceStatus.Status == "queued" {
			queueKeep[name] = struct{}{}
			s.stateStoreSet(stateQueueGroup, name, queueEntryFromStatus(workspaceStatus))
		} else {
			s.stateStoreDelete(stateQueueGroup, name)
		}
	}

	s.pruneStateGroup(stateRegistryGroup, registryKeep)
	s.pruneStateGroup(stateQueueGroup, queueKeep)
	s.refreshConcurrencySnapshot()

	return core.Result{OK: true}
}

// result := s.flushPersistedState(context.Background())
func (s *PrepSubsystem) flushPersistedState(_ context.Context) core.Result {
	if s == nil {
		return core.Result{OK: true}
	}

	if len(s.backoff) > 0 || len(s.failCount) > 0 {
		s.persistRuntimeState()
	}
	if s.workspaces == nil {
		return core.Result{OK: true}
	}

	if storeInstance := s.stateStoreInstance(); storeInstance == nil {
		if err := stateStoreErr(s); err != nil {
			core.Warn("agentic.flushPersistedState: state store unavailable", "reason", err)
		}
	}
	registryKeep := map[string]struct{}{}
	queueKeep := map[string]struct{}{}

	s.workspaces.Each(func(name string, workspaceStatus *WorkspaceStatus) {
		if name == "" {
			return
		}
		if workspaceStatus == nil {
			s.stateStoreDelete(stateRegistryGroup, name)
			s.stateStoreDelete(stateQueueGroup, name)
			return
		}

		registryKeep[name] = struct{}{}
		s.stateStoreSet(stateRegistryGroup, name, workspaceStatus)

		if workspaceStatus.Status == "queued" {
			queueKeep[name] = struct{}{}
			s.stateStoreSet(stateQueueGroup, name, queueEntryFromStatus(workspaceStatus))
		} else {
			s.stateStoreDelete(stateQueueGroup, name)
		}
	})

	s.pruneStateGroup(stateRegistryGroup, registryKeep)
	s.pruneStateGroup(stateQueueGroup, queueKeep)
	s.refreshConcurrencySnapshot()

	return core.Result{OK: true}
}

func (s *PrepSubsystem) restoreRegistrySnapshot(restored map[string]*WorkspaceStatus) {
	if restored == nil {
		return
	}
	s.stateStoreRestore(stateRegistryGroup, func(key, value string) bool {
		var workspaceStatus WorkspaceStatus
		if result := core.JSONUnmarshalString(value, &workspaceStatus); !result.OK {
			return true
		}
		restored[key] = cloneWorkspaceStatus(&workspaceStatus)
		return true
	})
}

func (s *PrepSubsystem) restoreQueueSnapshot(restored map[string]*WorkspaceStatus) {
	if restored == nil {
		return
	}
	s.stateStoreRestore(stateQueueGroup, func(key, value string) bool {
		if _, ok := restored[key]; ok {
			return true
		}

		var entry queueEntry
		if result := core.JSONUnmarshalString(value, &entry); !result.OK {
			return true
		}

		restored[key] = workspaceStatusFromQueueEntry(entry)
		return true
	})
}

func (s *PrepSubsystem) restoreConcurrencySnapshot() {
	s.stateStoreRestore(stateConcurrencyGroup, func(_ string, value string) bool {
		var snapshot concurrencySnapshot
		if result := core.JSONUnmarshalString(value, &snapshot); !result.OK {
			return true
		}
		return true
	})
}

func (s *PrepSubsystem) restoreWorkspaceSnapshot(restored map[string]*WorkspaceStatus) {
	if restored == nil {
		return
	}
	for _, statusPath := range WorkspaceStatusPaths() {
		workspaceDir := core.PathDir(statusPath)
		result := ReadStatusResult(workspaceDir)
		workspaceStatus, ok := workspaceStatusValue(result)
		if !ok {
			continue
		}
		restored[WorkspaceName(workspaceDir)] = cloneWorkspaceStatus(workspaceStatus)
	}
}

func (s *PrepSubsystem) normaliseRestoredWorkspace(workspaceStatus *WorkspaceStatus) bool {
	if workspaceStatus == nil {
		return false
	}

	if workspaceStatus.Status != "running" {
		if workspaceStatus.UpdatedAt.IsZero() {
			if workspaceStatus.StartedAt.IsZero() {
				workspaceStatus.UpdatedAt = time.Now().UTC()
			} else {
				workspaceStatus.UpdatedAt = workspaceStatus.StartedAt.UTC()
			}
			return true
		}
		return false
	}

	if s.pidAlive(workspaceStatus.ProcessID, workspaceStatus.PID) {
		if workspaceStatus.UpdatedAt.IsZero() {
			workspaceStatus.UpdatedAt = time.Now().UTC()
			return true
		}
		return false
	}

	workspaceStatus.Status = "failed"
	workspaceStatus.Question = deadWorkerOnRestartQuestion
	workspaceStatus.PID = 0
	workspaceStatus.ProcessID = ""
	workspaceStatus.UpdatedAt = time.Now().UTC()
	return true
}

func (s *PrepSubsystem) pidAlive(processID string, pid int) bool {
	if s != nil && s.ServiceRuntime != nil && ProcessAlive(s.Core(), processID, pid) {
		return true
	}
	if pid <= 0 {
		return false
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}

	return core.Is(err, syscall.EPERM)
}

func (s *PrepSubsystem) pruneStateGroup(group string, keep map[string]struct{}) {
	if keep == nil {
		keep = map[string]struct{}{}
	}

	var stale []string
	s.stateStoreRestore(group, func(key, _ string) bool {
		if _, ok := keep[key]; !ok {
			stale = append(stale, key)
		}
		return true
	})

	for _, key := range stale {
		s.stateStoreDelete(group, key)
	}
}

func (s *PrepSubsystem) writePersistedWorkspaceStatus(workspaceDir string, workspaceStatus *WorkspaceStatus) {
	if workspaceDir == "" || workspaceStatus == nil || !fs.IsDir(workspaceDir).OK {
		return
	}
	if workspaceStatus.UpdatedAt.IsZero() {
		workspaceStatus.UpdatedAt = time.Now().UTC()
	}
	if writeResult := fs.WriteAtomic(WorkspaceStatusPath(workspaceDir), core.JSONMarshalString(workspaceStatus)); !writeResult.OK {
		core.Warn("agentic.persist: failed to write persisted workspace status", `path`, WorkspaceStatusPath(workspaceDir), "reason", writeResult.Value)
	}
}

func cloneWorkspaceStatus(workspaceStatus *WorkspaceStatus) *WorkspaceStatus {
	if workspaceStatus == nil {
		return nil
	}
	cloned := *workspaceStatus
	return &cloned
}

func workspaceStatusFromQueueEntry(entry queueEntry) *WorkspaceStatus {
	queuedAt := entry.QueuedAt.UTC()
	if queuedAt.IsZero() {
		queuedAt = time.Now().UTC()
	}
	return &WorkspaceStatus{
		Status:    "queued",
		Agent:     entry.Agent,
		Repo:      entry.Repo,
		Org:       entry.Org,
		Task:      entry.Task,
		Branch:    entry.Branch,
		StartedAt: queuedAt,
		UpdatedAt: queuedAt,
	}
}
