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

func TestStatus_EmptyWorkspace_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 0, out.Total)
	core.AssertEqual(t, 0, out.Running)
	core.AssertEqual(t, 0, out.Completed)
}

func TestStatus_MixedWorkspaces_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	// Create completed workspace (old layout)
	ws1 := core.JoinPath(wsRoot, "task-1")
	core.RequireTrue(t, fs.EnsureDir(ws1).OK)
	core.RequireNoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	// Create failed workspace (old layout)
	ws2 := core.JoinPath(wsRoot, "task-2")
	core.RequireTrue(t, fs.EnsureDir(ws2).OK)
	core.RequireNoError(t, writeStatus(ws2, &WorkspaceStatus{
		Status: "failed",
		Repo:   "go-log",
		Agent:  "claude",
	}))

	// Create blocked workspace (old layout)
	ws3 := core.JoinPath(wsRoot, "task-3")
	core.RequireTrue(t, fs.EnsureDir(ws3).OK)
	core.RequireNoError(t, writeStatus(ws3, &WorkspaceStatus{
		Status:   "blocked",
		Repo:     "agent",
		Agent:    "gemini",
		Question: "Which API version?",
	}))

	// Create queued workspace (old layout)
	ws4 := core.JoinPath(wsRoot, "task-4")
	core.RequireTrue(t, fs.EnsureDir(ws4).OK)
	core.RequireNoError(t, writeStatus(ws4, &WorkspaceStatus{
		Status: "queued",
		Repo:   "go-mcp",
		Agent:  "codex",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 4, out.Total)
	core.AssertEqual(t, 1, out.Completed)
	core.AssertEqual(t, 1, out.Failed)
	core.AssertEqual(t, 1, out.Queued)
	core.AssertLen(t, out.Blocked, 1)
	core.AssertEqual(t, "Which API version?", out.Blocked[0].Question)
	core.AssertEqual(t, "agent", out.Blocked[0].Repo)
}

func TestStatus_FilteredWorkspaces_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	ws1 := core.JoinPath(wsRoot, "task-1")
	core.RequireTrue(t, fs.EnsureDir(ws1).OK)
	core.RequireNoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	ws2 := core.JoinPath(wsRoot, "task-2")
	core.RequireTrue(t, fs.EnsureDir(ws2).OK)
	core.RequireNoError(t, writeStatus(ws2, &WorkspaceStatus{
		Status:   "blocked",
		Repo:     "go-log",
		Agent:    "claude",
		Question: "Which log format?",
	}))

	ws3 := core.JoinPath(wsRoot, "task-3")
	core.RequireTrue(t, fs.EnsureDir(ws3).OK)
	core.RequireNoError(t, writeStatus(ws3, &WorkspaceStatus{
		Status: "running",
		Repo:   "agent",
		Agent:  "gemini",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{
		Workspace: "task-2",
		Status:    "blocked",
		Limit:     1,
	})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, out.Total)
	core.AssertEqual(t, 0, out.Failed)
	core.AssertEqual(t, 0, out.Completed)
	core.AssertLen(t, out.Blocked, 1)
	core.AssertEqual(t, "go-log", out.Blocked[0].Repo)
	core.AssertEqual(t, "Which log format?", out.Blocked[0].Question)
}

func TestStatus_DeepLayout_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	// Create workspace in deep layout (org/repo/task)
	ws := core.JoinPath(wsRoot, "core", "go-io", "task-15")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, out.Total)
	core.AssertEqual(t, 1, out.Completed)
}

func TestStatus_CorruptStatus_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "corrupt-ws")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(ws, "status.json"), "invalid-json{{{").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 1, out.Total)
	core.AssertEqual(t, 1, out.Failed) // corrupt status counts as failed
}

// --- shutdown tools ---

func TestShutdown_DispatchStart_Good_Case(t *testing.T) {
	// dispatchStart delegates to runner.start Action — verify it calls the Action and returns success.
	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         true,
		pokeCh:         make(chan struct{}, 1),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.dispatchStart(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "started")
}

func TestShutdown_ShutdownGraceful_Good_Case(t *testing.T) {
	// shutdownGraceful delegates to runner.stop Action — verify it returns success and frozen message.
	root := t.TempDir()
	setTestWorkspace(t, root)

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.shutdownGraceful(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "frozen")
}

func TestShutdown_ShutdownNow_Good_EmptyWorkspace(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — verify it returns success.
	root := t.TempDir()
	setTestWorkspace(t, root)

	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(root, "workspace")).OK)

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "killed all agents, cleared queue")
}

func TestShutdown_ShutdownNow_Good_ClearsQueued(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — queue clearing is now
	// handled by the runner service. Verify the delegation returns success.
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	// Create queued workspaces (runner.kill would clear these in production)
	for i := 1; i <= 3; i++ {
		ws := core.JoinPath(wsRoot, "task-"+itoa(i))
		core.RequireTrue(t, fs.EnsureDir(ws).OK)
		core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
			Status: "queued",
			Repo:   "go-io",
			Agent:  "codex",
		}))
	}

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "killed all agents, cleared queue")
}

// --- brainRecall ---

func TestPrep_BrainRecall_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertContains(t, r.URL.Path, "/v1/brain/recall")

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
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	core.AssertEqual(t, 2, count)
	core.AssertContains(t, result, "Core uses DI pattern")
	core.AssertContains(t, result, "Use E() for errors")
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
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	core.AssertEqual(t, 0, count)
	core.AssertEmpty(t, result)
}

func TestPrep_BrainRecall_Bad_NoBrainKey(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainKey:       "",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	core.AssertEqual(t, 0, count)
	core.AssertEmpty(t, result)
}

func TestPrep_BrainRecall_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		brainURL:       srv.URL,
		brainKey:       "test-brain-key",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	core.AssertEqual(t, 0, count)
	core.AssertEmpty(t, result)
}

// --- prepWorkspace ---

func TestPrep_PrepWorkspace_Bad_NoRepo(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "repo is required")
}

func TestPrep_PrepWorkspace_Bad_NoIdentifier(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo: "go-io",
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "one of issue, pr, branch, or tag is required")
}

func TestPrep_PrepWorkspace_Bad_InvalidRepoName(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{
		Repo:  "..",
		Issue: 1,
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid repo name")
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
		forge:          forge.NewForge(srv.URL, "test-token"),
		forgeURL:       srv.URL,
		forgeToken:     "test-token",
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.listPRs(context.Background(), nil, ListPRsInput{
		Repo: "go-io",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, 1, out.Count)
	core.AssertEqual(t, "Fix tests", out.PRs[0].Title)
	core.AssertEqual(t, "virgil", out.PRs[0].Author)
	core.AssertEqual(t, "agent/fix-tests", out.PRs[0].Branch)
	core.AssertContains(t, out.PRs[0].Labels, "agentic")
}

// --- Poke ---

func TestRunner_Poke_Good_SendsSignal(t *testing.T) {
	// Poke is now a no-op — queue poke is owned by pkg/runner.Service.
	// Verify it does not panic and does not send to the channel.
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		pokeCh:         make(chan struct{}, 1),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	core.AssertNotPanics(t, func() { s.Poke() })
	core.AssertLen(t, s.pokeCh, 0, "no-op Poke should not send to channel")
}

func TestRunner_Poke_Good_NonBlocking(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		pokeCh:         make(chan struct{}, 1),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Fill the channel
	s.pokeCh <- struct{}{}

	// Second poke should not block
	core.AssertNotPanics(t, func() {
		s.Poke()
	})
}

func TestRunner_Poke_Bad_NilChannel(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		pokeCh:         nil,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should not panic with nil channel
	core.AssertNotPanics(t, func() {
		s.Poke()
	})
}

// --- ReadStatusResult / writeStatus (extended) ---

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
	core.RequireNoError(t, err)

	// Read it back
	got := mustReadStatus(t, dir)
	core.AssertEqual(t, "running", got.Status)
	core.AssertEqual(t, "codex", got.Agent)
	core.AssertEqual(t, "go-io", got.Repo)
	core.AssertEqual(t, 12345, got.PID)
	core.AssertFalse(t, got.UpdatedAt.IsZero())
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
	core.RequireNoError(t, err)

	got := mustReadStatus(t, dir)
	core.AssertEqual(t, "blocked", got.Status)
	core.AssertEqual(t, "claude", got.Agent)
	core.AssertEqual(t, "core", got.Org)
	core.AssertEqual(t, 42, got.Issue)
	core.AssertEqual(t, "Which log format?", got.Question)
	core.AssertEqual(t, 3, got.Runs)
	core.AssertEqual(t, "https://forge.test/core/go-log/pulls/5", got.PRURL)
}

// --- OnStartup / OnShutdown ---

func TestPrep_OnShutdown_Good(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	r := s.OnShutdown(context.Background())
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, s.frozen)
}

// --- drainQueue ---

func TestQueue_DrainQueue_Good_FrozenDoesNothing(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:         true,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Should return immediately when frozen
	core.AssertNotPanics(t, func() {
		s.drainQueue()
	})
}

// --- shutdownNow (Ugly — deep layout with queued status) ---

func TestShutdown_ShutdownNow_Ugly_DeepLayout(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — queue clearing is now
	// handled by the runner service. Verify delegation with deep-layout workspaces.
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	// Create workspace in deep layout (org/repo/task)
	ws := core.JoinPath(wsRoot, "core", "go-io", "task-5")
	core.RequireTrue(t, fs.EnsureDir(ws).OK)
	core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "queued",
		Repo:   "go-io",
		Agent:  "codex",
		Task:   "Add tests",
	}))

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "killed all agents, cleared queue")
}

// --- dispatchStart Bad/Ugly ---

func TestShutdown_DispatchStart_Bad_NilPokeCh(t *testing.T) {
	// dispatchStart delegates to runner.start Action — verify it succeeds with nil pokeCh.
	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         true,
		pokeCh:         nil,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.dispatchStart(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "started")
}

func TestShutdown_DispatchStart_Ugly_AlreadyUnfrozen(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:         false, // already unfrozen
		pokeCh:         make(chan struct{}, 1),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.dispatchStart(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertFalse(t, s.frozen, "should remain unfrozen")
	core.AssertContains(t, out.Message, "started")
}

// --- shutdownGraceful Bad/Ugly ---

func TestShutdown_ShutdownGraceful_Bad_AlreadyFrozen(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		frozen:         true, // already frozen
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.shutdownGraceful(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertTrue(t, s.frozen, "should remain frozen")
	core.AssertContains(t, out.Message, "frozen")
}

func TestShutdown_ShutdownGraceful_Ugly_WithWorkspaces(t *testing.T) {
	// shutdownGraceful delegates to runner.stop Action — verify it returns success
	// even when workspaces exist.
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	// Create workspaces with various statuses
	for _, name := range []string{"ws-completed", "ws-failed", "ws-blocked"} {
		ws := core.JoinPath(wsRoot, name)
		core.RequireTrue(t, fs.EnsureDir(ws).OK)
		core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
			Status: "completed",
			Repo:   "go-io",
			Agent:  "codex",
		}))
	}

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.shutdownGraceful(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "frozen")
}

// --- shutdownNow Bad ---

func TestShutdown_ShutdownNow_Bad_NoRunningPIDs(t *testing.T) {
	// shutdownNow delegates to runner.kill Action — verify it returns success
	// even when there are no running PIDs. Kill counting is now in pkg/runner.
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsRoot := core.JoinPath(root, "workspace")

	// Create completed workspaces only (no running PIDs to kill)
	for i := 1; i <= 2; i++ {
		ws := core.JoinPath(wsRoot, "task-"+itoa(i))
		core.RequireTrue(t, fs.EnsureDir(ws).OK)
		core.RequireNoError(t, writeStatus(ws, &WorkspaceStatus{
			Status: "completed",
			Repo:   "go-io",
			Agent:  "codex",
		}))
	}

	c := coreWithRunnerActions()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		frozen:         false,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertContains(t, out.Message, "killed all agents, cleared queue")
}
