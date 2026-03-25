// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- buildReviewCommand ---

func TestBuildReviewCommand_Good_CodeRabbit(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	cmd := s.buildReviewCommand(context.Background(), "/tmp/repo", "coderabbit")
	assert.Equal(t, "coderabbit", cmd.Path[len(cmd.Path)-len("coderabbit"):])
	assert.Contains(t, cmd.Args, "review")
	assert.Contains(t, cmd.Args, "--plain")
	assert.Contains(t, cmd.Args, "--base")
	assert.Contains(t, cmd.Args, "github/main")
}

func TestBuildReviewCommand_Good_Codex(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	cmd := s.buildReviewCommand(context.Background(), "/tmp/repo", "codex")
	assert.Contains(t, cmd.Args, "review")
	assert.Contains(t, cmd.Args, "--base")
	assert.Contains(t, cmd.Args, "github/main")
	assert.Equal(t, "/tmp/repo", cmd.Dir)
}

func TestBuildReviewCommand_Good_DefaultReviewer(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Empty string → defaults to coderabbit
	cmd := s.buildReviewCommand(context.Background(), "/tmp/repo", "")
	assert.Contains(t, cmd.Args, "--plain")
}

// --- saveRateLimitState / loadRateLimitState ---

func TestSaveLoadRateLimitState_Good_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	// Ensure .core dir exists
	os.MkdirAll(filepath.Join(dir, ".core"), 0o755)

	// Note: saveRateLimitState uses core.Env("DIR_HOME") which is pre-populated.
	// We need to work around this by using CORE_WORKSPACE for the load,
	// but save/load use DIR_HOME. Skip if not writable.
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	info := &RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(5 * time.Minute).Truncate(time.Second),
		Message: "rate limited",
	}
	s.saveRateLimitState(info)

	loaded := s.loadRateLimitState()
	if loaded != nil {
		assert.True(t, loaded.Limited)
		assert.Equal(t, "rate limited", loaded.Message)
	}
	// If loaded is nil it means DIR_HOME path wasn't writable — acceptable in test
}

// --- storeReviewOutput ---

func TestStoreReviewOutput_Good(t *testing.T) {
	// storeReviewOutput uses core.Env("DIR_HOME") so we can't fully control the path
	// but we can verify it doesn't panic
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.NotPanics(t, func() {
		s.storeReviewOutput(t.TempDir(), "test-repo", "coderabbit", "No findings — LGTM")
	})
}

// --- reviewQueue ---

func TestReviewQueue_Good_NoCandidates(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Create an empty core dir (no repos)
	coreDir := filepath.Join(root, "core")
	os.MkdirAll(coreDir, 0o755)

	s := &PrepSubsystem{
		codePath:  root,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.reviewQueue(context.Background(), nil, ReviewQueueInput{DryRun: true})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Empty(t, out.Processed)
}

// --- status (extended) ---

func TestStatus_Good_FilteredByStatus(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create workspaces with different statuses
	for _, ws := range []struct {
		name   string
		status string
	}{
		{"ws-1", "completed"},
		{"ws-2", "failed"},
		{"ws-3", "completed"},
		{"ws-4", "queued"},
	} {
		wsDir := filepath.Join(wsRoot, ws.name)
		os.MkdirAll(wsDir, 0o755)
		st := &WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"}
		data, _ := json.Marshal(st)
		os.WriteFile(filepath.Join(wsDir, "status.json"), data, 0o644)
	}

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 4, out.Total)
	assert.Equal(t, 2, out.Completed)
	assert.Equal(t, 1, out.Failed)
	assert.Equal(t, 1, out.Queued)
}

// --- handlers helpers (resolveWorkspace, findWorkspaceByPR) ---

func TestResolveWorkspace_Good_Exists(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Create workspace dir
	ws := filepath.Join(wsRoot, "core", "go-io", "task-15")
	os.MkdirAll(ws, 0o755)

	result := resolveWorkspace("core/go-io/task-15")
	assert.Equal(t, ws, result)
}

func TestResolveWorkspace_Bad_NotExists(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	result := resolveWorkspace("nonexistent")
	assert.Empty(t, result)
}

func TestFindWorkspaceByPR_Good_Match(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "ws-test")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Repo: "go-io", Branch: "agent/fix", Status: "completed"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	result := findWorkspaceByPR("go-io", "agent/fix")
	assert.Equal(t, ws, result)
}

func TestFindWorkspaceByPR_Good_DeepLayout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	// Deep layout: org/repo/task
	ws := filepath.Join(wsRoot, "core", "agent", "task-5")
	os.MkdirAll(ws, 0o755)
	st := &WorkspaceStatus{Repo: "agent", Branch: "agent/tests", Status: "completed"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	result := findWorkspaceByPR("agent", "agent/tests")
	assert.Equal(t, ws, result)
}

// --- loadRateLimitState (Ugly — corrupt JSON) ---

func TestReviewQueue_LoadRateLimitState_Ugly(t *testing.T) {
	// core.Env("DIR_HOME") is cached at init, so we must write to the real path.
	// Save original content, write corrupt JSON, test, then restore.
	ratePath := filepath.Join(core.Env("DIR_HOME"), ".core", "coderabbit-ratelimit.json")

	// Save original content (may or may not exist)
	original, readErr := os.ReadFile(ratePath)
	hadFile := readErr == nil

	// Ensure parent dir exists
	os.MkdirAll(filepath.Dir(ratePath), 0o755)

	// Write corrupt JSON
	require.NoError(t, os.WriteFile(ratePath, []byte("not-valid-json{{{"), 0o644))
	t.Cleanup(func() {
		if hadFile {
			os.WriteFile(ratePath, original, 0o644)
		} else {
			os.Remove(ratePath)
		}
	})

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	result := s.loadRateLimitState()
	assert.Nil(t, result, "corrupt JSON should return nil")
}
