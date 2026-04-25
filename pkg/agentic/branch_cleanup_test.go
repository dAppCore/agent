// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupBranch_Good_DeletesAgentBranch(t *testing.T) {
	branch := "agent/fix-tests"
	_, remoteDir := newCleanupRemoteRepo(t)
	seedCleanupRemoteBranch(t, remoteDir, branch)

	server, state := newCleanupForgeServer(t, remoteDir, branch, http.StatusNoContent, false)
	s := newCleanupPrep(server.URL)

	result := s.cleanupBranch(context.Background(), "core/go-io", branch)
	require.True(t, result.OK)
	assert.Equal(t, 1, state.deleteCalls)
	assert.False(t, cleanupRemoteBranchExists(remoteDir, branch))
}

func TestCleanupBranch_Bad_RefuseProtected(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	s := newCleanupPrep(server.URL)
	result := s.cleanupBranch(context.Background(), "core/go-io", "main")

	require.False(t, result.OK)
	err, _ := result.Value.(error)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to delete protected branch")
	assert.False(t, called)
}

func TestCleanupBranch_Ugly_DeleteFailsForge(t *testing.T) {
	branch := "agent/fix-tests"
	server, state := newCleanupForgeServer(t, "", branch, http.StatusBadGateway, false)
	s := newCleanupPrep(server.URL)

	result := s.cleanupBranch(context.Background(), "core/go-io", branch)

	require.False(t, result.OK)
	err, _ := result.Value.(error)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete branch")
	assert.Equal(t, 1, state.deleteCalls)
}

func TestCleanupBranch_Good_CreatePRSuccessPathDeletesBranch(t *testing.T) {
	branch := "agent/fix-tests"
	root := t.TempDir()
	setTestWorkspace(t, root)

	remoteRoot, remoteDir := newCleanupRemoteRepo(t)
	writeCleanupGitRewrite(t, remoteRoot)

	server, state := newCleanupForgeServer(t, remoteDir, branch, http.StatusNoContent, true)
	s := newCleanupPrep(server.URL)

	workspaceDir := core.JoinPath(root, "workspace", "test-ws")
	repoDir := WorkspaceRepoDir(workspaceDir)
	createCleanupWorkspaceRepo(t, repoDir, branch)

	require.NoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Fix branch cleanup",
		Branch: branch,
		Runs:   1,
	}))

	_, output, err := s.createPR(context.Background(), nil, CreatePRInput{Workspace: "test-ws"})
	require.NoError(t, err)
	assert.True(t, output.Success)
	assert.Equal(t, "https://forge.test/core/go-io/pulls/42", output.PRURL)
	assert.Equal(t, 1, state.prCalls)
	assert.Equal(t, 1, state.deleteCalls)
	assert.False(t, cleanupRemoteBranchExists(remoteDir, branch))
}

func TestCleanupBranch_Good_CmdCompleteSuccessPathDeletesBranch(t *testing.T) {
	branch := "agent/fix-tests"
	root := t.TempDir()
	setTestWorkspace(t, root)

	_, remoteDir := newCleanupRemoteRepo(t)
	seedCleanupRemoteBranch(t, remoteDir, branch)

	server, state := newCleanupForgeServer(t, remoteDir, branch, http.StatusNoContent, false)
	c := core.New()
	c.Action("noop", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{OK: true}
	})
	c.Task("agent.completion", core.Task{
		Description: "cleanup branch",
		Steps: []core.Step{
			{Action: "noop"},
		},
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		forge:          forge.NewForge(server.URL, "test-token"),
		forgeURL:       server.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	workspaceDir := core.JoinPath(root, "workspace", "test-ws")
	require.True(t, fs.EnsureDir(workspaceDir).OK)
	require.NoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Org:    "core",
		Branch: branch,
		PRURL:  "https://forge.test/core/go-io/pulls/42",
	}))

	result := s.cmdComplete(core.NewOptions(core.Option{Key: "workspace", Value: workspaceDir}))
	require.True(t, result.OK)
	assert.Equal(t, 1, state.deleteCalls)
	assert.False(t, cleanupRemoteBranchExists(remoteDir, branch))
}

type cleanupForgeServerState struct {
	deleteCalls int
	prCalls     int
}

func newCleanupPrep(serverURL string) *PrepSubsystem {
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(serverURL, "test-token"),
		forgeURL:       serverURL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

func newCleanupRemoteRepo(t *testing.T) (string, string) {
	t.Helper()

	remoteRoot := t.TempDir()
	remoteDir := core.JoinPath(remoteRoot, "core", "go-io.git")
	require.True(t, fs.EnsureDir(core.PathDir(remoteDir)).OK)
	require.True(t, testCore.Process().Run(context.Background(), "git", "init", "--bare", remoteDir).OK)
	return remoteRoot, remoteDir
}

func seedCleanupRemoteBranch(t *testing.T, remoteDir, branch string) {
	t.Helper()

	repoDir := core.JoinPath(t.TempDir(), "seed-repo")
	createCleanupWorkspaceRepo(t, repoDir, branch)

	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "remote", "add", "origin", remoteDir).OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "push", "origin", "dev").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "push", "origin", branch).OK)
}

func createCleanupWorkspaceRepo(t *testing.T, repoDir, branch string) {
	t.Helper()

	require.True(t, testCore.Process().Run(context.Background(), "git", "init", "-b", "dev", repoDir).OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com").OK)

	require.True(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# cleanup").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "initial").OK)

	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", branch).OK)
	require.True(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# cleanup branch").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	require.True(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "branch work").OK)
}

func cleanupRemoteBranchExists(remoteDir, branch string) bool {
	ref := core.Concat("refs/heads/", branch)
	return testCore.Process().Run(context.Background(), "git", "--git-dir", remoteDir, "show-ref", "--verify", "--quiet", ref).OK
}

func writeCleanupGitRewrite(t *testing.T, remoteRoot string) {
	t.Helper()

	configPath := core.JoinPath(t.TempDir(), "gitconfig")
	configBody := core.Sprintf("[url \"%s\"]\n\tinsteadOf = ssh://git@forge.lthn.ai:2223/\n", core.Concat("file://", remoteRoot, "/"))
	require.True(t, fs.Write(configPath, configBody).OK)
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
}

func newCleanupForgeServer(t *testing.T, remoteDir, branch string, deleteStatus int, createPR bool) (*httptest.Server, *cleanupForgeServerState) {
	t.Helper()

	state := &cleanupForgeServerState{}
	deletePath := core.Concat("/api/v1/repos/core/go-io/branches/", url.PathEscape(branch))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case createPR && r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/core/go-io/pulls":
			state.prCalls++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"number":   42,
				"html_url": "https://forge.test/core/go-io/pulls/42",
			})))
		case r.Method == http.MethodDelete && r.URL.Path == deletePath:
			state.deleteCalls++
			if deleteStatus != http.StatusNoContent {
				http.Error(w, "delete failed", deleteStatus)
				return
			}
			if remoteDir != "" {
				ref := core.Concat("refs/heads/", branch)
				deleteResult := testCore.Process().Run(context.Background(), "git", "--git-dir", remoteDir, "update-ref", "-d", ref)
				if !deleteResult.OK {
					http.Error(w, "delete ref failed", http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	return server, state
}
