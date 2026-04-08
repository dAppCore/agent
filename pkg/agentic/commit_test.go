// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommit_HandleCommit_Good_WritesJournal(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-42"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	metaDir := WorkspaceMetaDir(workspaceDir)
	require.True(t, fs.EnsureDir(metaDir).OK)
	require.True(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "merged",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Fix tests",
		Branch: "agent/fix-tests",
		Runs:   3,
	}) == nil)
	require.True(t, fs.Write(core.JoinPath(metaDir, "report.json"), `{"findings":[{"file":"main.go"}],"changes":{"files_changed":1}}`).OK)

	s := &PrepSubsystem{}
	result := s.handleCommit(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspaceName},
	))

	require.True(t, result.OK)
	output, ok := result.Value.(CommitOutput)
	require.True(t, ok)
	assert.Equal(t, workspaceName, output.Workspace)
	assert.False(t, output.Skipped)
	assert.NotEmpty(t, output.CommittedAt)

	journal := fs.Read(output.JournalPath)
	require.True(t, journal.OK)
	assert.Contains(t, journal.Value.(string), `"repo":"go-io"`)
	assert.Contains(t, journal.Value.(string), `"committed_at"`)

	marker := fs.Read(output.MarkerPath)
	require.True(t, marker.OK)
	assert.Contains(t, marker.Value.(string), `"workspace":"core/go-io/task-42"`)
}

func TestCommit_HandleCommit_Bad_MissingWorkspace(t *testing.T) {
	s := &PrepSubsystem{}
	result := s.handleCommit(context.Background(), core.NewOptions())

	assert.False(t, result.OK)
	assert.Error(t, result.Value.(error))
}

func TestCommit_HandleCommit_Ugly_Idempotent(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-43"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	require.True(t, fs.EnsureDir(WorkspaceMetaDir(workspaceDir)).OK)
	require.True(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Fix tests",
		Branch: "agent/fix-tests",
		Runs:   1,
	}) == nil)

	s := &PrepSubsystem{}
	first := s.handleCommit(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspaceName},
	))
	require.True(t, first.OK)

	second := s.handleCommit(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspaceName},
	))
	require.True(t, second.OK)

	output, ok := second.Value.(CommitOutput)
	require.True(t, ok)
	assert.True(t, output.Skipped)

	journal := fs.Read(output.JournalPath)
	require.True(t, journal.OK)
	lines := len(core.Split(core.Trim(journal.Value.(string)), "\n"))
	assert.Equal(t, 1, lines)
}
