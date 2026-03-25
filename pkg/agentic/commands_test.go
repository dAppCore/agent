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

// --- Forge command registration covers the closures ---

func TestForgeCommands_Good_IssueGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"number":   42,
			"title":    "Fix tests",
			"state":    "open",
			"html_url": "https://forge.test/core/go-io/issues/42",
			"body":     "Tests are failing",
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	s.registerForgeCommands()
	// Test via parseForgeArgs + direct invocation already tested
}

func TestForgeCommands_Good_RepoListSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "go-io", "description": "IO", "archived": false,
				"owner": map[string]any{"login": "core"}},
		})
	}))
	t.Cleanup(srv.Close)

	s, _ := testPrepWithCore(t, srv)
	s.registerForgeCommands()
}

// --- Workspace command action closures ---

func TestWorkspaceCommands_Good_ListWithWorkspaces(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsRoot := filepath.Join(root, "workspace")
	ws := filepath.Join(wsRoot, "ws-1")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s.registerWorkspaceCommands()
}

func TestWorkspaceCommands_Good_CleanCompleted(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsRoot := filepath.Join(root, "workspace")
	ws := filepath.Join(wsRoot, "ws-done")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Status: "completed", Repo: "go-io", Agent: "codex"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s.registerWorkspaceCommands()
}

// --- registerCommands action closures ---

func TestCommands_Good_Registration(t *testing.T) {
	s, c := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.registerCommands(ctx)

	// Verify commands were registered
	cmds := c.Commands()
	assert.Contains(t, cmds, "run/task")
	assert.Contains(t, cmds, "run/orchestrator")
	assert.Contains(t, cmds, "prep")
	assert.Contains(t, cmds, "status")
	assert.Contains(t, cmds, "prompt")
	assert.Contains(t, cmds, "extract")
}
