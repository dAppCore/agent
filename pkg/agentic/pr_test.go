// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	forge_types "dappco.re/go/core/forge/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPRForgeServer creates a Forge API mock that handles PR creation and comments.
func mockPRForgeServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Create PR endpoint — returns Forgejo-compatible JSON
	mux.HandleFunc("/api/v1/repos/core/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var body forge_types.CreatePullRequestOption
			bodyStr := core.ReadAll(r.Body)
			core.JSONUnmarshalString(bodyStr.Value.(string), &body)
			w.WriteHeader(201)
			w.Write([]byte(core.JSONMarshalString(map[string]any{
				"number":   12,
				"html_url": "https://forge.test/core/test-repo/pulls/12",
				"title":    body.Title,
				"head":     map[string]any{"ref": body.Head},
				"base":     map[string]any{"ref": body.Base},
			})))
			return
		}
		// GET — list PRs
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	})

	// Issue comments
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && containsStr(r.URL.Path, "/comments") {
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(200)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// --- forgeCreatePR ---

func TestPr_ForgeCreatePR_Good_Success(t *testing.T) {
	srv := mockPRForgeServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	prURL, prNum, err := s.forgeCreatePR(
		context.Background(),
		"core", "test-repo",
		"agent/fix-bug", "dev",
		"Fix the login bug", "PR body text",
	)
	require.NoError(t, err)
	assert.Equal(t, 12, prNum)
	assert.Contains(t, prURL, "pulls/12")
}

func TestPr_ForgeCreatePR_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(core.JSONMarshalString(map[string]any{"message": "internal error"})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.forgeCreatePR(
		context.Background(),
		"core", "test-repo",
		"agent/fix", "dev",
		"Title", "Body",
	)
	assert.Error(t, err)
}

// --- createPR (MCP tool) ---

func TestPr_CreatePR_Bad_NoWorkspace(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace is required")
}

func TestPr_CreatePR_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken: "",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Forge token")
}

func TestPr_CreatePR_Bad_WorkspaceNotFound(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "nonexistent-workspace",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace not found")
}

func TestPr_CreatePR_Good_DryRun(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Create workspace with repo/.git
	wsDir := core.JoinPath(root, "workspace", "test-ws")
	repoDir := core.JoinPath(wsDir, "repo")
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", repoDir)
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com")

	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-bug",
		Task:   "Fix the login bug",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws",
		DryRun:    true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "agent/fix-bug", out.Branch)
	assert.Equal(t, "go-io", out.Repo)
	assert.Equal(t, "Fix the login bug", out.Title)
}

func TestPr_CreatePR_Good_CustomTitle(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "test-ws-2")
	repoDir := core.JoinPath(wsDir, "repo")
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", repoDir)
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com")

	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix",
		Task:   "Default task",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws-2",
		Title:     "Custom PR title",
		DryRun:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Custom PR title", out.Title)
}

// --- listPRs ---

func TestPr_ListPRs_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken: "",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, _, err := s.listPRs(context.Background(), nil, ListPRsInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Forge token")
}

// --- commentOnIssue ---

func TestPr_CommentOnIssue_Good_PostsComment(t *testing.T) {
	commentPosted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			commentPosted = true
			w.WriteHeader(201)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.commentOnIssue(context.Background(), "core", "go-io", 42, "Test comment")
	assert.True(t, commentPosted)
}

// --- buildPRBody ---

func TestPr_BuildPRBody_Good(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Task:   "Fix the login bug",
		Agent:  "codex",
		Branch: "agent/fix-login",
		Issue:  42,
		Runs:   3,
	}
	body := s.buildPRBody(st)
	assert.Contains(t, body, "## Summary")
	assert.Contains(t, body, "Fix the login bug")
	assert.Contains(t, body, "Closes #42")
	assert.Contains(t, body, "**Agent:** codex")
	assert.Contains(t, body, "**Runs:** 3")
}

func TestPr_BuildPRBody_Bad(t *testing.T) {
	// Empty status struct
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{}
	body := s.buildPRBody(st)
	assert.Contains(t, body, "## Summary")
	assert.Contains(t, body, "**Agent:**")
	assert.NotContains(t, body, "Closes #")
}

func TestPr_BuildPRBody_Ugly(t *testing.T) {
	// Very long task string
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	longTask := strings.Repeat("This is a very long task description. ", 100)
	st := &WorkspaceStatus{
		Task:  longTask,
		Agent: "claude",
		Runs:  1,
	}
	body := s.buildPRBody(st)
	assert.Contains(t, body, "## Summary")
	assert.Contains(t, body, "very long task")
}

// --- commentOnIssue Bad/Ugly ---

func TestPr_CommentOnIssue_Bad(t *testing.T) {
	// Forge returns error (500)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Should not panic even on server error
	assert.NotPanics(t, func() {
		s.commentOnIssue(context.Background(), "core", "go-io", 42, "Test comment")
	})
}

func TestPr_CommentOnIssue_Ugly(t *testing.T) {
	// Very long comment body
	commentPosted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			commentPosted = true
			w.WriteHeader(201)
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	longComment := strings.Repeat("This is a very long comment with details. ", 1000)
	s.commentOnIssue(context.Background(), "core", "go-io", 42, longComment)
	assert.True(t, commentPosted)
}

// --- createPR Ugly ---

func TestPr_CreatePR_Ugly(t *testing.T) {
	// Workspace with no branch in status (auto-detect from git)
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "test-ws-ugly")
	repoDir := core.JoinPath(wsDir, "repo")
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", repoDir)
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com")

	// Need an initial commit so HEAD exists for branch detection
	fs.Write(core.JoinPath(repoDir, "README.md"), "# Test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "add", ".")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "commit", "-m", "init")

	// Write status with empty branch
	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "", // empty branch — should auto-detect
		Task:   "Fix something",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws-ugly",
		DryRun:    true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.NotEmpty(t, out.Branch, "branch should be auto-detected from git")
}

// --- forgeCreatePR Ugly ---

func TestPr_ForgeCreatePR_Ugly(t *testing.T) {
	// Server returns 201 with unexpected JSON
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.WriteHeader(201)
			w.Write([]byte(core.JSONMarshalString(map[string]any{
				"unexpected": "fields",
				"number":     0,
			})))
			return
		}
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	// Should not panic — may return zero values for missing fields
	assert.NotPanics(t, func() {
		_, _, _ = s.forgeCreatePR(
			context.Background(),
			"core", "test-repo",
			"agent/fix", "dev",
			"Title", "Body",
		)
	})
}

// --- listPRs Ugly ---

func TestPr_ListPRs_Ugly(t *testing.T) {
	// State filter "all"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if containsStr(r.URL.Path, "/repos") && !containsStr(r.URL.Path, "/pulls") {
			w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"name": "go-io", "full_name": "core/go-io"},
			})))
			return
		}
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, out, err := s.listPRs(context.Background(), nil, ListPRsInput{
		State: "all",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
}

// --- listRepoPRs Good/Bad/Ugly ---

func TestPr_ListRepoPRs_Good(t *testing.T) {
	// Specific repo with PRs
	srv := mockPRForgeServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	prs, err := s.listRepoPRs(context.Background(), "core", "test-repo", "open")
	require.NoError(t, err)
	// May be empty depending on mock, but should not error
	_ = prs
}

func TestPr_ListRepoPRs_Bad(t *testing.T) {
	// Forge returns error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	_, err := s.listRepoPRs(context.Background(), "core", "go-io", "open")
	assert.Error(t, err)
}

func TestPr_ListRepoPRs_Ugly(t *testing.T) {
	// Repo with no PRs
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	prs, err := s.listRepoPRs(context.Background(), "core", "empty-repo", "open")
	require.NoError(t, err)
	assert.Empty(t, prs)
}
