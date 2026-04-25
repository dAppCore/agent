// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"time"

	core "dappco.re/go/core"
)

type runtimeState struct {
	Backoff   map[string]time.Time `json:"backoff,omitempty"`
	FailCount map[string]int       `json:"fail_count,omitempty"`
}

func runtimeStateDir() string {
	return core.JoinPath(CoreRoot(), "runtime")
}

func runtimeStatePath() string {
	return core.JoinPath(runtimeStateDir(), "dispatch.json")
}

func (s *PrepSubsystem) loadRuntimeState() {
	state := runtimeState{
		Backoff:   make(map[string]time.Time),
		FailCount: make(map[string]int),
	}

	// Read the go-store cached runtime state first — when go-store is
	// unavailable the read is a no-op and we fall back to the JSON file.
	s.stateStoreRestore(stateRuntimeGroup, func(key, value string) bool {
		switch key {
		case "backoff":
			backoff := map[string]time.Time{}
			if result := core.JSONUnmarshalString(value, &backoff); result.OK {
				for pool, deadline := range backoff {
					state.Backoff[pool] = deadline
				}
			}
		case "fail_count":
			failCount := map[string]int{}
			if result := core.JSONUnmarshalString(value, &failCount); result.OK {
				for pool, count := range failCount {
					state.FailCount[pool] = count
				}
			}
		}
		return true
	})

	// The JSON file remains authoritative when go-store is missing so
	// existing deployments do not regress during the rollout.
	if result := readRuntimeState(); result.OK {
		if fileState, ok := result.Value.(runtimeState); ok {
			for pool, deadline := range fileState.Backoff {
				if _, seen := state.Backoff[pool]; !seen {
					state.Backoff[pool] = deadline
				}
			}
			for pool, count := range fileState.FailCount {
				if _, seen := state.FailCount[pool]; !seen {
					state.FailCount[pool] = count
				}
			}
		}
	}

	if s.backoff == nil {
		s.backoff = make(map[string]time.Time)
	}
	for pool, value := range state.Backoff {
		s.backoff[pool] = value
	}

	if s.failCount == nil {
		s.failCount = make(map[string]int)
	}
	for pool, count := range state.FailCount {
		s.failCount[pool] = count
	}
}

func (s *PrepSubsystem) persistRuntimeState() {
	state := runtimeState{
		Backoff:   make(map[string]time.Time),
		FailCount: make(map[string]int),
	}

	for pool, until := range s.backoff {
		if until.IsZero() {
			continue
		}
		state.Backoff[pool] = until.UTC()
	}
	for pool, count := range s.failCount {
		if count <= 0 {
			continue
		}
		state.FailCount[pool] = count
	}

	if len(state.Backoff) == 0 && len(state.FailCount) == 0 {
		if deleteResult := fs.Delete(runtimeStatePath()); !deleteResult.OK {
			core.Warn("agentic: failed to delete runtime state file", "path", runtimeStatePath(), "reason", deleteResult.Value)
		}
		s.stateStoreDelete(stateRuntimeGroup, "backoff")
		s.stateStoreDelete(stateRuntimeGroup, "fail_count")
		return
	}

	if ensureResult := fs.EnsureDir(runtimeStateDir()); !ensureResult.OK {
		core.Warn("agentic: failed to prepare runtime state directory", "path", runtimeStateDir(), "reason", ensureResult.Value)
	}
	if writeResult := fs.WriteAtomic(runtimeStatePath(), core.JSONMarshalString(state)); !writeResult.OK {
		core.Warn("agentic: failed to write runtime state", "path", runtimeStatePath(), "reason", writeResult.Value)
	}

	// Mirror the authoritative JSON to the go-store cache so restarts see
	// the same state even when the JSON file is archived or rotated.
	if len(state.Backoff) > 0 {
		s.stateStoreSet(stateRuntimeGroup, "backoff", state.Backoff)
	} else {
		s.stateStoreDelete(stateRuntimeGroup, "backoff")
	}
	if len(state.FailCount) > 0 {
		s.stateStoreSet(stateRuntimeGroup, "fail_count", state.FailCount)
	} else {
		s.stateStoreDelete(stateRuntimeGroup, "fail_count")
	}
}

func readRuntimeState() core.Result {
	result := fs.Read(runtimeStatePath())
	if !result.OK {
		return core.Result{Value: runtimeState{}, OK: false}
	}

	var state runtimeState
	parseResult := core.JSONUnmarshalString(result.Value.(string), &state)
	if !parseResult.OK {
		return core.Result{Value: runtimeState{}, OK: false}
	}

	if state.Backoff == nil {
		state.Backoff = make(map[string]time.Time)
	}
	if state.FailCount == nil {
		state.FailCount = make(map[string]int)
	}

	return core.Result{Value: state, OK: true}
}
