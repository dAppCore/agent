// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/forge"
)

func TestAutopr_AutoCreatePR_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	remoteDir := core.JoinPath(root, "remote.git")
	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "init", "--bare", remoteDir).OK)

	seedDir := core.JoinPath(root, "seed")
	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "clone", remoteDir, seedDir).OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), seedDir, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), seedDir, "git", "config", "user.email", "test@example.com").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), seedDir, "git", "checkout", "-b", "dev").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(seedDir, "README.md"), "# seed\n").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), seedDir, "git", "add", ".").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), seedDir, "git", "commit", "-m", "seed").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), seedDir, "git", "push", "-u", "origin", "dev").OK)
	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "--git-dir", remoteDir, "symbolic-ref", "HEAD", "refs/heads/dev").OK)

	workspaceDir := core.JoinPath(root, "workspace", "core", "go-io", "task-5")
	repoDir := WorkspaceRepoDir(workspaceDir)
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "clone", remoteDir, repoDir).OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@example.com").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", "agent/fix-tests", "origin/dev").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# seed\nfeature change\n").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "add", "README.md").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "feature").OK)

	gitConfigPath := core.JoinPath(root, "gitconfig")
	gitConfig := core.Concat(
		"[url \"file://", remoteDir, "\"]\n",
		"\tinsteadOf = ssh://git@forge.lthn.ai:2223/core/go-io.git\n",
	)
	core.RequireTrue(t, fs.Write(gitConfigPath, gitConfig).OK)
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfigPath)

	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "/api/v1/repos/core/go-io/pulls", r.URL.Path)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		requestBody = bodyResult.Value.(string)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"number":17,"html_url":"https://forge.test/core/go-io/pulls/17"}`))
	}))
	defer server.Close()

	status := &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-io",
		Org:       "core",
		Task:      "Fix tests",
		Branch:    "agent/fix-tests",
		Issue:     42,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	core.RequireNoError(t, writeStatus(workspaceDir, status))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(server.URL, "test-token"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.autoCreatePR(workspaceDir)

	statusResult := ReadStatusResult(workspaceDir)
	updated, ok := workspaceStatusValue(statusResult)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "https://forge.test/core/go-io/pulls/17", updated.PRURL)
	core.AssertContains(t, requestBody, `"head":"agent/fix-tests"`)
	core.AssertContains(t, requestBody, `"base":"dev"`)
	core.AssertContains(t, requestBody, `"title":"[agent/codex] Fix tests"`)

	remoteHeads := testCore.Process().RunIn(context.Background(), repoDir, "git", "ls-remote", "--heads", remoteDir, "agent/fix-tests")
	core.RequireTrue(t, remoteHeads.OK)
	core.AssertNotContains(t, remoteHeads.Value.(string), "agent/fix-tests")
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
	core.AssertNotPanics(t, func() {
		s.autoCreatePR(wsNoStatus)
	})

	// Empty branch → early return
	wsNoBranch := core.JoinPath(root, "ws-no-branch")
	fs.EnsureDir(wsNoBranch)
	fs.Write(core.JoinPath(wsNoBranch, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
		Status: "completed", Agent: "codex", Repo: "go-io", Branch: "",
	}))
	core.AssertNotPanics(t, func() {
		s.autoCreatePR(wsNoBranch)
	})

	// Empty repo → early return
	wsNoRepo := core.JoinPath(root, "ws-no-repo")
	fs.EnsureDir(wsNoRepo)
	fs.Write(core.JoinPath(wsNoRepo, "status.json"), core.JSONMarshalString(&WorkspaceStatus{
		Status: "completed", Agent: "codex", Repo: "", Branch: "agent/fix-tests",
	}))
	core.AssertNotPanics(t, func() {
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
	core.AssertNotPanics(t, func() {
		s.autoCreatePR(wsDir)
	})
}

func TestAutopr_CleanupForgeBranch_Good_DeletesRemoteBranch(t *testing.T) {
	remoteDir := core.JoinPath(t.TempDir(), "remote.git")
	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "init", "--bare", remoteDir).OK)

	repoDir := core.JoinPath(t.TempDir(), "repo")
	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "clone", remoteDir, repoDir).OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@example.com").OK)

	branch := "agent/fix-branch"
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", branch).OK)
	fs.Write(core.JoinPath(repoDir, "README.md"), "# test")
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "init").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "push", "-u", "origin", branch).OK)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	core.AssertTrue(t, s.cleanupForgeBranch(context.Background(), repoDir, remoteDir, branch))

	remoteHeads := testCore.Process().RunIn(context.Background(), repoDir, "git", "ls-remote", "--heads", remoteDir, branch)
	core.RequireTrue(t, remoteHeads.OK)
	core.AssertNotContains(t, remoteHeads.Value.(string), branch)
}
