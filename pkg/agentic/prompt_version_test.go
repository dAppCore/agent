// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptVersion_ReadPromptSnapshot_Good(t *testing.T) {
	workspaceDir := t.TempDir()
	prompt := "TASK: Fix tests\n\nRead CODEX.md and commit when done."

	writeResult := writePromptSnapshot(workspaceDir, prompt)
	require.True(t, writeResult.OK)

	snapshot, err := readPromptSnapshot(workspaceDir)
	require.NoError(t, err)

	assert.Equal(t, promptSnapshotHash(prompt), snapshot.Hash)
	assert.Contains(t, snapshot.CreatedAt, "T")
	assert.Equal(t, prompt, snapshot.Content)
}

func TestPromptVersion_ReadPromptSnapshot_Bad_MissingWorkspace(t *testing.T) {
	_, err := readPromptSnapshot("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workspace is required")
}

func TestPromptVersion_ReadPromptSnapshot_Ugly_InvalidJson(t *testing.T) {
	workspaceDir := t.TempDir()
	metaDir := WorkspaceMetaDir(workspaceDir)
	require.True(t, fs.EnsureDir(metaDir).OK)
	require.True(t, fs.Write(core.JoinPath(metaDir, "prompt-version.json"), "{not-json").OK)

	_, err := readPromptSnapshot(workspaceDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse prompt snapshot")
}
