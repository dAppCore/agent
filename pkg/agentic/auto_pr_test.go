// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoPR_AutoCreatePR_Good(t *testing.T) {
	t.Skip("needs real git + forge integration")
}

func TestAutoPR_AutoCreatePR_Bad(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// No status file → early return (no panic)
	wsNoStatus := core.JoinPath(root, "ws-no-status")
	require.NoError(t, os.MkdirAll(wsNoStatus, 0o755))
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsNoStatus)
	})

	// Empty branch → early return
	wsNoBranch := core.JoinPath(root, "ws-no-branch")
	require.NoError(t, os.MkdirAll(wsNoBranch, 0o755))
	st := &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
		Branch: "",
	}
	data, err := json.MarshalIndent(st, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(core.JoinPath(wsNoBranch, "status.json"), data, 0o644))
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsNoBranch)
	})

	// Empty repo → early return
	wsNoRepo := core.JoinPath(root, "ws-no-repo")
	require.NoError(t, os.MkdirAll(wsNoRepo, 0o755))
	st2 := &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "",
		Branch: "agent/fix-tests",
	}
	data2, err := json.MarshalIndent(st2, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(core.JoinPath(wsNoRepo, "status.json"), data2, 0o644))
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsNoRepo)
	})
}

func TestAutoPR_AutoCreatePR_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Set up a real git repo with no commits ahead of origin/dev
	wsDir := core.JoinPath(root, "ws-no-ahead")
	repoDir := core.JoinPath(wsDir, "repo")
	require.NoError(t, os.MkdirAll(repoDir, 0o755))

	// Init the repo
	cmd := exec.Command("git", "init", "-b", "dev", repoDir)
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	require.NoError(t, os.WriteFile(core.JoinPath(repoDir, "README.md"), []byte("# test"), 0o644))
	cmd = exec.Command("git", "-C", repoDir, "add", ".")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "commit", "-m", "init")
	require.NoError(t, cmd.Run())

	// Write status with valid branch + repo
	st := &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Branch:    "agent/fix-tests",
		StartedAt: time.Now(),
	}
	data, err := json.MarshalIndent(st, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(core.JoinPath(wsDir, "status.json"), data, 0o644))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// git log origin/dev..HEAD will fail (no origin remote) → early return
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsDir)
	})
}
