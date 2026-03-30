// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"testing"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustReadStatus(t *testing.T, dir string) *WorkspaceStatus {
	t.Helper()

	result := ReadStatusResult(dir)
	require.True(t, result.OK)

	status, ok := result.Value.(*WorkspaceStatus)
	require.True(t, ok)
	return status
}

func TestPaths_CoreRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-root")
	assert.Equal(t, "/tmp/core-root", CoreRoot())
}

func TestPaths_CoreRoot_Bad(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home := core.Env("DIR_HOME")
	assert.Equal(t, home+"/Code/.core", CoreRoot())
}

func TestPaths_CoreRoot_Ugly(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-røot")
	assert.Equal(t, "/tmp/core-røot", CoreRoot())
}

func TestPaths_WorkspaceRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-root")
	assert.Equal(t, "/tmp/core-root/workspace", WorkspaceRoot())
}

func TestPaths_WorkspaceRoot_Bad(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home := core.Env("DIR_HOME")
	assert.Equal(t, home+"/Code/.core/workspace", WorkspaceRoot())
}

func TestPaths_WorkspaceRoot_Ugly(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/srv/core/tenant-a")
	assert.Equal(t, "/srv/core/tenant-a/workspace", WorkspaceRoot())
}

func TestPaths_ReadStatus_Good(t *testing.T) {
	wsDir := t.TempDir()
	status := &agentic.WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Task:      "Finish AX cleanup",
		Branch:    "agent/ax-cleanup",
		PID:       42,
		ProcessID: "proc-123",
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
		Question:  "Ready?",
		Runs:      2,
		PRURL:     "https://forge.test/core/go-io/pulls/12",
	}
	require.True(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), core.JSONMarshalString(status)).OK)

	st := mustReadStatus(t, wsDir)
	assert.Equal(t, "completed", st.Status)
	assert.Equal(t, "codex", st.Agent)
	assert.Equal(t, "go-io", st.Repo)
	assert.Equal(t, "agent/ax-cleanup", st.Branch)
	assert.Equal(t, 2, st.Runs)
}

func TestPaths_ReadStatusResult_Good(t *testing.T) {
	wsDir := t.TempDir()
	status := &agentic.WorkspaceStatus{
		Status:    "completed",
		Agent:     "claude",
		Repo:      "go-log",
		Org:       "core",
		Task:      "finish AX cleanup",
		Branch:    "agent/ax-cleanup",
		PID:       21,
		Question:  "Ready?",
		PRURL:     "https://forge.test/core/go-log/pulls/12",
		StartedAt: time.Now(),
		Runs:      4,
	}
	require.True(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), core.JSONMarshalString(status)).OK)

	result := ReadStatusResult(wsDir)
	require.True(t, result.OK)

	st, ok := result.Value.(*WorkspaceStatus)
	require.True(t, ok)
	assert.Equal(t, "completed", st.Status)
	assert.Equal(t, "claude", st.Agent)
	assert.Equal(t, "go-log", st.Repo)
	assert.Equal(t, "core", st.Org)
	assert.Equal(t, "finish AX cleanup", st.Task)
	assert.Equal(t, "agent/ax-cleanup", st.Branch)
	assert.Equal(t, 21, st.PID)
	assert.Equal(t, "Ready?", st.Question)
	assert.Equal(t, "https://forge.test/core/go-log/pulls/12", st.PRURL)
	assert.Equal(t, 4, st.Runs)
}

func TestPaths_ReadStatusResult_Bad_NoFile(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	assert.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Error(t, err)
}

func TestPaths_ReadStatusResult_Ugly_InvalidJSON(t *testing.T) {
	wsDir := t.TempDir()
	require.True(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), "{not-json").OK)

	result := ReadStatusResult(wsDir)
	assert.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Error(t, err)
}

func TestPaths_ReadStatus_Bad(t *testing.T) {
	wsDir := t.TempDir()
	require.True(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), "{not-json").OK)

	result := ReadStatusResult(wsDir)
	assert.False(t, result.OK)
	_, ok := result.Value.(error)
	assert.True(t, ok)
}

func TestPaths_ReadStatus_Ugly(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	assert.False(t, result.OK)
	_, ok := result.Value.(error)
	assert.True(t, ok)
}

func TestPaths_WriteStatus_Good(t *testing.T) {
	wsDir := t.TempDir()

	result := WriteStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "Track workspace",
		Branch: "agent/ax-cleanup",
		Runs:   1,
	})
	assert.True(t, result.OK)

	st := mustReadStatus(t, wsDir)
	assert.Equal(t, "running", st.Status)
	assert.Equal(t, "codex", st.Agent)
	assert.Equal(t, "go-io", st.Repo)
	assert.Equal(t, "agent/ax-cleanup", st.Branch)
	assert.Equal(t, 1, st.Runs)

	agenticResult := agentic.ReadStatusResult(wsDir)
	require.True(t, agenticResult.OK)
	agenticStatus, ok := agenticResult.Value.(*agentic.WorkspaceStatus)
	require.True(t, ok)
	assert.False(t, agenticStatus.UpdatedAt.IsZero())
}

func TestPaths_WriteStatus_Bad(t *testing.T) {
	wsDir := t.TempDir()

	result := WriteStatus(wsDir, nil)
	assert.False(t, result.OK)

	assert.False(t, agentic.LocalFs().Read(agentic.WorkspaceStatusPath(wsDir)).OK)
}

func TestPaths_WriteStatus_Ugly(t *testing.T) {
	wsDir := t.TempDir()

	assert.True(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "First run",
	}).OK)
	assert.True(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "claude",
		Repo:      "go-io",
		Task:      "Second run",
		Branch:    "agent/ax-cleanup",
		StartedAt: time.Now(),
		Runs:      3,
	}).OK)

	st := mustReadStatus(t, wsDir)
	assert.Equal(t, "completed", st.Status)
	assert.Equal(t, "claude", st.Agent)
	assert.Equal(t, "agent/ax-cleanup", st.Branch)
	assert.Equal(t, 3, st.Runs)
}
