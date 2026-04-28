// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestPromptVersion_ReadPromptSnapshot_Good(t *testing.T) {
	workspaceDir := t.TempDir()
	prompt := "TASK: Fix tests\n\nRead CODEX.md and commit when done."

	writeResult := writePromptSnapshot(workspaceDir, prompt)
	core.RequireTrue(t, writeResult.OK)

	snapshot, err := readPromptSnapshot(workspaceDir)
	core.RequireNoError(t, err)

	core.AssertEqual(t, promptSnapshotHash(prompt), snapshot.Hash)
	core.AssertContains(t, snapshot.CreatedAt, "T")
	core.AssertEqual(t, prompt, snapshot.Content)
}

func TestPromptVersion_ReadPromptSnapshot_Bad_MissingWorkspace(t *testing.T) {
	_, err := readPromptSnapshot("")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "workspace is required")
}

func TestPromptVersion_ReadPromptSnapshot_Ugly_InvalidJson(t *testing.T) {
	workspaceDir := t.TempDir()
	metaDir := WorkspaceMetaDir(workspaceDir)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(metaDir, "prompt-version.json"), "{not-json").OK)

	_, err := readPromptSnapshot(workspaceDir)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse prompt snapshot")
}

func TestPromptVersion_PromptVersionTool_Good(t *testing.T) {
	workspaceDir := t.TempDir()
	prompt := "TASK: Fix tests\n\nRead CODEX.md and commit when done."

	core.RequireTrue(t, writePromptSnapshot(workspaceDir, prompt).OK)

	_, output, err := (&PrepSubsystem{}).promptVersionTool(context.Background(), nil, PromptVersionInput{
		Workspace: workspaceDir,
	})
	core.RequireNoError(t, err)

	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, workspaceDir, output.Workspace)
	core.AssertEqual(t, promptSnapshotHash(prompt), output.Snapshot.Hash)
}

func TestPromptVersion_PromptVersionTool_Bad_MissingWorkspace(t *testing.T) {
	_, _, err := (&PrepSubsystem{}).promptVersionTool(context.Background(), nil, PromptVersionInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "workspace is required")
}

func TestPromptVersion_PromptVersionTool_Ugly_InvalidJson(t *testing.T) {
	workspaceDir := t.TempDir()
	metaDir := WorkspaceMetaDir(workspaceDir)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(metaDir, "prompt-version.json"), "{not-json").OK)

	_, _, err := (&PrepSubsystem{}).promptVersionTool(context.Background(), nil, PromptVersionInput{
		Workspace: workspaceDir,
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse prompt snapshot")
}
