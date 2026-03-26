// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
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
	fs.EnsureDir(wsNoStatus)
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsNoStatus)
	})

	// Empty branch → early return
	wsNoBranch := core.JoinPath(root, "ws-no-branch")
	fs.EnsureDir(wsNoBranch)
	fs.Write(core.JoinPath(wsNoBranch, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
		Status: "completed", Agent: "codex", Repo: "go-io", Branch: "",
	}))
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsNoBranch)
	})

	// Empty repo → early return
	wsNoRepo := core.JoinPath(root, "ws-no-repo")
	fs.EnsureDir(wsNoRepo)
	fs.Write(core.JoinPath(wsNoRepo, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
		Status: "completed", Agent: "codex", Repo: "", Branch: "agent/fix-tests",
	}))
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
	fs.EnsureDir(repoDir)

	// Init the repo
	cmd := exec.Command("git", "init", "-b", "dev", repoDir)
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", repoDir, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
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
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

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
