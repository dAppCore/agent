// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestCommandsResume_CmdResume_Good_DryRun(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-42")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo", ".git")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), core.JSONMarshalString(WorkspaceStatus{
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
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(ResumeOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "core/go-io/task-42", output.Workspace)
	core.AssertEqual(t, "codex", output.Agent)
	core.AssertContains(t, output.Prompt, "Fix the failing tests")
	core.AssertContains(t, output.Prompt, "Use the new Core API")
}

func TestCommandsResume_CmdResume_Bad_MissingWorkspace(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdResume(core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "workspace is required")
}

func TestCommandsResume_CmdResume_Ugly_CorruptStatus(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	workspaceDir := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-42")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(workspaceDir, "repo", ".git")).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(workspaceDir, "status.json"), "{broken json").OK)

	result := s.cmdResume(core.NewOptions(core.Option{Key: "_arg", Value: "core/go-io/task-42"}))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "no status.json in workspace")
}

func TestCommandsResume_RegisterCommands_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCommands(c.Context())

	core.AssertContains(t, c.Commands(), "resume")
	core.AssertContains(t, c.Commands(), "agentic:resume")
}
