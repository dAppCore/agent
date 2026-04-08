// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQueue_BaseAgent_Ugly_Empty(t *testing.T) {
	assert.Equal(t, "", baseAgent(""))
}

func TestQueue_BaseAgent_Ugly_MultipleColons(t *testing.T) {
	// SplitN with N=2 should only split on first colon
	assert.Equal(t, "claude", baseAgent("claude:opus:extra"))
}

func TestQueue_DispatchConfig_Good_Defaults(t *testing.T) {
	// loadAgentsConfig falls back to defaults when no config file exists
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	cfg := s.loadAgentsConfig()
	assert.Equal(t, "claude", cfg.Dispatch.DefaultAgent)
	assert.Equal(t, "coding", cfg.Dispatch.DefaultTemplate)
	assert.Equal(t, 1, cfg.Concurrency["claude"].Total)
	assert.Equal(t, 3, cfg.Concurrency["gemini"].Total)
}

func TestQueue_DispatchConfig_Good_WorkspaceRootOverride(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	customRoot := core.JoinPath(root, "agent-workspaces")
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  workspace_root: ", customRoot, "\n",
	)).OK)

	t.Cleanup(func() {
		setWorkspaceRootOverride("")
	})

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	cfg := s.loadAgentsConfig()

	assert.Equal(t, customRoot, cfg.Dispatch.WorkspaceRoot)
	assert.Equal(t, customRoot, WorkspaceRoot())
}

func TestQueue_CanDispatchAgent_Good_NoConfig(t *testing.T) {
	// With no running workspaces and default config, should be able to dispatch
	root := t.TempDir()
	setTestWorkspace(t, root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	assert.True(t, s.canDispatchAgent("gemini"))
}

func TestQueue_CanDispatchAgent_Good_UnknownAgent(t *testing.T) {
	// Unknown agent has no limit, so always allowed
	root := t.TempDir()
	setTestWorkspace(t, root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	assert.True(t, s.canDispatchAgent("unknown-agent"))
}

func TestQueue_CountRunningByAgent_Good_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.Equal(t, 0, s.countRunningByAgent("gemini"))
	assert.Equal(t, 0, s.countRunningByAgent("claude"))
}

func TestQueue_CountRunningByAgent_Good_NoRunning(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Create a workspace with completed status under workspace/
	ws := core.JoinPath(root, "workspace", "test-ws")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Agent:  "gemini",
		PID:    0,
	}))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.Equal(t, 0, s.countRunningByAgent("gemini"))
}

func TestQueue_DelayForAgent_Good_NoConfig(t *testing.T) {
	// With no config, delay should be 0
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir()}
	assert.Equal(t, 0, int(s.delayForAgent("gemini").Seconds()))
}
