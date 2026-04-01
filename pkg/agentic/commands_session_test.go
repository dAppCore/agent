// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsSession_RegisterSessionCommands_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerSessionCommands()

	assert.Contains(t, c.Commands(), "session/resume")
	assert.Contains(t, c.Commands(), "session/replay")
}

func TestCommandsSession_CmdSessionResume_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.NoError(t, writeSessionCache(&Session{
		SessionID:      "ses-abc123",
		AgentType:      "codex",
		Status:         "paused",
		ContextSummary: map[string]any{"repo": "go-io"},
		WorkLog: []map[string]any{
			{"type": "checkpoint", "message": "build passed"},
			{"type": "decision", "message": "open PR"},
		},
		Artifacts: []map[string]any{
			{"path": "pkg/agentic/session.go", "action": "modified"},
		},
		Handoff: map[string]any{
			"summary": "Ready for review",
		},
	}))

	result := s.cmdSessionResume(core.NewOptions(core.Option{Key: "session_id", Value: "ses-abc123"}))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionResumeOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "ses-abc123", output.Session.SessionID)
	assert.Equal(t, "active", output.Session.Status)
	assert.NotEmpty(t, output.HandoffContext)
	assert.Len(t, output.RecentActions, 2)
	assert.Len(t, output.Artifacts, 1)
}

func TestCommandsSession_CmdSessionResume_Bad_MissingSessionID(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionResume(core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "session_id is required")
}

func TestCommandsSession_CmdSessionResume_Ugly_CorruptedCacheFallsBackToRemoteError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.True(t, fs.EnsureDir(sessionCacheRoot()).OK)
	require.True(t, fs.WriteAtomic(sessionCachePath("ses-bad"), "{not-json").OK)

	result := s.cmdSessionResume(core.NewOptions(core.Option{Key: "session_id", Value: "ses-bad"}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "no platform API key configured")
}

func TestCommandsSession_CmdSessionReplay_Good(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.NoError(t, writeSessionCache(&Session{
		SessionID: "ses-replay",
		AgentType: "codex",
		Status:    "active",
		WorkLog: []map[string]any{
			{"type": "checkpoint", "message": "started", "timestamp": time.Now().Format(time.RFC3339)},
			{"type": "decision", "message": "kept scope small", "timestamp": time.Now().Format(time.RFC3339)},
			{"type": "error", "message": "flaky test", "timestamp": time.Now().Format(time.RFC3339)},
		},
		Artifacts: []map[string]any{
			{"path": "pkg/agentic/commands_session.go", "action": "created"},
		},
	}))

	result := s.cmdSessionReplay(core.NewOptions(core.Option{Key: "session_id", Value: "ses-replay"}))
	require.True(t, result.OK)

	output, ok := result.Value.(SessionReplayOutput)
	require.True(t, ok)
	assert.True(t, output.Success)
	assert.Equal(t, "ses-replay", output.ReplayContext["session_id"])
	assert.Contains(t, output.ReplayContext, "checkpoints")
	assert.Contains(t, output.ReplayContext, "decisions")
	assert.Contains(t, output.ReplayContext, "errors")
}

func TestCommandsSession_CmdSessionReplay_Bad_MissingSessionID(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdSessionReplay(core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "session_id is required")
}
