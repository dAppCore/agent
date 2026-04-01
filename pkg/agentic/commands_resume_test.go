// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsResume_CmdResume_Good_DryRun(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-42")
	require.True(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo", ".git")).OK)
	require.True(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status: "blocked",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "Fix the failing tests",
	})).OK)

	result := s.cmdResume(core.NewOptions(
		core.Option{Key: "workspace", Value: "core/go-io/task-42"},
		core.Option{Key: "answer", Value: "Use the new Core API"},
		core.Option{Key: "dry_run", Value: true},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(ResumeOutput)
	require.True(t, ok)
	assert.Equal(t, "core/go-io/task-42", output.Workspace)
	assert.Equal(t, "codex", output.Agent)
	assert.Contains(t, output.Prompt, "Fix the failing tests")
	assert.Contains(t, output.Prompt, "Use the new Core API")
}

func TestCommandsResume_CmdResume_Bad_MissingWorkspace(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdResume(core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "workspace is required")
}

func TestCommandsResume_CmdResume_Ugly_CorruptStatus(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-42")
	require.True(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo", ".git")).OK)
	require.True(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), "{broken json").OK)

	result := s.cmdResume(core.NewOptions(core.Option{Key: "_arg", Value: "core/go-io/task-42"}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "no status.json in workspace")
}

func TestCommandsResume_RegisterCommands_Good(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCommands(c.Context())

	assert.Contains(t, c.Commands(), "resume")
	assert.Contains(t, c.Commands(), "agentic:resume")
}
