// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestCommit_HandleCommit_Good_WritesJournal(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-42"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	metaDir := WorkspaceMetaDir(workspaceDir)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)
	core.RequireTrue(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "merged",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Fix tests",
		Branch: "agent/fix-tests",
		Runs:   3,
	}) == nil)
	core.RequireTrue(t, fs.Write(core.JoinPath(metaDir, "report.json"), `{"findings":[{"file":"main.go"}],"changes":{"files_changed":1}}`).OK)

	s := &PrepSubsystem{}
	result := s.handleCommit(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspaceName},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(CommitOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, workspaceName, output.Workspace)
	core.AssertFalse(t, output.Skipped)
	core.AssertNotEmpty(t, output.CommittedAt)

	journal := fs.Read(output.JournalPath)
	core.RequireTrue(t, journal.OK)
	core.AssertContains(t, journal.Value.(string), `"repo":"go-io"`)
	core.AssertContains(t, journal.Value.(string), `"committed_at"`)

	marker := fs.Read(output.MarkerPath)
	core.RequireTrue(t, marker.OK)
	core.AssertContains(t, marker.Value.(string), `"workspace":"core/go-io/task-42"`)
}

func TestCommit_HandleCommit_Bad_MissingWorkspace(t *testing.T) {
	s := &PrepSubsystem{}
	result := s.handleCommit(context.Background(), core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestCommit_HandleCommit_Ugly_Idempotent(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-43"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	core.RequireTrue(t, fs.EnsureDir(WorkspaceMetaDir(workspaceDir)).OK)
	core.RequireTrue(t, writeStatus(workspaceDir, &WorkspaceStatus{
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
	core.RequireTrue(t, first.OK)

	second := s.handleCommit(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspaceName},
	))
	core.RequireTrue(t, second.OK)

	output, ok := second.Value.(CommitOutput)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, output.Skipped)

	journal := fs.Read(output.JournalPath)
	core.RequireTrue(t, journal.OK)
	lines := len(core.Split(core.Trim(journal.Value.(string)), "\n"))
	core.AssertEqual(t, 1, lines)
}

func TestCommit_HandleCommit_Ugly_CorruptMarkerIsPreserved(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	workspaceName := "core/go-io/task-44"
	workspaceDir := core.JoinPath(WorkspaceRoot(), workspaceName)
	metaDir := WorkspaceMetaDir(workspaceDir)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)
	core.RequireTrue(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Fix tests",
		Branch: "agent/fix-tests",
		Runs:   2,
	}) == nil)
	core.RequireTrue(t, fs.Write(core.JoinPath(metaDir, "commit.json"), "{not-json").OK)

	s := &PrepSubsystem{}
	result := s.handleCommit(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspaceName},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(CommitOutput)
	core.RequireTrue(t, ok)
	core.AssertFalse(t, output.Skipped)

	marker := fs.Read(output.MarkerPath)
	core.RequireTrue(t, marker.OK)
	core.AssertContains(t, marker.Value.(string), `"workspace":"core/go-io/task-44"`)

	entries := listDirNames(fs.List(metaDir))
	var backupPath string
	for _, entry := range entries {
		if core.HasPrefix(entry, "commit.json.corrupt-") {
			backupPath = core.JoinPath(metaDir, entry)
			break
		}
	}
	core.RequireNotEmpty(t, backupPath)

	backup := fs.Read(backupPath)
	core.RequireTrue(t, backup.OK)
	core.AssertEqual(t, "{not-json", backup.Value.(string))
}
