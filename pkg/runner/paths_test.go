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

func TestPaths_CoreRoot_Good_EnvVar(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-root")
	assert.Equal(t, "/tmp/core-root", CoreRoot())
}

func TestPaths_CoreRoot_Bad_Fallback(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home := core.Env("DIR_HOME")
	assert.Equal(t, home+"/Code/.core", CoreRoot())
}

func TestPaths_CoreRoot_Ugly_UnicodePath(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-røot")
	assert.Equal(t, "/tmp/core-røot", CoreRoot())
}

func TestPaths_WorkspaceRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/core-root")
	assert.Equal(t, "/tmp/core-root/workspace", WorkspaceRoot())
}

func TestPaths_WorkspaceRoot_Bad_EmptyEnv(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home := core.Env("DIR_HOME")
	assert.Equal(t, home+"/Code/.core/workspace", WorkspaceRoot())
}

func TestPaths_WorkspaceRoot_Ugly_NestedCoreRoot(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/srv/core/tenant-a")
	assert.Equal(t, "/srv/core/tenant-a/workspace", WorkspaceRoot())
}

func TestPaths_ReadStatus_Good_AgenticShape(t *testing.T) {
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

	st, err := ReadStatus(wsDir)
	require.NoError(t, err)
	assert.Equal(t, "completed", st.Status)
	assert.Equal(t, "codex", st.Agent)
	assert.Equal(t, "go-io", st.Repo)
	assert.Equal(t, "agent/ax-cleanup", st.Branch)
	assert.Equal(t, 2, st.Runs)
}

func TestPaths_ReadStatus_Bad_InvalidJSON(t *testing.T) {
	wsDir := t.TempDir()
	require.True(t, agentic.LocalFs().WriteAtomic(agentic.WorkspaceStatusPath(wsDir), "{not-json").OK)

	_, err := ReadStatus(wsDir)
	assert.Error(t, err)
}

func TestPaths_WriteStatus_Ugly_AtomicOverwrite(t *testing.T) {
	wsDir := t.TempDir()

	WriteStatus(wsDir, &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "First run",
	})
	WriteStatus(wsDir, &WorkspaceStatus{
		Status:    "completed",
		Agent:     "claude",
		Repo:      "go-io",
		Task:      "Second run",
		Branch:    "agent/ax-cleanup",
		StartedAt: time.Now(),
		Runs:      3,
	})

	st, err := ReadStatus(wsDir)
	require.NoError(t, err)
	assert.Equal(t, "completed", st.Status)
	assert.Equal(t, "claude", st.Agent)
	assert.Equal(t, "agent/ax-cleanup", st.Branch)
	assert.Equal(t, 3, st.Runs)
}
