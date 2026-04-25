// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoSync_OnWorkspacePushed_Good(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "new.go", "package main\n")

	s.registerRepoSyncSupport()
	c.ACTION(messages.WorkspacePushed{
		Repo:   "test-repo",
		Branch: "main",
		Org:    "core",
	})

	assert.True(t, fs.Exists(core.JoinPath(repoDir, "new.go")))
	assert.Equal(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
}

func TestRepoSync_OnWorkspacePushed_Bad(t *testing.T) {
	s, _, _ := repoSyncTestPrep(t)
	result := s.onWorkspacePushed(context.Background(), messages.WorkspacePushed{
		Repo:   "missing-repo",
		Branch: "main",
		Org:    "core",
	})
	assert.False(t, result.OK)
}

func TestRepoSync_OnWorkspacePushed_Ugly(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	require.True(t, c.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", "feature/wip").OK)

	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "edge.go", "package edge\n")

	result := s.onWorkspacePushed(context.Background(), messages.WorkspacePushed{
		Repo:   "test-repo",
		Branch: "main",
		Org:    "core",
	})
	require.True(t, result.OK)

	assert.Equal(t, "main", repoSyncGitOutput(t, c, repoDir, "rev-parse", "--abbrev-ref", "HEAD"))
	assert.Equal(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
	assert.True(t, fs.Exists(core.JoinPath(repoDir, "edge.go")))
}

func TestRepoSync_BackgroundFetch_Good(t *testing.T) {
	s, c, root := repoSyncTestPrep(t)
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"repos:\n",
		"  - test-repo\n",
	)).OK)

	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "fetched.go", "package fetched\n")

	assert.Equal(t, fetchLoopDefaultInterval, s.fetchLoopInterval())

	s.fetchRegisteredRepos(context.Background())

	assert.Equal(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "origin/main"))
	assert.NotEqual(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
	assert.False(t, fs.Exists(core.JoinPath(repoDir, "fetched.go")))
}

func TestRepoSync_Command_Good(t *testing.T) {
	s, c, _ := repoSyncTestPrep(t)
	remoteDir, repoDir := repoSyncCreateTrackedRepo(t, c, s.codePath, "core", "test-repo")
	_, remoteHead := repoSyncPushCommit(t, c, remoteDir, "main", "command.go", "package command\n")

	s.registerCommands(context.Background())
	commandResult := c.Command("repo/sync")
	require.True(t, commandResult.OK)

	output := repoSyncCaptureStdout(t, func() {
		result := commandResult.Value.(*core.Command).Run(core.NewOptions(
			core.Option{Key: "repo", Value: "test-repo"},
			core.Option{Key: "reset", Value: true},
		))
		require.True(t, result.OK)

		commandOutput, ok := result.Value.(RepoSyncCommandOutput)
		require.True(t, ok)
		assert.True(t, commandOutput.Success)
		assert.Equal(t, 1, commandOutput.Count)
	})

	assert.Contains(t, output, "fetched core/test-repo@main")
	assert.Contains(t, output, "count: 1")
	assert.True(t, fs.Exists(core.JoinPath(repoDir, "command.go")))
	assert.Equal(t, remoteHead, repoSyncGitOutput(t, c, repoDir, "rev-parse", "HEAD"))
}

func repoSyncTestPrep(t *testing.T) (*PrepSubsystem, *core.Core, string) {
	t.Helper()

	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New(
		core.WithService(ProcessRegister),
	)
	require.True(t, c.ServiceStartup(context.Background(), nil).OK)
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
	require.True(t, fs.EnsureDir(remoteDir).OK)
	require.True(t, c.Process().Run(context.Background(), "git", "init", "--bare", remoteDir).OK)

	orgDir := core.JoinPath(codePath, org)
	require.True(t, fs.EnsureDir(orgDir).OK)
	require.True(t, c.Process().RunIn(context.Background(), orgDir, "git", "clone", remoteDir, repo).OK)

	repoDir := core.JoinPath(orgDir, repo)
	repoSyncConfigureGit(t, c, repoDir)
	require.True(t, c.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", "main").OK)
	require.True(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# tracked repo\n").OK)
	require.True(t, c.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	require.True(t, c.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "init").OK)
	require.True(t, c.Process().RunIn(context.Background(), repoDir, "git", "push", "-u", "origin", "main").OK)

	return remoteDir, repoDir
}

func repoSyncPushCommit(t *testing.T, c *core.Core, remoteDir, branch, fileName, content string) (string, string) {
	t.Helper()

	cloneParent := t.TempDir()
	cloneDir := core.JoinPath(cloneParent, "push-clone")
	require.True(t, c.Process().RunIn(context.Background(), cloneParent, "git", "clone", remoteDir, "push-clone").OK)
	repoSyncConfigureGit(t, c, cloneDir)
	require.True(t, c.Process().RunIn(context.Background(), cloneDir, "git", "checkout", branch).OK)
	require.True(t, fs.Write(core.JoinPath(cloneDir, fileName), content).OK)
	require.True(t, c.Process().RunIn(context.Background(), cloneDir, "git", "add", ".").OK)
	require.True(t, c.Process().RunIn(context.Background(), cloneDir, "git", "commit", "-m", "agent work").OK)
	require.True(t, c.Process().RunIn(context.Background(), cloneDir, "git", "push", "origin", branch).OK)

	return cloneDir, repoSyncGitOutput(t, c, cloneDir, "rev-parse", "HEAD")
}

func repoSyncConfigureGit(t *testing.T, c *core.Core, repoDir string) {
	t.Helper()
	require.True(t, c.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	require.True(t, c.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@example.com").OK)
}

func repoSyncGitOutput(t *testing.T, c *core.Core, repoDir string, args ...string) string {
	t.Helper()
	result := c.Process().RunIn(context.Background(), repoDir, "git", args...)
	require.True(t, result.OK)
	return core.Trim(result.Value.(string))
}

func repoSyncCaptureStdout(t *testing.T, run func()) string {
	t.Helper()

	old := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	defer func() {
		os.Stdout = old
	}()

	run()

	require.NoError(t, writer.Close())
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	return string(data)
}
