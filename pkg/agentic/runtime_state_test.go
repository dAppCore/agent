// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeState_PersistLoad_Good_RoundTrip(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	expectedBackoff := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	subsystem := &PrepSubsystem{
		backoff: map[string]time.Time{
			"codex": expectedBackoff,
		},
		failCount: map[string]int{
			"codex": 2,
		},
	}

	subsystem.persistRuntimeState()

	loaded := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	loaded.loadRuntimeState()

	require.Len(t, loaded.backoff, 1)
	assert.True(t, loaded.backoff["codex"].Equal(expectedBackoff))
	assert.Equal(t, 2, loaded.failCount["codex"])
}

func TestRuntimeState_Read_Bad_InvalidJSON(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	require.True(t, fs.EnsureDir(runtimeStateDir()).OK)
	require.True(t, fs.WriteAtomic(runtimeStatePath(), "{not-json").OK)

	result := readRuntimeState()
	assert.False(t, result.OK)
}

func TestRuntimeState_Persist_Ugly_EmptyStateDeletesFile(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	require.True(t, fs.EnsureDir(runtimeStateDir()).OK)
	require.True(t, fs.WriteAtomic(runtimeStatePath(), core.JSONMarshalString(runtimeState{
		Backoff: map[string]time.Time{
			"codex": time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
		},
		FailCount: map[string]int{
			"codex": 1,
		},
	})).OK)

	subsystem := &PrepSubsystem{
		backoff:   map[string]time.Time{},
		failCount: map[string]int{},
	}
	subsystem.persistRuntimeState()

	assert.False(t, fs.Read(runtimeStatePath()).OK)
}
