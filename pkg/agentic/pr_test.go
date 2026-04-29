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
	forge_types "dappco.re/go/forge/types"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
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
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prURL, prNum, err := s.forgeCreatePR(
		context.Background(),
		"core", "test-repo",
		"agent/fix-bug", "dev",
		"Fix the login bug", "PR body text",
	)
	core.RequireNoError(t, err)
	core.AssertEqual(t, 12, prNum)
	core.AssertContains(t, prURL, "pulls/12")
}

func TestPr_ForgeCreatePR_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(core.JSONMarshalString(map[string]any{"message": "internal error"})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.forgeCreatePR(
		context.Background(),
		"core", "test-repo",
		"agent/fix", "dev",
		"Title", "Body",
	)
	core.AssertError(t, err)
}

// --- createPR (MCP tool) ---

func TestPr_CreatePR_Bad_NoWorkspace(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "workspace is required")
}

func TestPr_CreatePR_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no Forge token")
}

func TestPr_CreatePR_Bad_WorkspaceNotFound(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "nonexistent-workspace",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "workspace not found")
}

func TestPr_CreatePR_Good_DryRun(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Create workspace with repo/.git
	wsDir := core.JoinPath(root, "workspace", "test-ws")
	repoDir := core.JoinPath(wsDir, "repo")
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", repoDir)
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com")

	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix-bug",
		Task:   "Fix the login bug",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws",
		DryRun:    true,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "agent/fix-bug", out.Branch)
	core.AssertEqual(t, "go-io", out.Repo)
	core.AssertEqual(t, "Fix the login bug", out.Title)
}

func TestPr_CreatePR_Good_CustomTitle(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "test-ws-2")
	repoDir := core.JoinPath(wsDir, "repo")
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", repoDir)
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.name", "Test")
	testCore.Process().RunIn(context.Background(), repoDir, "git", "config", "user.email", "test@test.com")

	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "agent/fix",
		Task:   "Default task",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws-2",
		Title:     "Custom PR title",
		DryRun:    true,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Custom PR title", out.Title)
}

func TestPr_ClosePR_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "/api/v1/repos/core/test-repo/pulls/7", r.URL.Path)

		bodyResult := core.ReadAll(r.Body)
		core.AssertTrue(t, bodyResult.OK)
		core.AssertContains(t, bodyResult.Value.(string), `"state":"closed"`)

		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 7,
			"state":  "closed",
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.closePR(context.Background(), nil, ClosePRInput{
		Repo:   "test-repo",
		Number: 7,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "core", out.Org)
	core.AssertEqual(t, "test-repo", out.Repo)
	core.AssertEqual(t, 7, out.Number)
	core.AssertEqual(t, "closed", out.State)
}

func TestPr_RegisterPRTools_Good_RegistersPRAliases(t *testing.T) {
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	s.registerPRTools(svc)

	server := svc.Server()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	core.RequireNoError(t, err)

	var toolNames []string
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	core.AssertContains(t, toolNames, "agentic_pr_get")
	core.AssertContains(t, toolNames, "pr_get")
	core.AssertContains(t, toolNames, "agentic_pr_list")
	core.AssertContains(t, toolNames, "pr_list")
	core.AssertContains(t, toolNames, "agentic_pr_merge")
	core.AssertContains(t, toolNames, "pr_merge")
	core.AssertContains(t, toolNames, "agentic_pr_close")
	core.AssertContains(t, toolNames, "pr_close")
}

func TestPr_PRGet_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodGet, r.Method)
		core.AssertEqual(t, "/api/v1/repos/core/test-repo/pulls/42", r.URL.Path)

		_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number":    42,
			"title":     "Fix login",
			"state":     "open",
			"mergeable": true,
			"html_url":  "https://forge.test/core/test-repo/pulls/42",
			"head":      map[string]any{"ref": "agent/fix-login"},
			"base":      map[string]any{"ref": "dev"},
			"user":      map[string]any{"login": "codex"},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.prGet(context.Background(), nil, PRGetInput{
		Repo:   "test-repo",
		Number: 42,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "test-repo", out.PR.Repo)
	core.AssertEqual(t, 42, out.PR.Number)
	core.AssertEqual(t, "Fix login", out.PR.Title)
	core.AssertEqual(t, "open", out.PR.State)
	core.AssertEqual(t, "agent/fix-login", out.PR.Branch)
}

func TestPr_PRGet_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.prGet(context.Background(), nil, PRGetInput{
		Repo:   "test-repo",
		Number: 42,
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no Forge token")
}

func TestPr_PRMerge_Good_Success(t *testing.T) {
	mergeCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && core.Contains(r.URL.Path, "/merge") {
			mergeCalled = true
			core.AssertEqual(t, "/api/v1/repos/core/test-repo/pulls/42/merge", r.URL.Path)
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"number": 42,
				"title":  "Fix login",
				"state":  "closed",
				"head":   map[string]any{"ref": "agent/fix-login"},
				"base":   map[string]any{"ref": "dev"},
			})))
			return
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"number": 42,
				"title":  "Fix login",
				"state":  "closed",
				"head":   map[string]any{"ref": "agent/fix-login"},
				"base":   map[string]any{"ref": "dev"},
			})))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.prMerge(context.Background(), nil, PRMergeInput{
		Repo:   "test-repo",
		Number: 42,
		Method: "merge",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertTrue(t, mergeCalled)
	core.AssertEqual(t, "test-repo", out.Repo)
	core.AssertEqual(t, 42, out.Number)
	core.AssertEqual(t, "merged", out.State)
}

// --- listPRs ---

func TestPr_ListPRs_Bad_NoToken(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.listPRs(context.Background(), nil, ListPRsInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no Forge token")
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
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	s.commentOnIssue(context.Background(), "core", "go-io", 42, "Test comment")
	core.AssertTrue(t, commentPosted)
}

// --- buildPRBody ---

func TestPr_BuildPRBody_Good_Case(t *testing.T) {
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
	core.AssertContains(t, body, "## Summary")
	core.AssertContains(t, body, "Fix the login bug")
	core.AssertContains(t, body, "Closes #42")
	core.AssertContains(t, body, "**Agent:** codex")
	core.AssertContains(t, body, "**Runs:** 3")
}

func TestPr_BuildPRBody_Bad_Case(t *testing.T) {
	// Empty status struct
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	st := &WorkspaceStatus{}
	body := s.buildPRBody(st)
	core.AssertContains(t, body, "## Summary")
	core.AssertContains(t, body, "**Agent:**")
	core.AssertNotContains(t, body, "Closes #")
}

func TestPr_BuildPRBody_Ugly_Case(t *testing.T) {
	// Very long task string
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	longTask := repeatString("This is a very long task description. ", 100)
	st := &WorkspaceStatus{
		Task:  longTask,
		Agent: "claude",
		Runs:  1,
	}
	body := s.buildPRBody(st)
	core.AssertContains(t, body, "## Summary")
	core.AssertContains(t, body, "very long task")
}

// --- commentOnIssue Bad/Ugly ---

func TestPr_CommentOnIssue_Bad_Case(t *testing.T) {
	// Forge returns error (500)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic even on server error
	core.AssertNotPanics(t, func() {
		s.commentOnIssue(context.Background(), "core", "go-io", 42, "Test comment")
	})
}

func TestPr_CommentOnIssue_Ugly_Case(t *testing.T) {
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
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	longComment := repeatString("This is a very long comment with details. ", 1000)
	s.commentOnIssue(context.Background(), "core", "go-io", 42, longComment)
	core.AssertTrue(t, commentPosted)
}

// --- createPR Ugly ---

func TestPr_CreatePR_Ugly_Case(t *testing.T) {
	// Workspace with no branch in status (auto-detect from git)
	root := t.TempDir()
	setTestWorkspace(t, root)

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
	core.RequireNoError(t, writeStatus(wsDir, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Branch: "", // empty branch — should auto-detect
		Task:   "Fix something",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "test-ws-ugly",
		DryRun:    true,
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertNotEmpty(t, out.Branch, "branch should be auto-detected from git")
}

// --- forgeCreatePR Ugly ---

func TestPr_ForgeCreatePR_Ugly_Case(t *testing.T) {
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
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic — may return zero values for missing fields
	core.AssertNotPanics(t, func() {
		_, _, _ = s.forgeCreatePR(
			context.Background(),
			"core", "test-repo",
			"agent/fix", "dev",
			"Title", "Body",
		)
	})
}

// --- listPRs Ugly ---

func TestPr_ListPRs_Ugly_Case(t *testing.T) {
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
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.listPRs(context.Background(), nil, ListPRsInput{
		State: "all",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
}

// --- listRepoPRs Good/Bad/Ugly ---

func TestPr_ListRepoPRs_Good_Case(t *testing.T) {
	// Specific repo with PRs
	srv := mockPRForgeServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prs, err := s.listRepoPRs(context.Background(), "core", "test-repo", "open")
	core.RequireNoError(t, err)
	// May be empty depending on mock, but should not error
	_ = prs
}

func TestPr_ListRepoPRs_Bad_Case(t *testing.T) {
	// Forge returns error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, err := s.listRepoPRs(context.Background(), "core", "go-io", "open")
	core.AssertError(t, err)
}

func TestPr_ListRepoPRs_Ugly_Case(t *testing.T) {
	// Repo with no PRs
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prs, err := s.listRepoPRs(context.Background(), "core", "empty-repo", "open")
	core.RequireNoError(t, err)
	core.AssertEmpty(t, prs)
}

func TestPr_DeleteBranch_Good_Success(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.deleteBranch(context.Background(), nil, DeleteBranchInput{
		Repo:   "test-repo",
		Branch: "agent/fix-tests",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "core", out.Org)
	core.AssertEqual(t, "test-repo", out.Repo)
	core.AssertEqual(t, "agent/fix-tests", out.Branch)
	core.AssertEqual(t, http.MethodDelete, method)
	core.AssertContains(t, path, "/branches/agent/fix-tests")
}

func TestPr_DeleteBranch_Bad_MissingRepo(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge("http://localhost:1", "test-token"),
		forgeToken:     "test-token",
	}

	_, _, err := s.deleteBranch(context.Background(), nil, DeleteBranchInput{
		Branch: "agent/fix-tests",
	})
	core.AssertError(t, err)
}

func TestPr_DeleteBranch_Bad_MissingBranch(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge("http://localhost:1", "test-token"),
		forgeToken:     "test-token",
	}

	_, _, err := s.deleteBranch(context.Background(), nil, DeleteBranchInput{
		Repo: "test-repo",
	})
	core.AssertError(t, err)
}

func TestPr_DeleteBranch_Ugly_NoForgeToken(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
	}

	_, _, err := s.deleteBranch(context.Background(), nil, DeleteBranchInput{
		Repo:   "test-repo",
		Branch: "agent/fix-tests",
	})
	core.AssertError(t, err)
}
