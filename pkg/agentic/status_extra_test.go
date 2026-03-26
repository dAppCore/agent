// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// coreWithRunnerActions builds a Core with stub runner.start/stop/kill Actions
// so shutdown tool tests can verify delegation without a real runner service.
func coreWithRunnerActions() *core.Core {
	c := core.New()
	c.Action("runner.start", func(_ context.Context, _ core.Options) core.Result { return core.Result{OK: true} })
	c.Action("runner.stop", func(_ context.Context, _ core.Options) core.Result { return core.Result{OK: true} })
	c.Action("runner.kill", func(_ context.Context, _ core.Options) core.Result { return core.Result{OK: true} })
	return c
}

// --- status tool ---

func TestStatus_EmptyWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 0, out.Total)
	assert.Equal(t, 0, out.Running)
	assert.Equal(t, 0, out.Completed)
}

func TestStatus_MixedWorkspaces_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create completed workspace (old layout)
	ws1 := core.JoinPath(wsRoot, "task-1")
	require.True(t, fs.EnsureDir(ws1).OK)
	require.NoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	// Create failed workspace (old layout)
	ws2 := core.JoinPath(wsRoot, "task-2")
	require.True(t, fs.EnsureDir(ws2).OK)
	require.NoError(t, writeStatus(ws2, &WorkspaceStatus{
		Status: "failed",
		Repo:   "go-log",
		Agent:  "claude",
	}))

	// Create blocked workspace (old layout)
	ws3 := core.JoinPath(wsRoot, "task-3")
	require.True(t, fs.EnsureDir(ws3).OK)
	require.NoError(t, writeStatus(ws3, &WorkspaceStatus{
		Status:   "blocked",
		Repo:     "agent",
		Agent:    "gemini",
		Question: "Which API version?",
	}))

	// Create queued workspace (old layout)
	ws4 := core.JoinPath(wsRoot, "task-4")
	require.True(t, fs.EnsureDir(ws4).OK)
	require.NoError(t, writeStatus(ws4, &WorkspaceStatus{
		Status: "queued",
		Repo:   "go-mcp",
		Agent:  "codex",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 4, out.Total)
	assert.Equal(t, 1, out.Completed)
	assert.Equal(t, 1, out.Failed)
	assert.Equal(t, 1, out.Queued)
	assert.Len(t, out.Blocked, 1)
	assert.Equal(t, "Which API version?", out.Blocked[0].Question)
	assert.Equal(t, "agent", out.Blocked[0].Repo)
}

func TestStatus_DeepLayout_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create workspace in deep layout (org/repo/task)
	ws := core.JoinPath(wsRoot, "core", "go-io", "task-15")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, 1, out.Completed)
}

func TestStatus_CorruptStatus_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "corrupt-ws")
	require.True(t, fs.EnsureDir(ws).OK)
	require.True(t, fs.Write(core.JoinPath(ws, "status.json"), "invalid-json{{{").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, 1, out.Failed) // corrupt status counts as failed
}

// --- shutdown tools ---

func TestShutdown_DispatchStart_Good(t *testing.T) {
	// dispatchStart delegates to runner.start Action — verify it calls the Action and returns success.
	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:    true,
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.dispatchStart(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "started")
}

func TestShutdown_ShutdownGraceful_Good(t *testing.T) {
	// shutdownGraceful delegates to runner.stop Action — verify it returns success and frozen message.
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownGraceful(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "frozen")
}

func TestShutdown_ShutdownNow_Good_EmptyWorkspace(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — verify it returns success.
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "killed all agents, cleared queue")
}

func TestShutdown_ShutdownNow_Good_ClearsQueued(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — queue clearing is now
	// handled by the runner service. Verify the delegation returns success.
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create queued workspaces (runner.kill would clear these in production)
	for i := 1; i <= 3; i++ {
		ws := core.JoinPath(wsRoot, "task-"+itoa(i))
		require.True(t, fs.EnsureDir(ws).OK)
		require.NoError(t, writeStatus(ws, &WorkspaceStatus{
			Status: "queued",
			Repo:   "go-io",
			Agent:  "codex",
		}))
	}

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "killed all agents, cleared queue")
}

// --- brainRecall ---

func TestPrep_BrainRecall_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/brain/recall")

		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"memories": []map[string]any{
				{"type": "architecture", "content": "Core uses DI pattern", "project": "go-core"},
				{"type": "convention", "content": "Use E() for errors", "project": "go-core"},
			},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 2, count)
	assert.Contains(t, result, "Core uses DI pattern")
	assert.Contains(t, result, "Use E() for errors")
}

func TestPrep_BrainRecall_Good_NoMemories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"memories": []map[string]any{},
		})))
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 0, count)
	assert.Empty(t, result)
}

func TestPrep_BrainRecall_Bad_NoBrainKey(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainKey:  "",
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 0, count)
	assert.Empty(t, result)
}

func TestPrep_BrainRecall_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 0, count)
	assert.Empty(t, result)
}

// --- prepWorkspace ---

func TestPrep_PrepWorkspace_Bad_NoRepo(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestPrep_PrepWorkspace_Bad_NoIdentifier(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo: "go-io",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "one of issue, pr, branch, or tag is required")
}

func TestPrep_PrepWorkspace_Bad_InvalidRepoName(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo:  "..",
		Issue: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo name")
}

// --- listPRs ---

func TestPr_ListPRs_Good_SpecificRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock PRs
		w.Write([]byte(core.JSONMarshalString([]map[string]any{
			{
				"number":    1,
				"title":     "Fix tests",
				"state":     "open",
				"html_url":  "https://forge.test/core/go-io/pulls/1",
				"mergeable": true,
				"user":      map[string]any{"login": "virgil"},
				"head":      map[string]any{"ref": "agent/fix-tests"},
				"base":      map[string]any{"ref": "dev"},
				"labels":    []map[string]any{{"name": "agentic"}},
			},
		})))
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
		Repo: "go-io",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, 1, out.Count)
	assert.Equal(t, "Fix tests", out.PRs[0].Title)
	assert.Equal(t, "virgil", out.PRs[0].Author)
	assert.Equal(t, "agent/fix-tests", out.PRs[0].Branch)
	assert.Contains(t, out.PRs[0].Labels, "agentic")
}

// --- Poke ---

func TestRunner_Poke_Good_SendsSignal(t *testing.T) {
	// Poke is now a no-op — queue poke is owned by pkg/runner.Service.
	// Verify it does not panic and does not send to the channel.
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	assert.NotPanics(t, func() { s.Poke() })
	assert.Len(t, s.pokeCh, 0, "no-op Poke should not send to channel")
}

func TestRunner_Poke_Good_NonBlocking(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Fill the channel
	s.pokeCh <- struct{}{}

	// Second poke should not block
	assert.NotPanics(t, func() {
		s.Poke()
	})
}

func TestRunner_Poke_Bad_NilChannel(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		pokeCh:    nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Should not panic with nil channel
	assert.NotPanics(t, func() {
		s.Poke()
	})
}

// --- ReadStatus / writeStatus (extended) ---

func TestStatus_WriteRead_Good_WithPID(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "Fix it",
		PID:    12345,
	}

	err := writeStatus(dir, st)
	require.NoError(t, err)

	// Read it back
	got, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "running", got.Status)
	assert.Equal(t, "codex", got.Agent)
	assert.Equal(t, "go-io", got.Repo)
	assert.Equal(t, 12345, got.PID)
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestStatus_WriteRead_Good_AllFields(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	st := &WorkspaceStatus{
		Status:    "blocked",
		Agent:     "claude",
		Repo:      "go-log",
		Org:       "core",
		Task:      "Add structured logging",
		Branch:    "agent/add-logging",
		Issue:     42,
		PID:       99999,
		StartedAt: now,
		Question:  "Which log format?",
		Runs:      3,
		PRURL:     "https://forge.test/core/go-log/pulls/5",
	}

	err := writeStatus(dir, st)
	require.NoError(t, err)

	got, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "blocked", got.Status)
	assert.Equal(t, "claude", got.Agent)
	assert.Equal(t, "core", got.Org)
	assert.Equal(t, 42, got.Issue)
	assert.Equal(t, "Which log format?", got.Question)
	assert.Equal(t, 3, got.Runs)
	assert.Equal(t, "https://forge.test/core/go-log/pulls/5", got.PRURL)
}

// --- OnStartup / OnShutdown ---

func TestPrep_OnShutdown_Good(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	r := s.OnShutdown(context.Background())
	assert.True(t, r.OK)
	assert.True(t, s.frozen)
}

// --- drainQueue ---

func TestQueue_DrainQueue_Good_FrozenDoesNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:    true,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Should return immediately when frozen
	assert.NotPanics(t, func() {
		s.drainQueue()
	})
}

// --- shutdownNow (Ugly — deep layout with queued status) ---

func TestPrep_Shutdown_ShutdownNow_Ugly(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — queue clearing is now
	// handled by the runner service. Verify delegation with deep-layout workspaces.
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create workspace in deep layout (org/repo/task)
	ws := core.JoinPath(wsRoot, "core", "go-io", "task-5")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "queued",
		Repo:   "go-io",
		Agent:  "codex",
		Task:   "Add tests",
	}))

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "killed all agents, cleared queue")
}

// --- dispatchStart Bad/Ugly ---

func TestShutdown_DispatchStart_Bad_NilPokeCh(t *testing.T) {
	// dispatchStart delegates to runner.start Action — verify it succeeds with nil pokeCh.
	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:    true,
		pokeCh:    nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.dispatchStart(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "started")
}

func TestShutdown_DispatchStart_Ugly_AlreadyUnfrozen(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:    false, // already unfrozen
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.dispatchStart(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.False(t, s.frozen, "should remain unfrozen")
	assert.Contains(t, out.Message, "started")
}

// --- shutdownGraceful Bad/Ugly ---

func TestShutdown_ShutdownGraceful_Bad_AlreadyFrozen(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:    true, // already frozen
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownGraceful(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.True(t, s.frozen, "should remain frozen")
	assert.Contains(t, out.Message, "frozen")
}

func TestShutdown_ShutdownGraceful_Ugly_WithWorkspaces(t *testing.T) {
	// shutdownGraceful delegates to runner.stop Action — verify it returns success
	// even when workspaces exist.
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create workspaces with various statuses
	for _, name := range []string{"ws-completed", "ws-failed", "ws-blocked"} {
		ws := core.JoinPath(wsRoot, name)
		require.True(t, fs.EnsureDir(ws).OK)
		require.NoError(t, writeStatus(ws, &WorkspaceStatus{
			Status: "completed",
			Repo:   "go-io",
			Agent:  "codex",
		}))
	}

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownGraceful(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "frozen")
}

// --- shutdownNow Bad ---

func TestShutdown_ShutdownNow_Bad_NoRunningPIDs(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — verify it returns success
	// even when there are no running PIDs. Kill counting is now in pkg/runner.
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create completed workspaces only (no running PIDs to kill)
	for i := 1; i <= 2; i++ {
		ws := core.JoinPath(wsRoot, "task-"+itoa(i))
		require.True(t, fs.EnsureDir(ws).OK)
		require.NoError(t, writeStatus(ws, &WorkspaceStatus{
			Status: "completed",
			Repo:   "go-io",
			Agent:  "codex",
		}))
	}

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Contains(t, out.Message, "killed all agents, cleared queue")
}
