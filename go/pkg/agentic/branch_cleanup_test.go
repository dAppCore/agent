// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestCleanupBranch_Good_DeletesAgentBranch(t *testing.T) {
	branch := "agent/fix-tests"
	_, remoteDir := newCleanupRemoteRepo(t)
	seedCleanupRemoteBranch(t, remoteDir, branch)

	server, state := newCleanupForgeServer(t, remoteDir, branch, http.StatusNoContent, false)
	s := newCleanupPrep(server.URL)

	result := s.cleanupBranch(context.Background(), "core/go-io", branch)
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, 1, state.deleteCalls)
	core.AssertFalse(t, cleanupRemoteBranchExists(remoteDir, branch))
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

	core.AssertFalse(t, result.OK)
	err, _ := result.Value.(error)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "refusing to delete protected branch")
	core.AssertFalse(t, called)
}

func TestCleanupBranch_Ugly_DeleteFailsForge(t *testing.T) {
	branch := "agent/fix-tests"
	server, state := newCleanupForgeServer(t, "", branch, http.StatusBadGateway, false)
	s := newCleanupPrep(server.URL)

	result := s.cleanupBranch(context.Background(), "core/go-io", branch)

	core.AssertFalse(t, result.OK)
	err, _ := result.Value.(error)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to delete branch")
	core.AssertEqual(t, 1, state.deleteCalls)
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

	core.RequireNoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
		Org:    "core",
		Task:   "Fix branch cleanup",
		Branch: branch,
		Runs:   1,
	}))

	_, output, err := s.createPR(context.Background(), nil, CreatePRInput{Workspace: "test-ws"})
	core.RequireNoError(t, err)
	core.AssertTrue(t, output.Success)
	core.AssertEqual(t, "https://forge.test/core/go-io/pulls/42", output.PRURL)
	core.AssertEqual(t, 1, state.prCalls)
	core.AssertEqual(t, 1, state.deleteCalls)
	core.AssertFalse(t, cleanupRemoteBranchExists(remoteDir, branch))
}

func TestCleanupBranch_Good_CmdCompleteSuccessPathDeletesBranch(t *testing.T) {
	branch := "agent/fix-tests"
	root := t.TempDir()
	setTestWorkspace(t, root)

	_, remoteDir := newCleanupRemoteRepo(t)
	seedCleanupRemoteBranch(t, remoteDir, branch)

	server, state := newCleanupForgeServer(t, remoteDir, branch, http.StatusNoContent, false)
	c := core.New()
	c.Action("test.noop", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{OK: true}
	})
	c.Task("agent.completion", core.Task{
		Description: "cleanup branch",
		Steps: []core.Step{
			{Action: "test.noop"},
		},
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		forge:          newForgeClient(server.URL, "test-token"),
		forgeURL:       server.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	workspaceDir := core.JoinPath(root, "workspace", "test-ws")
	core.RequireTrue(t, fs.EnsureDir(workspaceDir).OK)
	core.RequireNoError(t, writeStatus(workspaceDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Org:    "core",
		Branch: branch,
		PRURL:  "https://forge.test/core/go-io/pulls/42",
	}))

	result := s.cmdComplete(core.NewOptions(core.Option{Key: "workspace", Value: workspaceDir}))
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, 1, state.deleteCalls)
	core.AssertFalse(t, cleanupRemoteBranchExists(remoteDir, branch))
}

type cleanupForgeServerState struct {
	deleteCalls int
	prCalls     int
}

func newCleanupPrep(serverURL string) *PrepSubsystem {
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(serverURL, "test-token"),
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
	core.RequireTrue(t, fs.EnsureDir(core.PathDir(remoteDir)).OK)
	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "init", "--bare", remoteDir).OK)
	return remoteRoot, remoteDir
}

func seedCleanupRemoteBranch(t *testing.T, remoteDir, branch string) {
	t.Helper()

	repoDir := core.JoinPath(t.TempDir(), "seed-repo")
	createCleanupWorkspaceRepo(t, repoDir, branch)

	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "remote", "add", "origin", remoteDir).OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "push", "origin", "dev").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "push", "origin", branch).OK)
}

func createCleanupWorkspaceRepo(t *testing.T, repoDir, branch string) {
	t.Helper()

	core.RequireTrue(t, testCore.Process().Run(context.Background(), "git", "init", "-b", "dev", repoDir).OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com").OK)

	core.RequireTrue(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# cleanup").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "initial").OK)

	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "checkout", "-b", branch).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(repoDir, "README.md"), "# cleanup branch").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".").OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "branch work").OK)
}

func cleanupRemoteBranchExists(remoteDir, branch string) bool {
	ref := core.Concat("refs/heads/", branch)
	return testCore.Process().Run(context.Background(), "git", "--git-dir", remoteDir, "show-ref", "--verify", "--quiet", ref).OK
}

func writeCleanupGitRewrite(t *testing.T, remoteRoot string) {
	t.Helper()

	configPath := core.JoinPath(t.TempDir(), "gitconfig")
	configBody := core.Sprintf("[url \"%s\"]\n\tinsteadOf = ssh://git@forge.lthn.ai:2223/\n", core.Concat("file://", remoteRoot, "/"))
	core.RequireTrue(t, fs.Write(configPath, configBody).OK)
	t.Setenv("GIT_CONFIG_GLOBAL", configPath)
}

func newCleanupForgeServer(t *testing.T, remoteDir, branch string, deleteStatus int, createPR bool) (*httptest.Server, *cleanupForgeServerState) {
	t.Helper()

	state := &cleanupForgeServerState{}
	deletePath := core.Concat("/api/v1/repos/core/go-io/branches/", url.PathEscape(branch))
	deletePathDecoded := core.Concat("/api/v1/repos/core/go-io/branches/", branch)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case createPR && r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/core/go-io/pulls":
			state.prCalls++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"number":   42,
				"html_url": "https://forge.test/core/go-io/pulls/42",
			})))
		case r.Method == http.MethodDelete && (r.URL.Path == deletePath || r.URL.Path == deletePathDecoded || r.URL.EscapedPath() == deletePath):
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
