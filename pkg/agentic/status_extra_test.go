// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- status tool ---

func TestStatus_Good_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(filepath.Join(root, "workspace")).OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 0, out.Total)
	assert.Equal(t, 0, out.Running)
	assert.Equal(t, 0, out.Completed)
}

func TestStatus_Good_MixedWorkspaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create completed workspace (old layout)
	ws1 := filepath.Join(wsRoot, "task-1")
	require.True(t, fs.EnsureDir(ws1).OK)
	require.NoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	// Create failed workspace (old layout)
	ws2 := filepath.Join(wsRoot, "task-2")
	require.True(t, fs.EnsureDir(ws2).OK)
	require.NoError(t, writeStatus(ws2, &WorkspaceStatus{
		Status: "failed",
		Repo:   "go-log",
		Agent:  "claude",
	}))

	// Create blocked workspace (old layout)
	ws3 := filepath.Join(wsRoot, "task-3")
	require.True(t, fs.EnsureDir(ws3).OK)
	require.NoError(t, writeStatus(ws3, &WorkspaceStatus{
		Status:   "blocked",
		Repo:     "agent",
		Agent:    "gemini",
		Question: "Which API version?",
	}))

	// Create queued workspace (old layout)
	ws4 := filepath.Join(wsRoot, "task-4")
	require.True(t, fs.EnsureDir(ws4).OK)
	require.NoError(t, writeStatus(ws4, &WorkspaceStatus{
		Status: "queued",
		Repo:   "go-mcp",
		Agent:  "codex",
	}))

	s := &PrepSubsystem{
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

func TestStatus_Good_DeepLayout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create workspace in deep layout (org/repo/task)
	ws := filepath.Join(wsRoot, "core", "go-io", "task-15")
	require.True(t, fs.EnsureDir(ws).OK)
	require.NoError(t, writeStatus(ws, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
	}))

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, 1, out.Completed)
}

func TestStatus_Good_CorruptStatusFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "corrupt-ws")
	require.True(t, fs.EnsureDir(ws).OK)
	require.True(t, fs.Write(filepath.Join(ws, "status.json"), "invalid-json{{{").OK)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Equal(t, 1, out.Failed) // corrupt status counts as failed
}

// --- shutdown tools ---

func TestDispatchStart_Good(t *testing.T) {
	s := &PrepSubsystem{
		frozen:    true,
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.dispatchStart(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.False(t, s.frozen)
	assert.Contains(t, out.Message, "started")
}

func TestShutdownGraceful_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownGraceful(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.True(t, s.frozen)
	assert.Contains(t, out.Message, "frozen")
}

func TestShutdownNow_Good_EmptyWorkspace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	require.True(t, fs.EnsureDir(filepath.Join(root, "workspace")).OK)

	s := &PrepSubsystem{
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.True(t, s.frozen)
	assert.Contains(t, out.Message, "killed 0")
}

func TestShutdownNow_Good_ClearsQueued(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create queued workspaces
	for i := 1; i <= 3; i++ {
		ws := filepath.Join(wsRoot, "task-"+itoa(i))
		require.True(t, fs.EnsureDir(ws).OK)
		require.NoError(t, writeStatus(ws, &WorkspaceStatus{
			Status: "queued",
			Repo:   "go-io",
			Agent:  "codex",
		}))
	}

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.shutdownNow(context.Background(), nil, ShutdownInput{})
	require.NoError(t, err)
	assert.Contains(t, out.Message, "cleared 3")

	// Verify queued workspaces are now failed
	for i := 1; i <= 3; i++ {
		ws := filepath.Join(wsRoot, "task-"+itoa(i))
		st, err := ReadStatus(ws)
		require.NoError(t, err)
		assert.Equal(t, "failed", st.Status)
		assert.Contains(t, st.Question, "cleared by shutdown_now")
	}
}

// --- brainRecall ---

func TestBrainRecall_Good_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/v1/brain/recall")

		json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{
				{"type": "architecture", "content": "Core uses DI pattern", "project": "go-core"},
				{"type": "convention", "content": "Use E() for errors", "project": "go-core"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 2, count)
	assert.Contains(t, result, "Core uses DI pattern")
	assert.Contains(t, result, "Use E() for errors")
}

func TestBrainRecall_Good_NoMemories(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"memories": []map[string]any{},
		})
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 0, count)
	assert.Empty(t, result)
}

func TestBrainRecall_Bad_NoBrainKey(t *testing.T) {
	s := &PrepSubsystem{
		brainKey:  "",
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 0, count)
	assert.Empty(t, result)
}

func TestBrainRecall_Bad_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		brainURL:   srv.URL,
		brainKey:   "test-brain-key",
		client:     srv.Client(),
		backoff:    make(map[string]time.Time),
		failCount:  make(map[string]int),
	}

	result, count := s.brainRecall(context.Background(), "go-core")
	assert.Equal(t, 0, count)
	assert.Empty(t, result)
}

// --- prepWorkspace ---

func TestPrepWorkspace_Bad_NoRepo(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, _, err := s.prepWorkspace(context.Background(), nil, PrepInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestPrepWorkspace_Bad_NoIdentifier(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
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

func TestPrepWorkspace_Bad_InvalidRepoName(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
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

func TestListPRs_Good_SpecificRepo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock PRs
		json.NewEncoder(w).Encode([]map[string]any{
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
		})
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

func TestPoke_Good_SendsSignal(t *testing.T) {
	s := &PrepSubsystem{
		pokeCh:    make(chan struct{}, 1),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	s.Poke()
	// Should have something in the channel
	select {
	case <-s.pokeCh:
		// ok
	default:
		t.Fatal("expected poke signal in channel")
	}
}

func TestPoke_Good_NonBlocking(t *testing.T) {
	s := &PrepSubsystem{
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

func TestPoke_Bad_NilChannel(t *testing.T) {
	s := &PrepSubsystem{
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

func TestWriteReadStatus_Good_WithPID(t *testing.T) {
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

func TestWriteReadStatus_Good_AllFields(t *testing.T) {
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

func TestOnShutdown_Good(t *testing.T) {
	s := &PrepSubsystem{
		frozen:    false,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	err := s.OnShutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, s.frozen)
}

// --- drainQueue ---

func TestDrainQueue_Good_FrozenDoesNothing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		frozen:    true,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	// Should return immediately when frozen
	assert.NotPanics(t, func() {
		s.drainQueue()
	})
}
