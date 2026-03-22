// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestBaseAgent_Ugly_Empty(t *testing.T) {
	assert.Equal(t, "", baseAgent(""))
}

func TestBaseAgent_Ugly_MultipleColons(t *testing.T) {
	// SplitN with N=2 should only split on first colon
	assert.Equal(t, "claude", baseAgent("claude:opus:extra"))
}

func TestDispatchConfig_Good_Defaults(t *testing.T) {
	// loadAgentsConfig falls back to defaults when no config file exists
	s := &PrepSubsystem{codePath: t.TempDir()}
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	cfg := s.loadAgentsConfig()
	assert.Equal(t, "claude", cfg.Dispatch.DefaultAgent)
	assert.Equal(t, "coding", cfg.Dispatch.DefaultTemplate)
	assert.Equal(t, 1, cfg.Concurrency["claude"])
	assert.Equal(t, 3, cfg.Concurrency["gemini"])
}

func TestCanDispatchAgent_Good_NoConfig(t *testing.T) {
	// With no running workspaces and default config, should be able to dispatch
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(filepath.Join(root, "workspace")).OK)

	s := &PrepSubsystem{codePath: t.TempDir()}
	assert.True(t, s.canDispatchAgent("gemini"))
}

func TestCanDispatchAgent_Good_UnknownAgent(t *testing.T) {
	// Unknown agent has no limit, so always allowed
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(filepath.Join(root, "workspace")).OK)

	s := &PrepSubsystem{codePath: t.TempDir()}
	assert.True(t, s.canDispatchAgent("unknown-agent"))
}

func TestCountRunningByAgent_Good_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(filepath.Join(root, "workspace")).OK)

	s := &PrepSubsystem{}
	assert.Equal(t, 0, s.countRunningByAgent("gemini"))
	assert.Equal(t, 0, s.countRunningByAgent("claude"))
}

func TestCountRunningByAgent_Good_NoRunning(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Create a workspace with completed status under workspace/
	ws := filepath.Join(root, "workspace", "test-ws")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Agent:  "gemini",
		PID:    0,
	}))

	s := &PrepSubsystem{}
	assert.Equal(t, 0, s.countRunningByAgent("gemini"))
}

func TestDelayForAgent_Good_NoConfig(t *testing.T) {
	// With no config, delay should be 0
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	s := &PrepSubsystem{codePath: t.TempDir()}
	assert.Equal(t, 0, int(s.delayForAgent("gemini").Seconds()))
}
