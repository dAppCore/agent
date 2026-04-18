// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutopr_AutoCreatePR_Good(t *testing.T) {
	t.Skip("needs real git + forge integration")
}

func TestAutopr_AutoCreatePR_Bad(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
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

func TestAutopr_AutoCreatePR_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Set up a real git repo with no commits ahead of origin/dev
	wsDir := core.JoinPath(root, "ws-no-ahead")
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)

	// Init the repo
	testCore.Process().Run(context.Background(), "git", "init", "-b", "dev", repoDir)
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com")

	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "init")

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
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// git log origin/dev..HEAD will fail (no origin remote) → early return
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsDir)
	})
}

func TestAutopr_CleanupForgeBranch_Good_DeletesRemoteBranch(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote.git")
	require.True(t, testCore.Process().Run(context.Background(), "git", "init", "--bare", remoteDir).OK)

	repoDir := core.JoinPath(t.TempDir(), "repo")
	require.True(t, testCore.Process().Run(context.Background(), "git", "clone", remoteDir, repoDir).OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@example.com").OK)

	branch := "agent/fix-branch"
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", branch).OK)
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "init").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "push", "-u", "origin", branch).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.True(t, s.cleanupForgeBranch(context.Background(), repoDir, remoteDir, branch))

	remoteHeads := testCore.Process().RunIn(context.Background(), repoDir, "git", "ls-remote", "--heads", remoteDir, branch)
	require.True(t, remoteHeads.OK)
	assert.NotContains(t, remoteHeads.Value.(string), branch)
}
