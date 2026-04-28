// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestCommandsCommit_RegisterCommitCommands_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}

	s.registerCommitCommands()

	core.AssertContains(t, c.Commands(), "commit")
	core.AssertContains(t, c.Commands(), "agentic:commit")
}

func TestCommandsCommit_CmdCommit_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-42"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
	core.RequireTrue(t, writeStatus(workspaceDir, &WorkspaceStatus{
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
		core.RequireTrue(t, result.OK)

		commitOutput, ok := result.Value.(CommitOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, workspaceName, commitOutput.Workspace)
		core.AssertFalse(t, commitOutput.Skipped)
		core.AssertNotEmpty(t, commitOutput.JournalPath)
		core.AssertNotEmpty(t, commitOutput.MarkerPath)
		core.AssertNotEmpty(t, commitOutput.CommittedAt)
	})

	core.AssertContains(t, output, "workspace: core/go-io/task-42")
	core.AssertContains(t, output, "journal:")
	core.AssertContains(t, output, "committed:")
}

func TestCommandsCommit_CmdCommit_Bad_MissingWorkspace(t *testing.T) {
	s := &PrepSubsystem{}
	result := s.cmdCommit(core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "workspace is required")
}

func TestCommandsCommit_CmdCommit_Ugly_MissingStatus(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-99"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)

	s := &PrepSubsystem{}
	result := s.cmdCommit(core.NewOptions(core.Option{Key: "_arg", Value: workspaceName}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "status not found")
}

func TestCommandsCommit_CmdCommit_Ugly_Idempotent(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-100"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
	core.RequireTrue(t, writeStatus(workspaceDir, &WorkspaceStatus{
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
	core.RequireTrue(t, first.OK)

	second := s.cmdCommit(core.NewOptions(core.Option{Key: "_arg", Value: workspaceName}))
	core.RequireTrue(t, second.OK)

	commitOutput, ok := second.Value.(CommitOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, commitOutput.Skipped)
	core.AssertNotEmpty(t, commitOutput.MarkerPath)
}
