// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsCommit_RegisterCommitCommands_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerCommitCommands()

	assert.Contains(t, c.Commands(), "commit")
	assert.Contains(t, c.Commands(), "agentic:commit")
}

func TestCommandsCommit_CmdCommit_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	workspaceName := "core/go-io/task-42"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	require.True(t, fs.EnsureDir(workspaceDir).OK)
	require.True(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Fix tests",
		Branch: "agent/fix-tests",
		Runs:   2,
	}) == nil)

	s := &PrepSubsystem{}
	output := captureStdout(t, func() {
		result := s.cmdCommit(core.NewOptions(core.Option{Key: "_arg", Value: workspaceName}))
		require.True(t, result.OK)

		commitOutput, ok := result.Value.(CommitOutput)
		require.True(t, ok)
		assert.Equal(t, workspaceName, commitOutput.Workspace)
		assert.False(t, commitOutput.Skipped)
		assert.NotEmpty(t, commitOutput.JournalPath)
		assert.NotEmpty(t, commitOutput.MarkerPath)
		assert.NotEmpty(t, commitOutput.CommittedAt)
	})

	assert.Contains(t, output, "workspace: core/go-io/task-42")
	assert.Contains(t, output, "journal:")
	assert.Contains(t, output, "committed:")
}

func TestCommandsCommit_CmdCommit_Bad_MissingWorkspace(t *testing.T) {
	s := &PrepSubsystem{}
	result := s.cmdCommit(core.NewOptions())

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "workspace is required")
}

func TestCommandsCommit_CmdCommit_Ugly_MissingStatus(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	workspaceName := "core/go-io/task-99"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	require.True(t, fs.EnsureDir(workspaceDir).OK)

	s := &PrepSubsystem{}
	result := s.cmdCommit(core.NewOptions(core.Option{Key: "_arg", Value: workspaceName}))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "status not found")
}

func TestCommandsCommit_CmdCommit_Ugly_Idempotent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	workspaceName := "core/go-io/task-100"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	require.True(t, fs.EnsureDir(workspaceDir).OK)
	require.True(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "merged",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Merge cleanly",
		Branch: "agent/merge-cleanly",
		Runs:   1,
	}) == nil)

	s := &PrepSubsystem{}
	first := s.cmdCommit(core.NewOptions(core.Option{Key: "_arg", Value: workspaceName}))
	require.True(t, first.OK)

	second := s.cmdCommit(core.NewOptions(core.Option{Key: "_arg", Value: workspaceName}))
	require.True(t, second.OK)

	commitOutput, ok := second.Value.(CommitOutput)
	require.True(t, ok)
	assert.True(t, commitOutput.Skipped)
	assert.NotEmpty(t, commitOutput.MarkerPath)
}
