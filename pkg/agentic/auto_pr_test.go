// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestAutopr_AutoCreatePR_Good(t *testing.T) {
	t.Skip("needs real git + forge integration")
}

func TestAutopr_AutoCreatePR_Bad(t *testing.T) {
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

func TestAutopr_AutoCreatePR_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

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
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// git log origin/dev..HEAD will fail (no origin remote) → early return
	assert.NotPanics(t, func() {
		s.autoCreatePR(wsDir)
	})
}
