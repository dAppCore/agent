// SPDX-License-Identifier: EUPL-1.2

package runner

import (
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
)

func mustReadStatus(t *testing.T, dir string) *WorkspaceStatus {
	t.Helper()

	result := ReadStatusResult(dir)
	core.RequireTrue(t, result.OK)

	status, ok := result.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	return status
}

func TestPaths_CoreRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-root")
	got := CoreRoot()
	core.AssertEqual(t, "/tmp/core-root", got)
	core.AssertContains(t, got, "core-root")
}

func TestPaths_CoreRoot_Bad(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home := core.Env("DIR_HOME")
	core.AssertEqual(t, home+"/Code/.core", CoreRoot())
}

func TestPaths_CoreRoot_Ugly(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-røot")
	got := CoreRoot()
	core.AssertEqual(t, "/tmp/core-røot", got)
	core.AssertContains(t, got, "røot")
}

func TestPaths_WorkspaceRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-root")
	got := WorkspaceRoot()
	core.AssertEqual(t, "/tmp/core-root/workspace", got)
	core.AssertContains(t, got, "workspace")
}

func TestPaths_WorkspaceRoot_Bad(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home := core.Env("DIR_HOME")
	core.AssertEqual(t, home+"/Code/.core/workspace", WorkspaceRoot())
}

func TestPaths_WorkspaceRoot_Ugly(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/srv/core/tenant-a")
	got := WorkspaceRoot()
	core.AssertEqual(t, "/srv/core/tenant-a/workspace", got)
	core.AssertContains(t, got, "tenant-a")
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
	core.RequireTrue(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), core.JSONMarshalString(status)).OK)

	st := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "completed", st.Status)
	core.AssertEqual(t, "codex", st.Agent)
	core.AssertEqual(t, "go-io", st.Repo)
	core.AssertEqual(t, "agent/ax-cleanup", st.Branch)
	core.AssertEqual(t, 2, st.Runs)
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
	core.RequireTrue(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), core.JSONMarshalString(status)).OK)

	result := ReadStatusResult(wsDir)
	core.RequireTrue(t, result.OK)

	st, ok := result.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", st.Status)
	core.AssertEqual(t, "claude", st.Agent)
	core.AssertEqual(t, "go-log", st.Repo)
	core.AssertEqual(t, "core", st.Org)
	core.AssertEqual(t, "finish AX cleanup", st.Task)
	core.AssertEqual(t, "agent/ax-cleanup", st.Branch)
	core.AssertEqual(t, 21, st.PID)
	core.AssertEqual(t, "Ready?", st.Question)
	core.AssertEqual(t, "https://forge.test/core/go-log/pulls/12", st.PRURL)
	core.AssertEqual(t, 4, st.Runs)
}

func TestNoFile_ReadStatusResult_Bad(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
}

func TestInvalidJSON_ReadStatusResult_Ugly(t *testing.T) {
	wsDir := t.TempDir()
	core.RequireTrue(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), "{not-json").OK)

	result := ReadStatusResult(wsDir)
	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
}

func TestPaths_ReadStatus_Bad(t *testing.T) {
	wsDir := t.TempDir()
	core.RequireTrue(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), "{not-json").OK)

	result := ReadStatusResult(wsDir)
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.AssertTrue(t, ok)
}

func TestPaths_ReadStatus_Ugly(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.AssertTrue(t, ok)
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
	core.AssertTrue(t, result.OK)

	st := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "running", st.Status)
	core.AssertEqual(t, "codex", st.Agent)
	core.AssertEqual(t, "go-io", st.Repo)
	core.AssertEqual(t, "agent/ax-cleanup", st.Branch)
	core.AssertEqual(t, 1, st.Runs)

	agenticResult := agentic.ReadStatusResult(wsDir)
	core.RequireTrue(t, agenticResult.OK)
	agenticStatus, ok := agenticResult.Value.(*agentic.WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertFalse(t, agenticStatus.UpdatedAt.IsZero())
}

func TestPaths_WriteStatus_Bad(t *testing.T) {
	wsDir := t.TempDir()

	result := WriteStatus(wsDir, nil)
	core.AssertFalse(t, result.OK)

	core.AssertFalse(t, agentic.LocalFs().Read(agentic.WorkspaceStatusPath(wsDir)).OK)
}

func TestPaths_WriteStatus_Ugly(t *testing.T) {
	wsDir := t.TempDir()

	core.AssertTrue(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "First run",
	}).OK)
	core.AssertTrue(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "claude",
		Repo:      "go-io",
		Task:      "Second run",
		Branch:    "agent/ax-cleanup",
		StartedAt: time.Now(),
		Runs:      3,
	}).OK)

	st := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "completed", st.Status)
	core.AssertEqual(t, "claude", st.Agent)
	core.AssertEqual(t, "agent/ax-cleanup", st.Branch)
	core.AssertEqual(t, 3, st.Runs)
}
