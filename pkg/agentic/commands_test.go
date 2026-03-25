// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
)

// testPrepWithCore creates a PrepSubsystem backed by a real Core + Forge mock.
func testPrepWithCore(t *testing.T, srv *httptest.Server) (*PrepSubsystem, *core.Core) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	c := core.New()

	var f *forge.Forge
	var client *http.Client
	if srv != nil {
		f = forge.NewForge(srv.URL, "test-token")
		client = srv.Client()
	}

	s := &PrepSubsystem{
		core:       c,
		forge:      f,
		forgeURL:   "",
		forgeToken: "test-token",
		client:     client,
		codePath:   t.TempDir(),
		pokeCh:     make(chan struct{}, 1),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}
	if srv != nil {
		s.forgeURL = srv.URL
	}

	return s, c
}

// --- Forge command methods (extracted from closures) ---

func TestCmdIssueGet_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueGet(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdIssueGet_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"number": 42, "title": "Fix tests", "state": "open",
			"html_url": "https://forge.test/core/go-io/issues/42", "body": "broken",
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "42"},
	))
	assert.True(t, r.OK)
}

func TestCmdIssueGet_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "42"},
	))
	assert.False(t, r.OK)
}

func TestCmdIssueList_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueList(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdIssueList_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"number": 1, "title": "Bug", "state": "open"},
			{"number": 2, "title": "Feature", "state": "closed"},
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCmdIssueList_Good_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCmdIssueComment_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueComment(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdIssueComment_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": 99})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueComment(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "5"},
		core.Option{Key: "body", Value: "LGTM"},
	))
	assert.True(t, r.OK)
}

func TestCmdIssueCreate_Bad_MissingTitle(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueCreate(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCmdIssueCreate_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"number": 10, "title": "New bug", "html_url": "https://forge.test/issues/10",
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueCreate(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "title", Value: "New bug"},
		core.Option{Key: "body", Value: "Details here"},
		core.Option{Key: "assignee", Value: "virgil"},
	))
	assert.True(t, r.OK)
}

func TestCmdIssueCreate_Good_WithLabelsAndMilestone(t *testing.T) {
	callPaths := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		switch {
		case r.URL.Path == "/api/v1/repos/core/go-io/milestones":
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "title": "v0.8.0"},
				{"id": 2, "title": "v0.9.0"},
			})
		case r.URL.Path == "/api/v1/repos/core/go-io/labels":
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": 10, "name": "agentic"},
				{"id": 11, "name": "bug"},
			})
		default:
			json.NewEncoder(w).Encode(map[string]any{
				"number": 15, "title": "Full issue", "html_url": "https://forge.test/issues/15",
			})
		}
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueCreate(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "title", Value: "Full issue"},
		core.Option{Key: "labels", Value: "agentic,bug"},
		core.Option{Key: "milestone", Value: "v0.8.0"},
		core.Option{Key: "ref", Value: "dev"},
	))
	assert.True(t, r.OK)
}

func TestCmdIssueCreate_Bad_APIError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			json.NewEncoder(w).Encode([]map[string]any{}) // milestones/labels
		} else {
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueCreate(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "title", Value: "Fail"},
	))
	assert.False(t, r.OK)
}

func TestCmdPRGet_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRGet(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdPRGet_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"number": 3, "title": "Fix", "state": "open", "mergeable": true,
			"html_url": "https://forge.test/pulls/3", "body": "PR body here",
			"head": map[string]any{"ref": "fix/it"}, "base": map[string]any{"ref": "dev"},
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "3"},
	))
	assert.True(t, r.OK)
}

func TestCmdPRGet_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "99"},
	))
	assert.False(t, r.OK)
}

func TestCmdPRList_Good_WithPRs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"number": 1, "title": "Fix", "state": "open",
				"head": map[string]any{"ref": "fix/a"}, "base": map[string]any{"ref": "dev"}},
			{"number": 2, "title": "Feat", "state": "closed",
				"head": map[string]any{"ref": "feat/b"}, "base": map[string]any{"ref": "dev"}},
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCmdPRList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCmdPRMerge_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		json.NewEncoder(w).Encode(map[string]any{"message": "conflict"})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRMerge(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "5"},
	))
	assert.False(t, r.OK)
}

func TestCmdPRMerge_Good_CustomMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRMerge(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "5"},
		core.Option{Key: "method", Value: "squash"},
	))
	assert.True(t, r.OK)
}

func TestCmdIssueGet_Good_WithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"number": 1, "title": "Bug", "state": "open",
			"html_url": "https://forge.test/issues/1", "body": "Detailed description",
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "1"},
	))
	assert.True(t, r.OK)
}

func TestCmdIssueList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCmdIssueComment_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueComment(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "1"},
		core.Option{Key: "body", Value: "test"},
	))
	assert.False(t, r.OK)
}

func TestCmdRepoGet_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCmdRepoList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdPRList_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRList(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdPRList_Good_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCmdPRMerge_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRMerge(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdPRMerge_Good_DefaultMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRMerge(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "5"},
	))
	assert.True(t, r.OK)
}

func TestCmdRepoGet_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdRepoGet(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdRepoGet_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"name": "go-io", "description": "IO", "default_branch": "dev",
			"private": false, "archived": false, "html_url": "https://forge.test/go-io",
			"owner": map[string]any{"login": "core"},
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCmdRepoList_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "go-io", "description": "IO", "archived": false, "owner": map[string]any{"login": "core"}},
			{"name": "go-log", "description": "Logging", "archived": true, "owner": map[string]any{"login": "core"}},
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions())
	assert.True(t, r.OK)
}

// --- Workspace command methods ---

func TestCmdWorkspaceList_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCmdWorkspaceList_Good_WithEntries(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := filepath.Join(wsRoot, "ws-1")
	os.MkdirAll(ws, 0o755)
	data, _ := json.Marshal(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"})
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCmdWorkspaceClean_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceClean(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCmdWorkspaceClean_Good_RemovesCompleted(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := filepath.Join(wsRoot, "ws-done")
	os.MkdirAll(ws, 0o755)
	data, _ := json.Marshal(WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex"})
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	r := s.cmdWorkspaceClean(core.NewOptions())
	assert.True(t, r.OK)

	_, err := os.Stat(ws)
	assert.True(t, os.IsNotExist(err))
}

func TestCmdWorkspaceClean_Good_FilterFailed(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	for _, ws := range []struct{ name, status string }{
		{"ws-ok", "completed"},
		{"ws-bad", "failed"},
	} {
		d := filepath.Join(wsRoot, ws.name)
		os.MkdirAll(d, 0o755)
		data, _ := json.Marshal(WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"})
		os.WriteFile(filepath.Join(d, "status.json"), data, 0o644)
	}

	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "failed"}))
	assert.True(t, r.OK)

	_, err1 := os.Stat(filepath.Join(wsRoot, "ws-bad"))
	assert.True(t, os.IsNotExist(err1))
	_, err2 := os.Stat(filepath.Join(wsRoot, "ws-ok"))
	assert.NoError(t, err2)
}

func TestCmdWorkspaceClean_Good_FilterBlocked(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	d := filepath.Join(wsRoot, "ws-stuck")
	os.MkdirAll(d, 0o755)
	data, _ := json.Marshal(WorkspaceStatus{Status: "blocked", Repo: "test", Agent: "codex"})
	os.WriteFile(filepath.Join(d, "status.json"), data, 0o644)

	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "blocked"}))
	assert.True(t, r.OK)

	_, err := os.Stat(d)
	assert.True(t, os.IsNotExist(err))
}

func TestCmdWorkspaceDispatch_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceDispatch(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdWorkspaceDispatch_Good_Stub(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceDispatch(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

// --- commands.go extracted methods ---

func TestCmdPrep_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdPrep_Good_DefaultsToDev(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	// Will fail (no local clone) but exercises the default branch logic
	r := s.cmdPrep(core.NewOptions(core.Option{Key: "_arg", Value: "nonexistent-repo"}))
	assert.False(t, r.OK) // expected — no local repo
}

func TestCmdStatus_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdStatus(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCmdStatus_Good_WithWorkspaces(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := filepath.Join(wsRoot, "ws-1")
	os.MkdirAll(ws, 0o755)
	data, _ := json.Marshal(WorkspaceStatus{Status: "completed", Repo: "test", Agent: "codex"})
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	r := s.cmdStatus(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCmdPrompt_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrompt(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdPrompt_Good_DefaultTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrompt(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCmdExtract_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	target := filepath.Join(t.TempDir(), "extract-test")
	r := s.cmdExtract(core.NewOptions(
		core.Option{Key: "_arg", Value: "default"},
		core.Option{Key: "target", Value: target},
	))
	assert.True(t, r.OK)
}

func TestCmdRunTask_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := s.cmdRunTask(ctx, core.NewOptions())
	assert.False(t, r.OK)
}

func TestCmdRunTask_Bad_MissingTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := s.cmdRunTask(ctx, core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCmdOrchestrator_Good_CancelledCtx(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	r := s.cmdOrchestrator(ctx, core.NewOptions())
	assert.True(t, r.OK)
}

func TestParseIntStr_Good(t *testing.T) {
	assert.Equal(t, 42, parseIntStr("42"))
	assert.Equal(t, 123, parseIntStr("issue-123"))
	assert.Equal(t, 0, parseIntStr(""))
	assert.Equal(t, 0, parseIntStr("abc"))
	assert.Equal(t, 7, parseIntStr("#7"))
}

// --- Registration verification ---

func TestRegisterCommands_Good_AllRegistered(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.registerCommands(ctx)

	cmds := c.Commands()
	assert.Contains(t, cmds, "run/task")
	assert.Contains(t, cmds, "run/orchestrator")
	assert.Contains(t, cmds, "prep")
	assert.Contains(t, cmds, "status")
	assert.Contains(t, cmds, "prompt")
	assert.Contains(t, cmds, "extract")
}
