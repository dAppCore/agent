// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
)

// testPrepWithCore creates a PrepSubsystem backed by a real Core + Forge mock.
func testPrepWithCore(t *testing.T, srv *httptest.Server) (*PrepSubsystem, *core.Core) {
	t.Helper()
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := core.New()

	var f *forgeClient
	if srv != nil {
		f = newForgeClient(srv.URL, "test-token")
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

// --- Forge command methods (extracted from closures) ---

func TestCommandsforge_CmdIssueGet_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueGet(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
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
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueList(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Good_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdIssueComment_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueComment(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdIssueCreate_Bad_MissingTitle(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdIssueCreate(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

func TestCommands_RegisterCommands_Good_BrainRecall(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCommands(context.Background())

	core.AssertContains(t, c.Commands(), "brain/recall")
	core.AssertContains(t, c.Commands(), "brain:recall")
	core.AssertContains(t, c.Commands(), "brain/remember")
	core.AssertContains(t, c.Commands(), "brain:remember")
}

func TestCommands_CmdBrainList_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.list", func(_ context.Context, options core.Options) core.Result {
		core.AssertEqual(t, "agent", options.String("project"))
		core.AssertEqual(t, "architecture", options.String("type"))
		core.AssertEqual(t, "virgil", options.String("agent_id"))
		return core.Result{Value: map[string]any{
			"success": true,
			"count":   1,
			"memories": []any{
				map[string]any{
					"id":               "mem-1",
					"type":             "architecture",
					"content":          "Use named actions.",
					"project":          "agent",
					"agent_id":         "virgil",
					"confidence":       0.9,
					"supersedes_count": 3,
					"deleted_at":       "2026-03-31T12:30:00Z",
					"tags":             []any{"architecture", "convention"},
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
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "count: 1")
	core.AssertContains(t, output, "mem-1 architecture")
	core.AssertContains(t, output, "supersedes: 3")
	core.AssertContains(t, output, "deleted_at: 2026-03-31T12:30:00Z")
	core.AssertContains(t, output, "Use named actions.")
}

func TestCommands_CmdBrainRemember_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.remember", func(_ context.Context, options core.Options) core.Result {
		core.AssertEqual(t, "Use named actions.", options.String("content"))
		core.AssertEqual(t, "convention", options.String("type"))
		core.AssertEqual(t, []string{"architecture", "history"}, optionStringSliceValue(options, "tags"))
		core.AssertEqual(t, "agent", options.String("project"))
		core.AssertEqual(t, "0.9", options.String("confidence"))
		core.AssertEqual(t, "mem-1", options.String("supersedes"))
		core.AssertEqual(t, 24, options.Int("expires_in"))
		return core.Result{Value: map[string]any{
			"success":   true,
			"memory_id": "mem-1",
			"timestamp": "2026-03-31T12:00:00Z",
		}, OK: true}
	})

	output := captureStdout(t, func() {
		result := s.cmdBrainRemember(core.NewOptions(
			core.Option{Key: "_arg", Value: "Use named actions."},
			core.Option{Key: "type", Value: "convention"},
			core.Option{Key: "tags", Value: []string{"architecture", "history"}},
			core.Option{Key: "project", Value: "agent"},
			core.Option{Key: "confidence", Value: "0.9"},
			core.Option{Key: "supersedes", Value: "mem-1"},
			core.Option{Key: "expires_in", Value: 24},
		))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "remembered: mem-1")
	core.AssertContains(t, output, "timestamp:  2026-03-31T12:00:00Z")
}

func TestCommands_CmdBrainRemember_Bad_MissingContent(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdBrainRemember(core.NewOptions(
		core.Option{Key: "type", Value: "convention"},
	))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "content and type are required")
}

func TestCommands_CmdBrainRemember_Ugly_InvalidOutput(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.remember", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{Value: 123, OK: true}
	})

	result := s.cmdBrainRemember(core.NewOptions(
		core.Option{Key: "_arg", Value: "Use named actions."},
		core.Option{Key: "type", Value: "convention"},
	))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "invalid brain remember output")
}

func TestCommands_CmdBrainList_Bad_MissingAction(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdBrainList(core.NewOptions())

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "action not registered")
}

func TestCommands_CmdBrainList_Ugly_InvalidOutput(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.list", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{Value: 123, OK: true}
	})

	result := s.cmdBrainList(core.NewOptions())

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "invalid brain list output")
}

func TestCommands_CmdBrainRecall_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.recall", func(_ context.Context, options core.Options) core.Result {
		core.AssertEqual(t, "workspace handoff context", options.String("query"))
		core.AssertEqual(t, 3, options.Int("top_k"))
		core.AssertEqual(t, "agent", options.String("project"))
		core.AssertEqual(t, "architecture", options.String("type"))
		core.AssertEqual(t, "virgil", options.String("agent_id"))
		core.AssertEqual(t, "0.75", options.String("min_confidence"))
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
					"confidence": 0.75,
					"tags":       []any{"architecture", "convention"},
				},
			},
		}, OK: true}
	})

	output := captureStdout(t, func() {
		result := s.cmdBrainRecall(core.NewOptions(
			core.Option{Key: "_arg", Value: "workspace handoff context"},
			core.Option{Key: "top_k", Value: 3},
			core.Option{Key: "project", Value: "agent"},
			core.Option{Key: "type", Value: "architecture"},
			core.Option{Key: "agent", Value: "virgil"},
			core.Option{Key: "min_confidence", Value: "0.75"},
		))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "count: 1")
	core.AssertContains(t, output, "mem-1 architecture")
	core.AssertContains(t, output, "Use named actions.")
}

func TestCommands_CmdBrainRecall_Bad_MissingQuery(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdBrainRecall(core.NewOptions())

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "query is required")
}

func TestCommands_CmdBrainRecall_Ugly_InvalidOutput(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.recall", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{Value: 123, OK: true}
	})

	result := s.cmdBrainRecall(core.NewOptions(core.Option{Key: "_arg", Value: "workspace handoff context"}))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "invalid brain recall output")
}

func TestCommands_CmdBrainForget_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.forget", func(_ context.Context, options core.Options) core.Result {
		core.AssertEqual(t, "mem-1", options.String("id"))
		core.AssertEqual(t, "superseded", options.String("reason"))
		return core.Result{Value: map[string]any{
			"success":   true,
			"forgotten": "mem-1",
		}, OK: true}
	})

	output := captureStdout(t, func() {
		result := s.cmdBrainForget(core.NewOptions(
			core.Option{Key: "_arg", Value: "mem-1"},
			core.Option{Key: "reason", Value: "superseded"},
		))
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "forgotten: mem-1")
	core.AssertContains(t, output, "reason:    superseded")
}

func TestCommands_CmdBrainForget_Bad_MissingID(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdBrainForget(core.NewOptions())

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "memory id is required")
}

func TestCommands_CmdBrainForget_Ugly_ActionFailure(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	c.Action("brain.forget", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{Value: core.E("brain.forget", "failed to forget memory", nil), OK: false}
	})

	result := s.cmdBrainForget(core.NewOptions(core.Option{Key: "_arg", Value: "mem-1"}))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "failed to forget memory")
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
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdPRGet_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRGet(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
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
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdPRList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
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
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdPRClose_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRClose(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdPRClose_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "/api/v1/repos/core/go-io/pulls/5", r.URL.Path)

		bodyResult := core.ReadAll(r.Body)
		core.AssertTrue(t, bodyResult.OK)
		core.AssertContains(t, bodyResult.Value.(string), `"state":"closed"`)

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
	core.AssertTrue(t, r.OK)
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
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdIssueList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
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
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdRepoGet_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdRepoList_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdPRList_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRList(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommandsforge_CmdPRList_Good_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString([]map[string]any{})))
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdPRMerge_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRMerge(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

func TestCommandsforge_CmdRepoGet_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdRepoGet(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
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
	core.AssertTrue(t, r.OK)
}

// --- Workspace command methods ---

func TestCommandsworkspace_CmdWorkspaceList_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceList(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceList_Good_WithEntries(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-1")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex"}))

	r := s.cmdWorkspaceList(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceClean(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_RemovesCompleted(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-done")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex"}))

	r := s.cmdWorkspaceClean(core.NewOptions())
	core.AssertTrue(t, r.OK)

	core.AssertFalse(t, fs.Exists(ws).OK)
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
	core.AssertTrue(t, r.OK)

	core.AssertFalse(t, fs.Exists(core.JoinPath(wsRoot, "ws-bad")).OK)
	core.AssertTrue(t, fs.Exists(core.JoinPath(wsRoot, "ws-ok")).OK)
}

func TestCommandsworkspace_CmdWorkspaceClean_Good_FilterBlocked(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	d := core.JoinPath(wsRoot, "ws-stuck")
	fs.EnsureDir(d)
	fs.Write(core.JoinPath(d, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "blocked", Repo: "test", Agent: "codex"}))

	r := s.cmdWorkspaceClean(core.NewOptions(core.Option{Key: "_arg", Value: "blocked"}))
	core.AssertTrue(t, r.OK)

	core.AssertFalse(t, fs.Exists(d).OK)
}

func TestCommandsworkspace_CmdWorkspaceDispatch_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceDispatch(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommandsworkspace_CmdWorkspaceDispatch_Bad_MissingTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdWorkspaceDispatch(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertFalse(t, r.OK) // task is required
}

// --- commands.go extracted methods ---

func TestCommands_CmdPrep_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdPrep_Good_DefaultsToDev(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	// Will fail (no local clone) but exercises the default branch logic
	r := s.cmdPrep(core.NewOptions(core.Option{Key: "_arg", Value: "nonexistent-repo"}))
	core.AssertFalse(t, r.OK) // expected — no local repo
}

func TestCommands_CmdStatus_Good_Empty(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdStatus(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestCommands_CmdStatus_Good_WithWorkspaces(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	wsRoot := WorkspaceRoot()
	ws := core.JoinPath(wsRoot, "ws-1")
	fs.EnsureDir(ws)
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(WorkspaceStatus{Status: "completed", Repo: "test", Agent: "codex"}))

	r := s.cmdStatus(core.NewOptions())
	core.AssertTrue(t, r.OK)
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "completed")
	core.AssertContains(t, output, "codex")
	core.AssertContains(t, output, "go-io")
	core.AssertContains(t, output, "core/go-io/task-5")
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "completed")
	core.AssertContains(t, output, "core/go-io/feature/new-ui")
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "blocked")
	core.AssertContains(t, output, "gemini")
	core.AssertContains(t, output, "go-io")
	core.AssertContains(t, output, "Which API version?")
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "core/go-io/task-9")
	core.AssertNotContains(t, output, "core/go-log/task-4")
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "ws-1")
	core.AssertNotContains(t, output, "ws-2")
	core.AssertNotContains(t, output, "ws-3")
}

func TestCommands_CmdPrompt_Bad_MissingRepo(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrompt(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdPrompt_Good_DefaultTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrompt(core.NewOptions(core.Option{Key: "_arg", Value: "go-io"}))
	core.AssertTrue(t, r.OK)
}

func TestCommands_CmdPrompt_Good_PlanTemplateVariables(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CORE_HOME", home)

	repoDir := core.JoinPath(home, "Code", "core", "go-io")
	fs.EnsureDir(repoDir)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module example.com/go-io\n\ngo 1.24\n")

	s, _ := testPrepWithCore(t, nil)
	output := captureStdout(t, func() {
		r := s.cmdPrompt(core.NewOptions(
			core.Option{Key: "_arg", Value: "go-io"},
			core.Option{Key: "task", Value: "Add a login flow"},
			core.Option{Key: "plan_template", Value: "new-feature"},
			core.Option{Key: "variables", Value: `{"feature_name":"Authentication"}`},
		))
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "PLAN:")
	core.AssertContains(t, output, "Authentication")
}

func TestCommands_CmdGenerate_Bad_MissingPrompt(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdGenerate(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdGenerate_Good_Case(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/generate", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"data":{"id":"gen_1","provider":"claude","model":"claude-3.7-sonnet","content":"Release notes draft","input_tokens":12,"output_tokens":48,"status":"completed"}}`))
	}))
	defer server.Close()

	s := testPrepWithPlatformServer(t, server, "secret-token")
	output := captureStdout(t, func() {
		r := s.cmdGenerate(core.NewOptions(
			core.Option{Key: "_arg", Value: "Draft a release note"},
			core.Option{Key: "provider", Value: "claude"},
		))
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "provider: claude")
	core.AssertContains(t, output, "model:    claude-3.7-sonnet")
	core.AssertContains(t, output, "status:   completed")
	core.AssertContains(t, output, "content:  Release notes draft")
}

func TestCommands_CmdGenerate_Good_BriefTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/v1/content/generate", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)

		var payload map[string]any
		parseResult := core.JSONUnmarshalString(bodyResult.Value.(string), &payload)
		core.RequireTrue(t, parseResult.OK)
		core.AssertEqual(t, "brief_1", payload["brief_id"])
		core.AssertEqual(t, "help-article", payload["template"])

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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "provider: claude")
	core.AssertContains(t, output, "model:    claude-3.7-sonnet")
	core.AssertContains(t, output, "status:   completed")
	core.AssertContains(t, output, "content:  Template draft")
}

func TestCommands_CmdContentSchemaGenerate_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	output := captureStdout(t, func() {
		r := s.cmdContentSchemaGenerate(core.NewOptions(
			core.Option{Key: "type", Value: "howto"},
			core.Option{Key: "title", Value: "Set up the workspace"},
			core.Option{Key: "description", Value: "Prepare a fresh workspace for an agent."},
			core.Option{Key: "url", Value: "https://example.test/workspace"},
			core.Option{Key: "steps", Value: `[{"name":"Clone","text":"Clone the repository."},{"name":"Prepare","text":"Build the prompt."}]`},
		))
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "schema type: HowTo")
	core.AssertContains(t, output, `"@type":"HowTo"`)
	core.AssertContains(t, output, `"name":"Set up the workspace"`)
}

func TestCommands_CmdContentSchemaGenerate_Bad_MissingTitle(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdContentSchemaGenerate(core.NewOptions(
		core.Option{Key: "type", Value: "howto"},
	))
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdContentSchemaGenerate_Ugly_InvalidSchemaType(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdContentSchemaGenerate(core.NewOptions(
		core.Option{Key: "type", Value: "toast"},
		core.Option{Key: "title", Value: "Set up the workspace"},
	))
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdComplete_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	c.Action("test.noop", func(_ context.Context, _ core.Options) core.Result {
		return core.Result{OK: true}
	})
	c.Task("agent.completion", core.Task{
		Description: "QA → PR → Verify → Commit → Ingest → Poke",
		Steps: []core.Step{
			{Action: "test.noop"},
		},
	})

	r := s.cmdComplete(core.NewOptions(
		core.Option{Key: "workspace", Value: core.JoinPath(WorkspaceRoot(), "core/go-io/task-42")},
	))
	core.AssertTrue(t, r.OK)
}

func TestCommands_CmdComplete_Bad_MissingTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	r := s.cmdComplete(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdScan_Good_Case(t *testing.T) {
	server := mockScanServer(t)
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          newForgeClient(server.URL, "secret-token"),
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "count:")
	core.AssertContains(t, output, "go-io#10")
	core.AssertContains(t, output, "Add missing tests")
}

func TestCommands_CmdScan_Bad_NoForgeToken(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	s.forgeToken = ""

	r := s.cmdScan(core.NewOptions())
	core.AssertFalse(t, r.OK)
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
		forge:          newForgeClient(server.URL, "secret-token"),
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
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "count: 0")
}

func TestCommands_CmdPlanCreate_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	r := s.cmdPlanCreate(core.NewOptions(
		core.Option{Key: "slug", Value: "migrate-core"},
		core.Option{Key: "title", Value: "Migrate Core"},
		core.Option{Key: "objective", Value: "Use Core.Process everywhere"},
	))

	core.AssertTrue(t, r.OK)

	output, ok := r.Value.(PlanCreateOutput)
	core.RequireTrue(t, ok)
	core.RequireNotEmpty(t, output.ID)
	core.RequireNotEmpty(t, output.Path)
	core.AssertTrue(t, fs.Exists(output.Path).OK)

	plan, err := readPlan(PlansRoot(), output.ID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "Migrate Core", plan.Title)
	core.AssertEqual(t, "Use Core.Process everywhere", plan.Objective)
	core.AssertEqual(t, "draft", plan.Status)
	core.AssertEqual(t, "migrate-core", plan.Slug)
}

func TestCommands_CmdPlanStatus_Good_GetAndSet(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Status Plan",
		Objective: "Exercise plan status management",
	})
	core.RequireNoError(t, err)

	getOutput := captureStdout(t, func() {
		r := s.cmdPlanStatus(core.NewOptions(core.Option{Key: "_arg", Value: created.ID}))
		core.AssertTrue(t, r.OK)
	})
	core.AssertContains(t, getOutput, "status:")
	core.AssertContains(t, getOutput, "draft")

	setOutput := captureStdout(t, func() {
		r := s.cmdPlanStatus(core.NewOptions(
			core.Option{Key: "_arg", Value: created.ID},
			core.Option{Key: "set", Value: "ready"},
		))
		core.AssertTrue(t, r.OK)
	})
	core.AssertContains(t, setOutput, "status:")
	core.AssertContains(t, setOutput, "ready")

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "ready", plan.Status)
}

func TestCommands_CmdPlanArchive_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Archive Plan",
		Objective: "Exercise archive command",
	})
	core.RequireNoError(t, err)

	output := captureStdout(t, func() {
		r := s.cmdPlanArchive(core.NewOptions(
			core.Option{Key: "_arg", Value: created.ID},
		))
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "archived:")

	plan, err := readPlan(PlansRoot(), created.ID)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "archived", plan.Status)
	core.AssertFalse(t, plan.ArchivedAt.IsZero())
}

func TestCommands_CmdPlanDelete_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, created, err := s.planCreate(context.Background(), nil, PlanCreateInput{
		Title:     "Delete Plan",
		Objective: "Exercise delete command",
	})
	core.RequireNoError(t, err)

	output := captureStdout(t, func() {
		r := s.cmdPlanDelete(core.NewOptions(
			core.Option{Key: "_arg", Value: created.ID},
			core.Option{Key: "reason", Value: "RFC contract says soft delete"},
		))
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "deleted:")

	core.AssertFalse(t, fs.Exists(created.Path).OK)

	_, readErr := readPlan(PlansRoot(), created.ID)
	core.AssertError(t, readErr)
}

func TestCommands_CmdExtract_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	target := core.JoinPath(t.TempDir(), "extract-test")
	r := s.cmdExtract(core.NewOptions(
		core.Option{Key: "_arg", Value: "default"},
		core.Option{Key: "target", Value: target},
	))
	core.AssertTrue(t, r.OK)
}

func TestCommands_CmdExtract_Good_FromAgentOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	source := core.JoinPath(dir, "agent-output.md")
	target := core.JoinPath(dir, "extracted.json")
	core.RequireTrue(t, fs.Write(source, "Agent run complete.\n\n```json\n{\"summary\":\"done\",\"findings\":2}\n```\n").OK)

	output := captureStdout(t, func() {
		r := s.cmdExtract(core.NewOptions(
			core.Option{Key: "source", Value: source},
			core.Option{Key: "target", Value: target},
		))
		core.AssertTrue(t, r.OK)
		core.AssertEqual(t, "{\"summary\":\"done\",\"findings\":2}", r.Value)
	})

	core.AssertContains(t, output, "written: ")
	core.AssertTrue(t, fs.Exists(target).OK)
	written := fs.Read(target)
	core.RequireTrue(t, written.OK)
	core.AssertEqual(t, "{\"summary\":\"done\",\"findings\":2}", written.Value)
}

func TestCommands_CmdExtract_Bad_NoExtractableContent(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	source := core.JoinPath(dir, "agent-output.md")
	core.RequireTrue(t, fs.Write(source, "Agent run complete.\nNothing structured here.\n").OK)

	r := s.cmdExtract(core.NewOptions(core.Option{Key: "source", Value: source}))
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdRunTask_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startupContext = ctx
	r := s.cmdRunTask(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdDispatchSync_Bad_MissingArgs(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startupContext = ctx
	r := s.cmdDispatchSync(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdRunTask_Bad_MissingTask(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.startupContext = ctx
	r := s.cmdRunTask(core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
	core.AssertFalse(t, r.OK)
}

func TestCommands_CmdOrchestrator_Good_CancelledCtx(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestCommands_CmdDispatch_Good_CancelledCtx(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.startupContext = ctx
	r := s.cmdDispatch(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestCommands_CmdDispatchStart_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	called := false
	c.Action("runner.start", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	output := captureStdout(t, func() {
		r := s.cmdDispatchStart(core.NewOptions())
		core.AssertTrue(t, r.OK)
	})

	core.AssertTrue(t, called)
	core.AssertContains(t, output, "dispatch started")
}

func TestCommands_CmdDispatchShutdown_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	called := false
	c.Action("runner.stop", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	output := captureStdout(t, func() {
		r := s.cmdDispatchShutdown(core.NewOptions())
		core.AssertTrue(t, r.OK)
	})

	core.AssertTrue(t, called)
	core.AssertContains(t, output, "queue frozen")
}

func TestCommands_CmdDispatchShutdownNow_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	called := false
	c.Action("runner.kill", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	output := captureStdout(t, func() {
		r := s.cmdDispatchShutdownNow(core.NewOptions())
		core.AssertTrue(t, r.OK)
	})

	core.AssertTrue(t, called)
	core.AssertContains(t, output, "killed all agents")
}

func TestCommands_CmdPoke_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	called := false
	c.Action("runner.poke", func(_ context.Context, _ core.Options) core.Result {
		called = true
		return core.Result{OK: true}
	})

	output := captureStdout(t, func() {
		r := s.cmdPoke(core.NewOptions())
		core.AssertTrue(t, r.OK)
	})

	core.AssertTrue(t, called)
	core.AssertContains(t, output, "queue poke requested")
}

func TestCommands_ParseIntStr_Good_Case(t *testing.T) {
	core.AssertEqual(t, 42, parseIntString("42"))
	core.AssertEqual(t, 123, parseIntString("issue-123"))
	core.AssertEqual(t, 0, parseIntString(""))
	core.AssertEqual(t, 0, parseIntString("abc"))
	core.AssertEqual(t, 7, parseIntString("#7"))
}

// --- Registration verification ---

func TestCommands_RegisterCommands_Good_AllRegistered(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.registerCommands(ctx)

	cmds := c.Commands()
	core.AssertContains(t, cmds, "run/task")
	core.AssertContains(t, cmds, "agentic:run/task")
	core.AssertContains(t, cmds, "run/flow")
	core.AssertContains(t, cmds, "agentic:run/flow")
	core.AssertContains(t, cmds, "flow/preview")
	core.AssertContains(t, cmds, "agentic:flow/preview")
	core.AssertContains(t, cmds, "dispatch/sync")
	core.AssertContains(t, cmds, "agentic:dispatch/sync")
	core.AssertContains(t, cmds, "run/orchestrator")
	core.AssertContains(t, cmds, "agentic:run/orchestrator")
	core.AssertContains(t, cmds, "dispatch")
	core.AssertContains(t, cmds, "agentic:dispatch")
	core.AssertContains(t, cmds, "dispatch/start")
	core.AssertContains(t, cmds, "agentic:dispatch/start")
	core.AssertContains(t, cmds, "dispatch/shutdown")
	core.AssertContains(t, cmds, "agentic:dispatch/shutdown")
	core.AssertContains(t, cmds, "dispatch/shutdown-now")
	core.AssertContains(t, cmds, "agentic:dispatch/shutdown-now")
	core.AssertContains(t, cmds, "poke")
	core.AssertContains(t, cmds, "agentic:poke")
	core.AssertContains(t, cmds, "prep")
	core.AssertContains(t, cmds, "agentic:prep-workspace")
	core.AssertContains(t, cmds, "resume")
	core.AssertContains(t, cmds, "agentic:resume")
	core.AssertContains(t, cmds, "content/generate")
	core.AssertContains(t, cmds, "agentic:content/generate")
	core.AssertContains(t, cmds, "content/schema/generate")
	core.AssertContains(t, cmds, "agentic:content/schema/generate")
	core.AssertContains(t, cmds, "complete")
	core.AssertContains(t, cmds, "agentic:complete")
	core.AssertContains(t, cmds, "scan")
	core.AssertContains(t, cmds, "agentic:scan")
	core.AssertContains(t, cmds, "mirror")
	core.AssertContains(t, cmds, "agentic:mirror")
	core.AssertContains(t, cmds, "brain/ingest")
	core.AssertContains(t, cmds, "brain:ingest")
	core.AssertContains(t, cmds, "brain/seed-memory")
	core.AssertContains(t, cmds, "brain:seed-memory")
	core.AssertContains(t, cmds, "brain/list")
	core.AssertContains(t, cmds, "brain:list")
	core.AssertContains(t, cmds, "brain/forget")
	core.AssertContains(t, cmds, "brain:forget")
	core.AssertContains(t, cmds, "status")
	core.AssertContains(t, cmds, "agentic:status")
	core.AssertContains(t, cmds, "prompt")
	core.AssertContains(t, cmds, "agentic:prompt")
	core.AssertContains(t, cmds, "prompt_version")
	core.AssertContains(t, cmds, "agentic:prompt_version")
	core.AssertContains(t, cmds, "prompt/version")
	core.AssertContains(t, cmds, "extract")
	core.AssertContains(t, cmds, "agentic:extract")
	core.AssertContains(t, cmds, "lang/detect")
	core.AssertContains(t, cmds, "agentic:lang/detect")
	core.AssertContains(t, cmds, "lang/list")
	core.AssertContains(t, cmds, "agentic:lang/list")
	core.AssertContains(t, cmds, "epic")
	core.AssertContains(t, cmds, "agentic:epic")
	core.AssertContains(t, cmds, "plan")
	core.AssertContains(t, cmds, "agentic:plan/create")
	core.AssertContains(t, cmds, "plan/create")
	core.AssertContains(t, cmds, "agentic:plan/list")
	core.AssertContains(t, cmds, "plan/list")
	core.AssertContains(t, cmds, "agentic:plan/read")
	core.AssertContains(t, cmds, "plan/read")
	core.AssertContains(t, cmds, "agentic:plan/show")
	core.AssertContains(t, cmds, "plan/show")
	core.AssertContains(t, cmds, "agentic:plan/status")
	core.AssertContains(t, cmds, "plan/update")
	core.AssertContains(t, cmds, "agentic:plan/update")
	core.AssertContains(t, cmds, "agentic:plan/check")
	core.AssertContains(t, cmds, "plan/status")
	core.AssertContains(t, cmds, "plan/check")
	core.AssertContains(t, cmds, "agentic:plan/archive")
	core.AssertContains(t, cmds, "plan/archive")
	core.AssertContains(t, cmds, "agentic:plan/delete")
	core.AssertContains(t, cmds, "plan/delete")
	core.AssertContains(t, cmds, "agentic:plan-cleanup")
	core.AssertContains(t, cmds, "commit")
	core.AssertContains(t, cmds, "agentic:commit")
	core.AssertContains(t, cmds, "session/start")
	core.AssertContains(t, cmds, "session/get")
	core.AssertContains(t, cmds, "agentic:session/get")
	core.AssertContains(t, cmds, "session/list")
	core.AssertContains(t, cmds, "agentic:session/list")
	core.AssertContains(t, cmds, "agentic:session/start")
	core.AssertContains(t, cmds, "session/continue")
	core.AssertContains(t, cmds, "agentic:session/continue")
	core.AssertContains(t, cmds, "session/end")
	core.AssertContains(t, cmds, "agentic:session/end")
	core.AssertContains(t, cmds, "pr-manage")
	core.AssertContains(t, cmds, "agentic:pr-manage")
	core.AssertContains(t, cmds, "review-queue")
	core.AssertContains(t, cmds, "agentic:review-queue")
	core.AssertContains(t, cmds, "task")
	core.AssertContains(t, cmds, "phase")
	core.AssertContains(t, cmds, "agentic:phase")
	core.AssertContains(t, cmds, "phase/get")
	core.AssertContains(t, cmds, "agentic:phase/get")
	core.AssertContains(t, cmds, "phase/update_status")
	core.AssertContains(t, cmds, "agentic:phase/update_status")
	core.AssertContains(t, cmds, "phase/add_checkpoint")
	core.AssertContains(t, cmds, "agentic:phase/add_checkpoint")
	core.AssertContains(t, cmds, "task/update")
	core.AssertContains(t, cmds, "task/toggle")
	core.AssertContains(t, cmds, "sprint")
	core.AssertContains(t, cmds, "sprint/create")
}

func TestCommands_CmdPRManage_Good_NoCandidates(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPRManage(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

func TestCommands_CmdMirror_Good_NoRepos(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	output := captureStdout(t, func() {
		r := s.cmdMirror(core.NewOptions())
		core.AssertTrue(t, r.OK)
	})

	core.AssertContains(t, output, "count: 0")
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
	core.AssertTrue(t, r.OK)
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
	core.AssertFalse(t, r.OK)
}

// --- CmdOrchestrator Bad/Ugly ---

func TestCommands_CmdOrchestrator_Bad_DoneContext(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	core.AssertTrue(t, r.OK) // returns OK after ctx.Done()
}

func TestCommands_CmdOrchestrator_Ugly_CancelledImmediately(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	core.AssertTrue(t, r.OK) // exits immediately when context is already cancelled
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
		core.Option{Key: "plan_template", Value: "new-feature"},
		core.Option{Key: "variables", Value: `{"feature_name":"Authentication"}`},
		core.Option{Key: "persona", Value: "engineering"},
		core.Option{Key: "dry-run", Value: "true"},
	))
	// Will fail (no local clone) but exercises all option parsing paths
	core.AssertFalse(t, r.OK)
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
	core.AssertTrue(t, r.OK)
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
	core.AssertFalse(t, r.OK)
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
	core.AssertFalse(t, r.OK)
}

// --- CommandContext Good/Bad/Ugly ---

func TestCommands_CommandContext_Good_StoredStartupContext(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.registerCommands(ctx)
	core.AssertSame(t, ctx, s.commandContext())
}

func TestCommands_CommandContext_Bad_FallsBackToBackground(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	core.AssertNotNil(t, s.commandContext())
	core.AssertNoError(t, s.commandContext().Err())
}

func TestCommands_CommandContext_Ugly_CancelledStartupContext(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled
	s.startupContext = ctx
	r := s.cmdOrchestrator(core.NewOptions())
	core.AssertTrue(t, r.OK)
}

// --- CmdStatus Bad/Ugly ---

func TestCommands_CmdStatus_Bad_NoWorkspaceDir(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	// Don't create workspace dir — WorkspaceRoot() returns root+"/workspace" which won't exist

	c := core.New()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.cmdStatus(core.NewOptions())
	core.AssertTrue(t, r.OK) // returns OK with "no workspaces found"
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
	core.AssertTrue(t, r.OK)
}

// --- ParseIntStr Bad/Ugly ---

func TestCommands_ParseIntStr_Bad_NegativeAndOverflow(t *testing.T) {
	// parseIntString extracts digits only, ignoring minus signs
	core.AssertEqual(t, 5, parseIntString("-5"))  // extracts "5", ignores "-"
	core.AssertEqual(t, 0, parseIntString("-"))   // no digits
	core.AssertEqual(t, 0, parseIntString("---")) // no digits
}

func TestCommands_ParseIntStr_Ugly_UnicodeAndMixed(t *testing.T) {
	// Unicode digits (e.g. Arabic-Indic) are NOT ASCII 0-9 so ignored
	core.AssertEqual(t, 0, parseIntString("\u0661\u0662\u0663")) // ١٢٣ — not ASCII digits
	core.AssertEqual(t, 42, parseIntString("abc42xyz"))          // mixed chars
	core.AssertEqual(t, 123, parseIntString("1a2b3c"))           // interleaved
	core.AssertEqual(t, 0, parseIntString("  \t\n"))             // whitespace only
}
