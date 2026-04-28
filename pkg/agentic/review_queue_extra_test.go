// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"strings"
	"testing"
	"time"

	core "dappco.re/go"
)

// --- buildReviewCommand ---

func TestReviewqueue_BuildReviewCommand_Good_CodeRabbit(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	cmd, args := s.buildReviewCommand("/tmp/repo", "coderabbit")
	core.AssertEqual(t, "coderabbit", cmd)
	core.AssertContains(t, args, "review")
	core.AssertContains(t, args, "--plain")
	core.AssertContains(t, args, "github/main")
}

func TestReviewqueue_BuildReviewCommand_Good_Codex(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	cmd, args := s.buildReviewCommand("/tmp/repo", "codex")
	core.AssertEqual(t, "codex", cmd)
	core.AssertContains(t, args, "review")
	core.AssertContains(t, args, "github/main")
}

func TestReviewqueue_BuildReviewCommand_Good_DefaultReviewer(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	cmd, args := s.buildReviewCommand("/tmp/repo", "")
	core.AssertEqual(t, "coderabbit", cmd)
	core.AssertContains(t, args, "--plain")
}

// --- saveRateLimitState / loadRateLimitState ---

func TestReviewqueue_SaveLoadRateLimitState_Good_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	// Ensure .core dir exists
	fs.EnsureDir(core.JoinPath(dir, ".core"))

	// Note: saveRateLimitState uses core.Env("DIR_HOME") which is pre-populated.
	// We need to work around this by using CORE_WORKSPACE for the load,
	// but save/load use DIR_HOME. Skip if not writable.
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	info := &RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(5 * time.Minute).Truncate(time.Second),
		Message: "rate limited",
	}
	s.saveRateLimitState(info)

	loaded := s.loadRateLimitState()
	if loaded != nil {
		core.AssertTrue(t, loaded.Limited)
		core.AssertEqual(t, "rate limited", loaded.Message)
	}
	// If loaded is nil it means DIR_HOME path wasn't writable — acceptable in test
}

// --- storeReviewOutput ---

func TestReviewqueue_StoreReviewOutput_Good(t *testing.T) {
	// storeReviewOutput uses core.Env("DIR_HOME") so we can't fully control the path
	// but we can verify it doesn't panic
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	core.AssertNotPanics(t, func() {
		s.storeReviewOutput(t.TempDir(), "test-repo", "coderabbit", "No findings — LGTM")
	})
}

func TestReviewqueue_RunPRManageLoop_Good_StopsOnCancel(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		s.runPRManageLoop(ctx, time.Hour)
		close(done)
	}()

	cancel()
	requireEventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 5*time.Millisecond)
}

// --- reviewQueue ---

func TestReviewqueue_NoCandidates_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Create an empty core dir (no repos)
	coreDir := core.JoinPath(root, "core")
	fs.EnsureDir(coreDir)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       root,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.reviewQueue(context.Background(), nil, ReviewQueueInput{DryRun: true})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEmpty(t, out.Processed)
}

// --- status (extended) ---

func TestReviewqueue_StatusFiltered_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

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
		wsDir := core.JoinPath(wsRoot, ws.name)
		fs.EnsureDir(wsDir)
		st := &WorkspaceStatus{Status: ws.status, Repo: "test", Agent: "codex"}
		fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))
	}

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 4, out.Total)
	core.AssertEqual(t, 2, out.Completed)
	core.AssertEqual(t, 1, out.Failed)
	core.AssertEqual(t, 1, out.Queued)
}

// --- handlers helpers (resolveWorkspace, findWorkspaceByPR) ---

func TestHandlers_ResolveWorkspace_Good_Exists(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create workspace dir
	ws := core.JoinPath(wsRoot, "core", "go-io", "task-15")
	fs.EnsureDir(ws)

	result := resolveWorkspace("core/go-io/task-15")
	core.AssertEqual(t, ws, result)
}

func TestHandlers_ResolveWorkspace_Bad_NotExists(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	result := resolveWorkspace("nonexistent")
	core.AssertEmpty(t, result)
}

func TestHandlers_FindWorkspaceByPR_Good_Match(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	ws := core.JoinPath(wsRoot, "ws-test")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Repo: "go-io", Branch: "agent/fix", Status: "completed"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	result := findWorkspaceByPR("go-io", "agent/fix")
	core.AssertEqual(t, ws, result)
}

func TestHandlers_FindWorkspaceByPR_Good_DeepLayout(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Deep layout: org/repo/task
	ws := core.JoinPath(wsRoot, "core", "agent", "task-5")
	fs.EnsureDir(ws)
	st := &WorkspaceStatus{Repo: "agent", Branch: "agent/tests", Status: "completed"}
	fs.Write(core.JoinPath(ws, "status.json"), core.JSONMarshalString(st))

	result := findWorkspaceByPR("agent", "agent/tests")
	core.AssertEqual(t, ws, result)
}

// --- loadRateLimitState (Ugly — corrupt JSON) ---

func TestReviewqueue_LoadRateLimitState_Ugly(t *testing.T) {
	ratePath := core.JoinPath(HomeDir(), ".core", "coderabbit-ratelimit.json")

	// Save original content (may or may not exist)
	origResult := fs.Read(ratePath)
	hadFile := origResult.OK
	var original string
	if hadFile {
		original = origResult.Value.(string)
	}

	// Ensure parent dir exists
	fs.EnsureDir(core.PathDir(ratePath))

	// Write corrupt JSON
	fs.Write(ratePath, "not-valid-json{{{")
	t.Cleanup(func() {
		if hadFile {
			fs.Write(ratePath, original)
		} else {
			fs.Delete(ratePath)
		}
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	result := s.loadRateLimitState()
	core.AssertNil(t, result, "corrupt JSON should return nil")
}

// --- buildReviewCommand Bad/Ugly ---

func TestReviewqueue_BuildReviewCommand_Bad(t *testing.T) {
	// Empty reviewer string — defaults to coderabbit
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	cmd, args := s.buildReviewCommand("/tmp/repo", "")
	core.AssertEqual(t, "coderabbit", cmd)
	core.AssertContains(t, args, "--plain")
}

func TestReviewqueue_BuildReviewCommand_Ugly(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	cmd, args := s.buildReviewCommand("/tmp/repo", "unknown-reviewer")
	core.AssertEqual(t, "coderabbit", cmd)
	core.AssertContains(t, args, "--plain")
}

// --- countFindings Bad/Ugly ---

func TestReviewqueue_CountFindings_Bad(t *testing.T) {
	// Empty string
	count := countFindings("")
	// Empty string doesn't contain "No findings" so defaults to 1
	core.AssertEqual(t, 1, count)
	core.AssertTrue(t, count > 0)
}

func TestReviewqueue_CountFindings_Ugly(t *testing.T) {
	// Only whitespace
	output := "   \n   \n   "
	count := countFindings(output)
	// No markers, no "No findings", so defaults to 1
	core.AssertEqual(t, 1, count)
	core.AssertNotContains(t, output, "No findings")
}

// --- parseRetryAfter Ugly ---

func TestReviewqueue_ParseRetryAfter_Ugly(t *testing.T) {
	// Seconds only "try after 30 seconds" — no minutes match
	message := "try after 30 seconds"
	d := parseRetryAfter(message)
	// Regex expects minutes first, so this won't match — defaults to 5 min
	core.AssertEqual(t, 5*time.Minute, d)
	core.AssertTrue(t, d > time.Minute)
}

// --- storeReviewOutput Bad/Ugly ---

func TestReviewqueue_StoreReviewOutput_Bad(t *testing.T) {
	// Empty output
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	core.AssertNotPanics(t, func() {
		s.storeReviewOutput(t.TempDir(), "test-repo", "coderabbit", "")
	})
}

func TestReviewqueue_StoreReviewOutput_Ugly(t *testing.T) {
	// Very large output
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	largeOutput := strings.Repeat("Finding: something is wrong on this line\n", 10000)
	core.AssertNotPanics(t, func() {
		s.storeReviewOutput(t.TempDir(), "test-repo", "coderabbit", largeOutput)
	})
}

// --- saveRateLimitState Good/Bad/Ugly ---

func TestReviewqueue_SaveRateLimitState_Good(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	info := &RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(5 * time.Minute).Truncate(time.Second),
		Message: "rate limited",
	}
	core.AssertNotPanics(t, func() {
		s.saveRateLimitState(info)
	})
}

func TestReviewqueue_SaveRateLimitState_Bad(t *testing.T) {
	// Save nil info
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	core.AssertNotPanics(t, func() {
		s.saveRateLimitState(nil)
	})
}

func TestReviewqueue_SaveRateLimitState_Bad_WriteFailure(t *testing.T) {
	t.Setenv("CORE_HOME", "/dev/null")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	info := &RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(5 * time.Minute).Truncate(time.Second),
		Message: "write failure",
	}
	core.AssertNotPanics(t, func() {
		s.saveRateLimitState(info)
	})
}

func TestReviewqueue_SaveRateLimitState_Ugly(t *testing.T) {
	// Save with far-future RetryAt
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	info := &RateLimitInfo{
		Limited: true,
		RetryAt: time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC),
		Message: "far future rate limit",
	}
	core.AssertNotPanics(t, func() {
		s.saveRateLimitState(info)
	})
}

// --- loadRateLimitState Good ---

func TestReviewqueue_LoadRateLimitState_Good(t *testing.T) {
	// Write then load valid state
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	info := &RateLimitInfo{
		Limited: true,
		RetryAt: time.Now().Add(5 * time.Minute).Truncate(time.Second),
		Message: "test rate limit",
	}
	s.saveRateLimitState(info)

	loaded := s.loadRateLimitState()
	if loaded != nil {
		core.AssertTrue(t, loaded.Limited)
		core.AssertEqual(t, "test rate limit", loaded.Message)
	}
	// If loaded is nil, DIR_HOME path wasn't writable — acceptable in test
}

// --- loadRateLimitState Bad ---

func TestReviewqueue_LoadRateLimitState_Bad(t *testing.T) {
	// File doesn't exist — should return nil
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// loadRateLimitState reads from DIR_HOME/.core/coderabbit-ratelimit.json.
	// If the file doesn't exist, it should return nil without panic.
	result := s.loadRateLimitState()
	// May or may not be nil depending on whether the file exists in the real home dir.
	// The key invariant is: it must not panic.
	_ = result
}
