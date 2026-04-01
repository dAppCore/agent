// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPrepWithCore creates a PrepSubsystem backed by a real Core + Forge mock.
func testPrepWithCore(t *testing.T, srv *httptest.Server) (*PrepSubsystem, *core.Core) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	c := core.New()

	var f *forge.Forge
	if srv != nil {
		f = forge.NewForge(srv.URL, "test-token")
	}

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		forge:          f,
		forgeURL:       "",
		forgeToken:     "test-token",
		codePath:       t.TempDir(),
		pokeCh:         make(chan struct{}, 1),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	if srv != nil {
		s.forgeURL = srv.URL
	}

	return s, c
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = old
	}()

	run()

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}

	return string(data)
}

// --- Forge command methods (extracted from closures) ---

func TestCommandsforge_CmdIssueGet_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueGet(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueGet_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 42, "title": "Fix tests", "state": "open",
			"html_url": "https://forge.test/core/go-io/issues/42", "body": "broken",
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "42"},
	))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdIssueGet_Bad_APIError(t *testing.T) {
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

func TestCommandsforge_CmdIssueList_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueList(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{"number": 1, "title": "Bug", "state": "open"},
			{"number": 2, "title": "Feature", "state": "closed"},
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Good_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdIssueComment_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueComment(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueComment_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{"id": 99})))
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

func TestCommandsforge_CmdIssueCreate_Bad_MissingTitle(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueCreate(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueCreate_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 10, "title": "New bug", "html_url": "https://forge.test/issues/10",
		})))
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

func TestCommandsforge_CmdIssueCreate_Good_WithLabelsAndMilestone(t *testing.T) {
	callPaths := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callPaths = append(callPaths, r.URL.Path)
		switch {
		case r.URL.Path == "/api/v1/repos/core/go-io/milestones":
			w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"id": 1, "title": "v0.8.0"},
				{"id": 2, "title": "v0.9.0"},
			})))
		case r.URL.Path == "/api/v1/repos/core/go-io/labels":
			w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"id": 10, "name": "agentic"},
				{"id": 11, "name": "bug"},
			})))
		default:
			w.Write([]byte(core.JSONMarshalString(map[string]any{
				"number": 15, "title": "Full issue", "html_url": "https://forge.test/issues/15",
			})))
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

func TestCommands_CmdBrainList_Good(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.list", func(_ context.Context, options core.Options) core.Result {
		assert.Equal(t, "agent", options.String("project"))
		assert.Equal(t, "architecture", options.String("type"))
		assert.Equal(t, "virgil", options.String("agent_id"))
		return core.Result{Value: map[string]any{
			"success": true,
			"count":   1,
			"memories": []any{
				map[string]any{
					"id":         "mem-1",
					"type":       "architecture",
					"content":    "Use named actions.",
					"project":    "agent",
					"agent_id":   "virgil",
					"confidence": 0.9,
					"tags":       []any{"architecture", "convention"},
				},
			},
		}, OK: true}
	})

	output := captureStdout(t, func() {
		result := s.cmdBrainList(core.NewOptions(
			core.Option{Key: "project", Value: "agent"},
			core.Option{Key: "type", Value: "architecture"},
			core.Option{Key: "agent", Value: "virgil"},
		))
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "count: 1")
	assert.Contains(t, output, "mem-1 architecture")
	assert.Contains(t, output, "Use named actions.")
}

func TestCommands_CmdBrainList_Bad_MissingAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdBrainList(core.NewOptions())

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "action not registered")
}

func TestCommands_CmdBrainList_Ugly_InvalidOutput(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.list", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{Value: 123, OK: true}
	})

	result := s.cmdBrainList(core.NewOptions())

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "invalid brain list output")
}

func TestCommandsforge_CmdIssueCreate_Bad_APIError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			w.Write([]byte(core.JSONMarshalString([]map[string]any{}))) // milestones/labels
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

func TestCommandsforge_CmdPRGet_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRGet(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRGet_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 3, "title": "Fix", "state": "open", "mergeable": true,
			"html_url": "https://forge.test/pulls/3", "body": "PR body here",
			"head": map[string]any{"ref": "fix/it"}, "base": map[string]any{"ref": "dev"},
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "3"},
	))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdPRGet_Bad_APIError(t *testing.T) {
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

func TestCommandsforge_CmdPRList_Good_WithPRs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{"number": 1, "title": "Fix", "state": "open",
				"head": map[string]any{"ref": "fix/a"}, "base": map[string]any{"ref": "dev"}},
			{"number": 2, "title": "Feat", "state": "closed",
				"head": map[string]any{"ref": "feat/b"}, "base": map[string]any{"ref": "dev"}},
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdPRList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRMerge_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		w.Write([]byte(core.JSONMarshalString(map[string]any{"message": "conflict"})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRMerge(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "5"},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRMerge_Good_CustomMethod(t *testing.T) {
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

func TestCommandsforge_CmdPRClose_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRClose(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRClose_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/v1/repos/core/go-io/pulls/5", r.URL.Path)

		bodyResult := core.ReadAll(r.Body)
		assert.True(t, bodyResult.OK)
		assert.Contains(t, bodyResult.Value.(string), `"state":"closed"`)

		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 5,
			"state":  "closed",
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRClose(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "5"},
	))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdPRClose_Ugly_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRClose(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "5"},
	))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueGet_Good_WithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 1, "title": "Bug", "state": "open",
			"html_url": "https://forge.test/issues/1", "body": "Detailed description",
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "1"},
	))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdIssueComment_Bad_APIError(t *testing.T) {
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

func TestCommandsforge_CmdRepoGet_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdRepoList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRList_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRList(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRList_Good_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdPRMerge_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRMerge(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdPRMerge_Good_DefaultMethod(t *testing.T) {
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

func TestCommandsforge_CmdRepoGet_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdRepoGet(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsforge_CmdRepoGet_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"name": "go-io", "description": "IO", "default_branch": "dev",
			"private": false, "archived": false, "html_url": "https://forge.test/go-io",
			"owner": map[string]any{"login": "core"},
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCommandsforge_CmdRepoList_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{"name": "go-io", "description": "IO", "archived": false, "owner": map[string]any{"login": "core"}},
			{"name": "go-log", "description": "Logging", "archived": true, "owner": map[string]any{"login": "core"}},
		})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions())
	assert.True(t, r.OK)
}

// --- Workspace command methods ---

func TestCommandsworkspace_CmdWorkspaceList_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceList_Good_WithEntries(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-1")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"}))

	r := s.cmdWorkspaceList(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceClean(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_RemovesCompleted(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-done")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex"}))

	r := s.cmdWorkspaceClean(core.NewOptions())
	assert.True(t, r.OK)

	assert.False(t, fs.Exists(ws))
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_FilterFailed(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	for _, ws := range []struct{ name, status string }{
		{"ws-ok", "completed"},
		{"ws-bad", "failed"},
	} {
		d := core.JoinPath(wsRoot, ws.name)
		fs.EnsureDir(d)
		fs.Write(core.JoinPath(d, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"}))
	}

	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "failed"}))
	assert.True(t, r.OK)

	assert.False(t, fs.Exists(core.JoinPath(wsRoot, "ws-bad")))
	assert.True(t, fs.Exists(core.JoinPath(wsRoot, "ws-ok")))
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_FilterBlocked(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	d := core.JoinPath(wsRoot, "ws-stuck")
	fs.EnsureDir(d)
	fs.Write(core.JoinPath(d, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "blocked", Repo: "test", Agent: "codex"}))

	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "blocked"}))
	assert.True(t, r.OK)

	assert.False(t, fs.Exists(d))
}

func TestCommandsworkspace_CmdWorkspaceDispatch_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceDispatch(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceDispatch_Bad_MissingTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceDispatch(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.False(t, r.OK) // task is required
}

// --- commands.go extracted methods ---

func TestCommands_CmdPrep_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommands_CmdPrep_Good_DefaultsToDev(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	// Will fail (no local clone) but exercises the default branch logic
	r := s.cmdPrep(core.NewOptions(core.Option{Key: "_arg", Value: "nonexistent-repo"}))
	assert.False(t, r.OK) // expected — no local repo
}

func TestCommands_CmdStatus_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdStatus(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCommands_CmdStatus_Good_WithWorkspaces(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-1")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "completed", Repo: "test", Agent: "codex"}))

	r := s.cmdStatus(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCommands_CmdStatus_Good_DeepWorkspace(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	ws := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-5")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	output := captureStdout(t, func() {
		r := s.cmdStatus(core.NewOptions())
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "completed")
	assert.Contains(t, output, "codex")
	assert.Contains(t, output, "go-io")
	assert.Contains(t, output, "core/go-io/task-5")
}

func TestCommands_CmdStatus_Good_BranchWorkspace(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	ws := core.JoinPath(WorkspaceRoot(), "core", "go-io", "feature", "new-ui")
	fs.EnsureDir(WorkspaceRepoDir(ws))
	fs.EnsureDir(WorkspaceMetaDir(ws))
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	output := captureStdout(t, func() {
		r := s.cmdStatus(core.NewOptions())
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "completed")
	assert.Contains(t, output, "core/go-io/feature/new-ui")
}

func TestCommands_CmdStatus_Good_BlockedQuestion(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	ws := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-9")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status:   "blocked",
		Repo:     "go-io",
		Agent:    "gemini",
		Question: "Which API version?",
	}))

	output := captureStdout(t, func() {
		r := s.cmdStatus(core.NewOptions())
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "blocked")
	assert.Contains(t, output, "gemini")
	assert.Contains(t, output, "go-io")
	assert.Contains(t, output, "Which API version?")
}

func TestCommands_CmdStatus_Good_WorkspaceFilter(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsA := core.JoinPath(WorkspaceRoot(), "core", "go-io", "task-9")
	fs.EnsureDir(wsA)
	fs.Write(core.JoinPath(wsA, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status: "blocked",
		Repo:   "go-io",
		Agent:  "gemini",
	}))

	wsB := core.JoinPath(WorkspaceRoot(), "core", "go-log", "task-4")
	fs.EnsureDir(wsB)
	fs.Write(core.JoinPath(wsB, "status.json"), core.JSONMarshalString(WorkspaceStatus{
		Status: "completed",
		Repo:   "go-log",
		Agent:  "codex",
	}))

	output := captureStdout(t, func() {
		r := s.cmdStatus(core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-9"}))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "core/go-io/task-9")
	assert.NotContains(t, output, "core/go-log/task-4")
}

func TestCommands_CmdStatus_Good_StatusFilterAndLimit(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	for _, workspace := range []struct {
		name   string
		status string
		repo   string
	}{
		{name: "ws-1", status: "blocked", repo: "go-io"},
		{name: "ws-2", status: "blocked", repo: "go-log"},
		{name: "ws-3", status: "completed", repo: "go-scm"},
	} {
		ws := core.JoinPath(WorkspaceRoot(), workspace.name)
		fs.EnsureDir(ws)
		fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{
			Status: workspace.status,
			Repo:   workspace.repo,
			Agent:  "codex",
		}))
	}

	output := captureStdout(t, func() {
		r := s.cmdStatus(core.NewOptions(
			core.Option{Key: "status", Value: "blocked"},
			core.Option{Key: "limit", Value: 1},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "ws-1")
	assert.NotContains(t, output, "ws-2")
	assert.NotContains(t, output, "ws-3")
}

func TestCommands_CmdPrompt_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrompt(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommands_CmdPrompt_Good_DefaultTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrompt(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	assert.True(t, r.OK)
}

func TestCommands_CmdGenerate_Bad_MissingPrompt(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdGenerate(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommands_CmdGenerate_Good(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/content/generate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"data":{"id":"gen_1","provider":"claude","model":"claude-3.7-sonnet","content":"Release notes draft","input_tokens":12,"output_tokens":48,"status":"completed"}}`))
	}))
	defer server.Close()

	s := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		r := s.cmdGenerate(core.NewOptions(
			core.Option{Key: "_arg", Value: "Draft a release note"},
			core.Option{Key: "provider", Value: "claude"},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "provider: claude")
	assert.Contains(t, output, "model:    claude-3.7-sonnet")
	assert.Contains(t, output, "status:   completed")
	assert.Contains(t, output, "content:  Release notes draft")
}

func TestCommands_CmdGenerate_Good_BriefTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/content/generate", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		require.True(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		require.True(t, parseResult.OK)
		assert.Equal(t, "brief_1", payload["brief_id"])
		assert.Equal(t, "help-article", payload["template"])

		_, _ = w.Write([]byte(`{"data":{"id":"gen_2","provider":"claude","model":"claude-3.7-sonnet","content":"Template draft","status":"completed"}}`))
	}))
	defer server.Close()

	s := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		r := s.cmdGenerate(core.NewOptions(
			core.Option{Key: "brief_id", Value: "brief_1"},
			core.Option{Key: "template", Value: "help-article"},
			core.Option{Key: "provider", Value: "claude"},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "provider: claude")
	assert.Contains(t, output, "model:    claude-3.7-sonnet")
	assert.Contains(t, output, "status:   completed")
	assert.Contains(t, output, "content:  Template draft")
}

func TestCommands_CmdScan_Good(t *testing.T) {
	server := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(server.URL, "secret-token"),
		forgeURL:       server.URL,
		forgeToken:     "secret-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	output := captureStdout(t, func() {
		r := s.cmdScan(core.NewOptions(
			core.Option{Key: "org", Value: "core"},
			core.Option{Key: "labels", Value: "agentic,bug"},
			core.Option{Key: "limit", Value: 5},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "count:")
	assert.Contains(t, output, "go-io#10")
	assert.Contains(t, output, "Add missing tests")
}

func TestCommands_CmdScan_Bad_NoForgeToken(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	s.forgeToken = ""

	r := s.cmdScan(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommands_CmdScan_Ugly_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/orgs/core/repos":
			_, _ = w.Write([]byte(core.JSONMarshalString([]map[string]any{
				{"name": "go-io"},
			})))
		default:
			_, _ = w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
		}
	}))
	defer server.Close()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(server.URL, "secret-token"),
		forgeURL:       server.URL,
		forgeToken:     "secret-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	output := captureStdout(t, func() {
		r := s.cmdScan(core.NewOptions(
			core.Option{Key: "org", Value: "core"},
			core.Option{Key: "limit", Value: 1},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "count: 0")
}

func TestCommands_CmdPlanCreate_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	r := s.cmdPlanCreate(core.NewOptions(
		core.Option{Key: "slug", Value: "migrate-core"},
		core.Option{Key: "title", Value: "Migrate Core"},
		core.Option{Key: "objective", Value: "Use Core.Process everywhere"},
	))

	assert.True(t, r.OK)

	output, ok := r.Value.(PlanCreateOutput)
	require.True(t, ok)
	require.NotEmpty(t, output.ID)
	require.NotEmpty(t, output.Path)
	assert.True(t, fs.Exists(output.Path))

	plan, err := readPlan(PlansRoot(), output.ID)
	require.NoError(t, err)
	assert.Equal(t, "Migrate Core", plan.Title)
	assert.Equal(t, "Use Core.Process everywhere", plan.Objective)
	assert.Equal(t, "draft", plan.Status)
	assert.Equal(t, "migrate-core", plan.Slug)
}

func TestCommands_CmdPlanStatus_Good_GetAndSet(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Status Plan",
		Objective: "Exercise plan status management",
	})
	require.NoError(t, err)

	getOutput := captureStdout(t, func() {
		r := s.cmdPlanStatus(core.NewOptions(core.Option{Key: "_arg", Value: created.ID}))
		assert.True(t, r.OK)
	})
	assert.Contains(t, getOutput, "status:")
	assert.Contains(t, getOutput, "draft")

	setOutput := captureStdout(t, func() {
		r := s.cmdPlanStatus(core.NewOptions(
			core.Option{Key: "_arg", Value: created.ID},
			core.Option{Key: "set", Value: "ready"},
		))
		assert.True(t, r.OK)
	})
	assert.Contains(t, setOutput, "status:")
	assert.Contains(t, setOutput, "ready")

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "ready", plan.Status)
}

func TestCommands_CmdPlanArchive_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Archive Plan",
		Objective: "Exercise archive command",
	})
	require.NoError(t, err)

	output := captureStdout(t, func() {
		r := s.cmdPlanArchive(core.NewOptions(
			core.Option{Key: "_arg", Value: created.ID},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "archived:")

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "archived", plan.Status)
	assert.False(t, plan.ArchivedAt.IsZero())
}

func TestCommands_CmdPlanDelete_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Delete Plan",
		Objective: "Exercise delete command",
	})
	require.NoError(t, err)

	output := captureStdout(t, func() {
		r := s.cmdPlanDelete(core.NewOptions(
			core.Option{Key: "_arg", Value: created.ID},
		))
		assert.True(t, r.OK)
	})

	assert.Contains(t, output, "deleted:")

	plan, err := readPlan(PlansRoot(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "archived", plan.Status)
	assert.False(t, plan.ArchivedAt.IsZero())
}

func TestCommands_CmdExtract_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	target := core.JoinPath(t.TempDir(), "extract-test")
	r := s.cmdExtract(core.NewOptions(
		core.Option{Key: "_arg", Value: "default"},
		core.Option{Key: "target", Value: target},
	))
	assert.True(t, r.OK)
}

func TestCommands_CmdRunTask_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startupContext = ctx
	r := s.cmdRunTask(core.NewOptions())
	assert.False(t, r.OK)
}

func TestCommands_CmdRunTask_Bad_MissingTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startupContext = ctx
	r := s.cmdRunTask(core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	assert.False(t, r.OK)
}

func TestCommands_CmdOrchestrator_Good_CancelledCtx(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	assert.True(t, r.OK)
}

func TestCommands_ParseIntStr_Good(t *testing.T) {
	assert.Equal(t, 42, parseIntString("42"))
	assert.Equal(t, 123, parseIntString("issue-123"))
	assert.Equal(t, 0, parseIntString(""))
	assert.Equal(t, 0, parseIntString("abc"))
	assert.Equal(t, 7, parseIntString("#7"))
}

// --- Registration verification ---

func TestCommands_RegisterCommands_Good_AllRegistered(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.registerCommands(ctx)

	cmds := c.Commands()
	assert.Contains(t, cmds, "run/task")
	assert.Contains(t, cmds, "run/orchestrator")
	assert.Contains(t, cmds, "prep")
	assert.Contains(t, cmds, "scan")
	assert.Contains(t, cmds, "status")
	assert.Contains(t, cmds, "prompt")
	assert.Contains(t, cmds, "extract")
	assert.Contains(t, cmds, "lang/detect")
	assert.Contains(t, cmds, "lang/list")
	assert.Contains(t, cmds, "plan")
	assert.Contains(t, cmds, "plan/create")
	assert.Contains(t, cmds, "plan/list")
	assert.Contains(t, cmds, "plan/show")
	assert.Contains(t, cmds, "plan/status")
	assert.Contains(t, cmds, "plan/archive")
	assert.Contains(t, cmds, "plan/delete")
	assert.Contains(t, cmds, "pr-manage")
	assert.Contains(t, cmds, "task")
	assert.Contains(t, cmds, "task/update")
	assert.Contains(t, cmds, "task/toggle")
}

func TestCommands_CmdPRManage_Good_NoCandidates(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRManage(core.NewOptions())
	assert.True(t, r.OK)
}

// --- CmdExtract Bad/Ugly ---

func TestCommands_CmdExtract_Bad_TargetDirAlreadyHasFiles(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	target := core.JoinPath(t.TempDir(), "extract-existing")
	fs.EnsureDir(target)
	fs.Write(core.JoinPath(target, "existing.txt"), "data")

	// Missing template arg uses "default", target already has files — still succeeds (overwrites)
	r := s.cmdExtract(core.NewOptions(
		core.Option{Key: "target", Value: target},
	))
	assert.True(t, r.OK)
}

func TestCommands_CmdExtract_Ugly_TargetIsFile(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	target := core.JoinPath(t.TempDir(), "not-a-dir")
	fs.Write(target, "I am a file")

	r := s.cmdExtract(core.NewOptions(
		core.Option{Key: "_arg", Value: "default"},
		core.Option{Key: "target", Value: target},
	))
	// Extraction should fail because target is a file, not a directory
	assert.False(t, r.OK)
}

// --- CmdOrchestrator Bad/Ugly ---

func TestCommands_CmdOrchestrator_Bad_DoneContext(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	assert.True(t, r.OK) // returns OK after ctx.Done()
}

func TestCommands_CmdOrchestrator_Ugly_CancelledImmediately(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	assert.True(t, r.OK) // exits immediately when context is already cancelled
}

// --- CmdPrep Ugly ---

func TestCommands_CmdPrep_Ugly_AllOptionalFields(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions(
		core.Option{Key: "_arg", Value: "nonexistent-repo"},
		core.Option{Key: "issue", Value: "42"},
		core.Option{Key: "pr", Value: "7"},
		core.Option{Key: "branch", Value: "feat/test"},
		core.Option{Key: "tag", Value: "v1.0.0"},
		core.Option{Key: "task", Value: "do stuff"},
		core.Option{Key: "template", Value: "coding"},
		core.Option{Key: "persona", Value: "engineering"},
		core.Option{Key: "dry-run", Value: "true"},
	))
	// Will fail (no local clone) but exercises all option parsing paths
	assert.False(t, r.OK)
}

// --- CmdPrompt Ugly ---

func TestCommands_CmdPrompt_Ugly_AllOptionalFields(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrompt(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "task", Value: "review security"},
		core.Option{Key: "template", Value: "verify"},
		core.Option{Key: "persona", Value: "engineering/security"},
	))
	assert.True(t, r.OK)
}

// --- CmdRunTask Good/Ugly ---

func TestCommands_CmdRunTask_Good_DefaultsApplied(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.startupContext = ctx
	// Provide repo + task but omit agent + org — tests that defaults (codex, core) are applied
	r := s.cmdRunTask(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "run all tests"},
	))
	// Will fail on dispatch but exercises the default-filling path
	assert.False(t, r.OK)
}

func TestCommands_CmdRunTask_Ugly_MixedIssueString(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.startupContext = ctx
	r := s.cmdRunTask(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "fix it"},
		core.Option{Key: "issue", Value: "issue-42abc"},
	))
	// Will fail on dispatch but exercises parseIntString with mixed chars
	assert.False(t, r.OK)
}

// --- CommandContext Good/Bad/Ugly ---

func TestCommands_CommandContext_Good_StoredStartupContext(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.registerCommands(ctx)
	assert.Same(t, ctx, s.commandContext())
}

func TestCommands_CommandContext_Bad_FallsBackToBackground(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	assert.NotNil(t, s.commandContext())
	assert.NoError(t, s.commandContext().Err())
}

func TestCommands_CommandContext_Ugly_CancelledStartupContext(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	assert.True(t, r.OK)
}

// --- CmdStatus Bad/Ugly ---

func TestCommands_CmdStatus_Bad_NoWorkspaceDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	// Don't create workspace dir — WorkspaceRoot() returns root+"/workspace" which won't exist

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdStatus(core.NewOptions())
	assert.True(t, r.OK) // returns OK with "no workspaces found"
}

func TestCommands_CmdStatus_Ugly_NonDirEntries(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	wsRoot := WorkspaceRoot()
	fs.EnsureDir(wsRoot)

	// Create a file (not a dir) inside workspace root
	fs.Write(core.JoinPath(wsRoot, "not-a-workspace.txt"), "junk")

	// Also create a proper workspace
	ws := core.JoinPath(wsRoot, "ws-valid")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "running", Repo: "test", Agent: "codex"}))

	r := s.cmdStatus(core.NewOptions())
	assert.True(t, r.OK)
}

// --- ParseIntStr Bad/Ugly ---

func TestCommands_ParseIntStr_Bad_NegativeAndOverflow(t *testing.T) {
	// parseIntString extracts digits only, ignoring minus signs
	assert.Equal(t, 5, parseIntString("-5"))  // extracts "5", ignores "-"
	assert.Equal(t, 0, parseIntString("-"))   // no digits
	assert.Equal(t, 0, parseIntString("---")) // no digits
}

func TestCommands_ParseIntStr_Ugly_UnicodeAndMixed(t *testing.T) {
	// Unicode digits (e.g. Arabic-Indic) are NOT ASCII 0-9 so ignored
	assert.Equal(t, 0, parseIntString("\u0661\u0662\u0663")) // ١٢٣ — not ASCII digits
	assert.Equal(t, 42, parseIntString("abc42xyz"))          // mixed chars
	assert.Equal(t, 123, parseIntString("1a2b3c"))           // interleaved
	assert.Equal(t, 0, parseIntString("  \t\n"))             // whitespace only
}
