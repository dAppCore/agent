// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

func TestRepoSync_OnWorkspacePushed_Good_Case(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "new.go", "package main\n")

	s.registerRepoSyncSupport()
	c.ACTION(messages.WorkspacePushed{
		Repo:   "test-repo",
		Branch: "main",
		Org:    "core",
	})

	core.AssertTrue(t, fs.Exists(core.JoinPath(repoDir, "new.go")))
	core.AssertEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
}

func TestRepoSync_OnWorkspacePushed_Bad_Case(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)
	result := s.onWorkspacePushed(context.Background(), messages.WorkspacePushed{
		Repo:   "missing-repo",
		Branch: "main",
		Org:    "core",
	})
	core.AssertFalse(t, result.OK)
}

func TestRepoSync_OnWorkspacePushed_Ugly_Case(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", "feature/wip").OK)

	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "edge.go", "package edge\n")

	result := s.onWorkspacePushed(context.Background(), messages.WorkspacePushed{
		Repo:   "test-repo",
		Branch: "main",
		Org:    "core",
	})
	core.RequireTrue(t, result.OK)

	core.AssertEqual(t, "main", repoSyncGitOutput(t, c, repoDir, "rev-parse", "--abbrev-ref", "HEAD"))
	core.AssertEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
	core.AssertTrue(t, fs.Exists(core.JoinPath(repoDir, "edge.go")))
}

func TestRepoSync_BackgroundFetch_Good_Case(t *testing.T) {
	s, c, root := repoSyncTestPrep(t)
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"repos:\n",
		"  - test-repo\n",
	)).OK)

	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "fetched.go", "package fetched\n")

	core.AssertEqual(t, fetchLoopDefaultInterval, s.fetchLoopInterval())

	s.fetchRegisteredRepos(context.Background())

	core.AssertEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "origin/main"))
	core.AssertNotEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
	core.AssertFalse(t, fs.Exists(core.JoinPath(repoDir, "fetched.go")))
}

func TestRepoSync_Command_Good_Case(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "command.go", "package command\n")

	s.registerCommands(context.Background())
	commandResult := c.Command("repo/sync")
	core.RequireTrue(t, commandResult.OK)

	output := captureStdout(t, func() {
		result := commandResult.Value.(*core.Command).Run(core.NewOptions(
			core.Option{Key: "repo", Value: "test-repo"},
			core.Option{Key: "reset", Value: true},
		))
		core.RequireTrue(t, result.OK)

		commandOutput, ok := result.Value.(RepoSyncCommandOutput)
		core.RequireTrue(t, ok)
		core.AssertTrue(t, commandOutput.Success)
		core.AssertEqual(t, 1, commandOutput.Count)
	})

	core.AssertContains(t, output, "fetched core/test-repo@main")
	core.AssertContains(t, output, "count: 1")
	core.AssertTrue(t, fs.Exists(core.JoinPath(repoDir, "command.go")))
	core.AssertEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
}

func repoSyncTestPrep(t *testing.T) (*PrepSubsystem, *core.Core, string) {
	t.Helper()

	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New(
		core.WithService(ProcessRegister),
	)
	core.RequireTrue(t, c.ServiceStartup(context.Background(), nil).OK)
	t.Cleanup(func() {
		c.ServiceShutdown(context.Background())
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	return s, c, root
}

func repoSyncCreateTrackedRepo(t *testing.T, c *core.Core, codePath, org, repo string) (string, string) {
	t.Helper()

	remoteDir := core.JoinPath(t.TempDir(), "remote.git")
	core.RequireTrue(t, fs.EnsureDir(remoteDir).OK)
	core.RequireTrue(t, c.Process().Run(context.Background(), "git", "init", "--bare", remoteDir).OK)

	orgDir := core.JoinPath(codePath, org)
	core.RequireTrue(t, fs.EnsureDir(orgDir).OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), orgDir, "git", "clone", remoteDir, repo).OK)

	repoDir := core.JoinPath(orgDir, repo)
	repoSyncConfigureGit(t, c, repoDir)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", "main").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# tracked repo\n").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "init").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "push", "-u", "origin", "main").OK)

	return remoteDir, repoDir
}

func repoSyncPushCommit(t *testing.T, c *core.Core, remoteDir, branch, fileName, content string) (string, string) {
	t.Helper()

	cloneParent := t.TempDir()
	cloneDir := core.JoinPath(cloneParent, "push-clone")
	core.RequireTrue(t, c.Process().RunIn(context.Background(), cloneParent, "git", "clone", remoteDir, "push-clone").OK)
	repoSyncConfigureGit(t, c, cloneDir)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), cloneDir, "git", "checkout", branch).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(cloneDir, fileName), content).OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), cloneDir, "git", "add", ".").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), cloneDir, "git", "commit", "-m", "agent work").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), cloneDir, "git", "push", "origin", branch).OK)

	return cloneDir, repoSyncGitOutput(t, c, cloneDir, "rev-parse", "HEAD")
}

func repoSyncConfigureGit(t *testing.T, c *core.Core, repoDir string) {
	t.Helper()
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@example.com").OK)
}

func repoSyncGitOutput(t *testing.T, c *core.Core, repoDir string, args ...string) string {
	t.Helper()
	result := c.Process().RunIn(context.Background(), repoDir, "git", args...)
	core.RequireTrue(t, result.OK)
	return core.Trim(result.Value.(string))
}
