// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"testing"

	core "dappco.re/go"
)

// --- runnerWorkspaceStatusFromAgentic / agenticWorkspaceStatusFromRunner: nil ---

func TestPaths_RunnerWorkspaceStatusFromAgentic_Bad_Nil(t *testing.T) {
	core.AssertNil(t, runnerWorkspaceStatusFromAgentic(nil))
}

func TestPaths_AgenticWorkspaceStatusFromRunner_Bad_Nil(t *testing.T) {
	core.AssertNil(t, agenticWorkspaceStatusFromRunner(nil))
}

// --- WriteStatus: WriteAtomic failure ---

// When workspaceDir is an existing regular file, status.json's parent dir
// cannot be created (MkdirAll over a file fails) so WriteAtomic returns
// non-OK and WriteStatus surfaces the wrapped error.
func TestPaths_WriteStatus_Ugly_WriteAtomicFails(t *testing.T) {
	dir := t.TempDir()
	filePath := core.JoinPath(dir, "not-a-dir")
	core.RequireTrue(t, fs.Write(filePath, "i am a file").OK)

	result := WriteStatus(filePath, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
	})
	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
}
