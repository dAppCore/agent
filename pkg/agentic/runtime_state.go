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
	result := readRuntimeState()
	if !result.OK {
		return
	}

	state, ok := result.Value.(runtimeState)
	if !ok {
		return
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
		fs.Delete(runtimeStatePath())
		return
	}

	fs.EnsureDir(runtimeStateDir())
	fs.WriteAtomic(runtimeStatePath(), core.JSONMarshalString(state))
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
