// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

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
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]any{
				"number":   12,
				"html_url": "https://forge.test/core/test-repo/pulls/12",
				"title":    body.Title,
				"head":     map[string]any{"ref": body.Head},
				"base":     map[string]any{"ref": body.Base},
			})
			return
		}
		// GET — list PRs
		json.NewEncoder(w).Encode([]map[string]any{})
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
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
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
		json.NewEncoder(w).Encode(map[string]any{"message": "internal error"})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
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
	wsDir := filepath.Join(root, "workspace", "test-ws")
	repoDir := filepath.Join(wsDir, "repo")
	require.NoError(t, exec.Command("git", "init", "-b", "main", repoDir).Run())
	gitCmd := exec.Command("git", "config", "user.name", "Test")
	gitCmd.Dir = repoDir
	gitCmd.Run()
	gitCmd = exec.Command("git", "config", "user.email", "test@test.com")
	gitCmd.Dir = repoDir
	gitCmd.Run()

	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-bug",
		Task:   "Fix the login bug",
	}))

	s := &PrepSubsystem{
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

	wsDir := filepath.Join(root, "workspace", "test-ws-2")
	repoDir := filepath.Join(wsDir, "repo")
	require.NoError(t, exec.Command("git", "init", "-b", "main", repoDir).Run())
	gitCmd := exec.Command("git", "config", "user.name", "Test")
	gitCmd.Dir = repoDir
	gitCmd.Run()
	gitCmd = exec.Command("git", "config", "user.email", "test@test.com")
	gitCmd.Dir = repoDir
	gitCmd.Run()

	require.NoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix",
		Task:   "Default task",
	}))

	s := &PrepSubsystem{
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
		forge:      forge.NewForge(srv.URL, "test-token"),
		forgeURL:   srv.URL,
		forgeToken: "test-token",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	s.commentOnIssue(context.Background(), "core", "go-io", 42, "Test comment")
	assert.True(t, commentPosted)
}
