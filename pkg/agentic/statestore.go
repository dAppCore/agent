// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"sync"

	core "dappco.re/go/core"
	store "dappco.re/go/core/store"
)

// Usage example: `groupName := queueGroup` // "queue"
const (
	stateQueueGroup          = "queue"
	stateConcurrencyGroup    = "concurrency"
	stateRegistryGroup       = "registry"
	stateDispatchHistoryGroup = "dispatch_history"
	stateSyncQueueGroup      = "sync_queue"
	stateRuntimeGroup        = "runtime"
)

// stateStorePath returns the canonical path for the top-level agent DuckDB
// state file described in RFC §15.2 — `.core/db.duckdb` relative to CoreRoot.
//
// Usage example: `path := stateStorePath()`
func stateStorePath() string {
	return core.JoinPath(CoreRoot(), "db.duckdb")
}

// stateStoreRef keeps the store instance, its initialisation error, and a
// sync.Once so multiple callers observe the same lazily-initialised value.
type stateStoreRef struct {
	once     sync.Once
	instance *store.Store
	err      error
}

// stateStoreReference is a subsystem-scoped handle that exposes the lazily
// initialised go-store Store. The agent works fully offline when go-store
// cannot be initialised — RFC §15.6.
//
// Usage example: `st := s.stateStoreInstance(); if st == nil { return } // in-memory fallback`
func (s *PrepSubsystem) stateStoreInstance() *store.Store {
	if s == nil {
		return nil
	}
	ref := s.stateStoreRef()
	if ref == nil {
		return nil
	}
	ref.once.Do(func() {
		ref.instance, ref.err = openStateStore()
	})
	if ref.err != nil {
		return nil
	}
	return ref.instance
}

// stateStoreErr reports the last error observed while opening the go-store
// backend, so callers can decide whether to log or silently fall back.
//
// Usage example: `if err := s.stateStoreErr(); err != nil { core.Warn("state store unavailable", "err", err) }`
func (s *PrepSubsystem) stateStoreErr() error {
	if s == nil {
		return nil
	}
	ref := s.stateStoreRef()
	if ref == nil {
		return nil
	}
	_ = s.stateStoreInstance()
	return ref.err
}

// stateStoreRef returns the subsystem-scoped reference, allocating it lazily
// so zero-value subsystems (used by tests) do not crash.
func (s *PrepSubsystem) stateStoreRef() *stateStoreRef {
	if s == nil {
		return nil
	}
	s.stateOnce.Do(func() {
		s.state = &stateStoreRef{}
	})
	return s.state
}

// closeStateStore releases the go-store handle. Safe to call multiple times.
//
// Usage example: `s.closeStateStore()`
func (s *PrepSubsystem) closeStateStore() {
	if s == nil {
		return
	}
	ref := s.state
	if ref == nil {
		return
	}
	if ref.instance != nil {
		_ = ref.instance.Close()
		ref.instance = nil
	}
	ref.err = nil
	s.state = nil
	s.stateOnce = sync.Once{}
}

// openStateStore attempts to open the canonical state store at
// `.core/db.duckdb`. The filesystem is prepared first so new workspaces do
// not fail the first call. Errors are returned but never cause a panic — the
// caller falls back to in-memory or file-based state per RFC §15.6.
//
// Usage example: `st, err := openStateStore()`
func openStateStore() (*store.Store, error) {
	path := stateStorePath()
	directory := core.PathDir(path)
	if ensureResult := fs.EnsureDir(directory); !ensureResult.OK {
		if err, ok := ensureResult.Value.(error); ok {
			return nil, core.E("agentic.stateStore", "prepare state directory", err)
		}
		return nil, core.E("agentic.stateStore", "prepare state directory", nil)
	}

	storeInstance, err := store.New(path)
	if err != nil {
		return nil, core.E("agentic.stateStore", "open state store", err)
	}
	return storeInstance, nil
}

// stateStoreSet writes a JSON-encoded value to the given group+key if the
// store is available. No-op when go-store is not initialised.
//
// Usage example: `s.stateStoreSet(stateQueueGroup, "core/go-io", queueEntry)`
func (s *PrepSubsystem) stateStoreSet(group, key string, value any) {
	st := s.stateStoreInstance()
	if st == nil {
		return
	}
	payload := core.JSONMarshalString(value)
	_ = st.Set(group, key, payload)
}

// stateStoreDelete removes a key from the given group if the store is
// available. No-op when go-store is not initialised.
//
// Usage example: `s.stateStoreDelete(stateRegistryGroup, "core/go-io/task-5")`
func (s *PrepSubsystem) stateStoreDelete(group, key string) {
	st := s.stateStoreInstance()
	if st == nil {
		return
	}
	_ = st.Delete(group, key)
}

// stateStoreRestore iterates every entry in the given group and invokes
// the visitor with the decoded JSON payload. The visitor must return true
// to continue iteration or false to stop early. No-op when go-store is not
// initialised — callers continue to use file-based/in-memory state.
//
// Usage example:
//
//	s.stateStoreRestore(stateQueueGroup, func(key, value string) bool {
//	    var task QueuedTask
//	    core.JSONUnmarshalString(value, &task)
//	    s.queue.Enqueue(task)
//	    return true
//	})
func (s *PrepSubsystem) stateStoreRestore(group string, visit func(key, value string) bool) {
	st := s.stateStoreInstance()
	if st == nil || visit == nil {
		return
	}
	for entry, err := range st.AllSeq(group) {
		if err != nil {
			return
		}
		if !visit(entry.Key, entry.Value) {
			return
		}
	}
}

// stateStoreCount reports the number of entries in a group. Returns 0 when
// the store is unavailable so call sites can compare to zero without guards.
//
// Usage example: `if s.stateStoreCount(stateRegistryGroup) > 0 { /* restore workspaces */ }`
func (s *PrepSubsystem) stateStoreCount(group string) int {
	st := s.stateStoreInstance()
	if st == nil {
		return 0
	}
	count, err := st.Count(group)
	if err != nil {
		return 0
	}
	return count
}
