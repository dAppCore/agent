// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- agentOutputFile ---

func TestAgentOutputFile_Good(t *testing.T) {
	assert.Contains(t, agentOutputFile("/ws", "codex"), ".meta/agent-codex.log")
	assert.Contains(t, agentOutputFile("/ws", "claude:opus"), ".meta/agent-claude.log")
	assert.Contains(t, agentOutputFile("/ws", "gemini:flash"), ".meta/agent-gemini.log")
}

// --- detectFinalStatus ---

func TestDetectFinalStatus_Good_Completed(t *testing.T) {
	dir := t.TempDir()
	status, question := detectFinalStatus(dir, 0, "completed")
	assert.Equal(t, "completed", status)
	assert.Empty(t, question)
}

func TestDetectFinalStatus_Good_Blocked(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "BLOCKED.md"), []byte("Need API key for external service"), 0o644)

	status, question := detectFinalStatus(dir, 0, "completed")
	assert.Equal(t, "blocked", status)
	assert.Equal(t, "Need API key for external service", question)
}

func TestDetectFinalStatus_Good_BlockedEmpty(t *testing.T) {
	dir := t.TempDir()
	// BLOCKED.md exists but is empty — should NOT be treated as blocked
	os.WriteFile(filepath.Join(dir, "BLOCKED.md"), []byte("  \n  "), 0o644)

	status, _ := detectFinalStatus(dir, 0, "completed")
	assert.Equal(t, "completed", status)
}

func TestDetectFinalStatus_Good_FailedExitCode(t *testing.T) {
	dir := t.TempDir()
	status, question := detectFinalStatus(dir, 1, "completed")
	assert.Equal(t, "failed", status)
	assert.Contains(t, question, "code 1")
}

func TestDetectFinalStatus_Good_FailedKilled(t *testing.T) {
	dir := t.TempDir()
	status, _ := detectFinalStatus(dir, 0, "killed")
	assert.Equal(t, "failed", status)
}

func TestDetectFinalStatus_Good_FailedStatus(t *testing.T) {
	dir := t.TempDir()
	status, _ := detectFinalStatus(dir, 0, "failed")
	assert.Equal(t, "failed", status)
}

func TestDetectFinalStatus_Good_BlockedTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	// Agent wrote BLOCKED.md AND exited non-zero — blocked takes precedence
	os.WriteFile(filepath.Join(dir, "BLOCKED.md"), []byte("Need help"), 0o644)

	status, question := detectFinalStatus(dir, 1, "failed")
	assert.Equal(t, "blocked", status)
	assert.Equal(t, "Need help", question)
}

// --- trackFailureRate ---

func TestTrackFailureRate_Good_SuccessResetsCount(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: map[string]int{"codex": 2},
	}
	triggered := s.trackFailureRate("codex", "completed", time.Now().Add(-10*time.Second))
	assert.False(t, triggered)
	assert.Equal(t, 0, s.failCount["codex"])
}

func TestTrackFailureRate_Good_SlowFailureResetsCount(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: map[string]int{"codex": 2},
	}
	// Started 5 minutes ago = slow failure
	triggered := s.trackFailureRate("codex", "failed", time.Now().Add(-5*time.Minute))
	assert.False(t, triggered)
	assert.Equal(t, 0, s.failCount["codex"])
}

func TestTrackFailureRate_Good_FastFailureIncrementsCount(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Started 10 seconds ago = fast failure
	triggered := s.trackFailureRate("codex", "failed", time.Now().Add(-10*time.Second))
	assert.False(t, triggered)
	assert.Equal(t, 1, s.failCount["codex"])
}

func TestTrackFailureRate_Good_ThreeFailsTriggersBackoff(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: map[string]int{"codex": 2}, // already 2 fast failures
	}
	triggered := s.trackFailureRate("codex", "failed", time.Now().Add(-10*time.Second))
	assert.True(t, triggered)
	assert.True(t, time.Now().Before(s.backoff["codex"]))
}

func TestTrackFailureRate_Good_ModelVariantUsesPool(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	s.trackFailureRate("codex:gpt-5.4", "failed", time.Now().Add(-10*time.Second))
	assert.Equal(t, 1, s.failCount["codex"], "should track by base agent pool")
}

// --- startIssueTracking / stopIssueTracking ---

func TestStartIssueTracking_Good_NoForge(t *testing.T) {
	s := &PrepSubsystem{
		forge:     nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Should not panic
	s.startIssueTracking(t.TempDir())
}

func TestStopIssueTracking_Good_NoForge(t *testing.T) {
	s := &PrepSubsystem{
		forge:     nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	s.stopIssueTracking(t.TempDir())
}

func TestStartIssueTracking_Good_NoIssue(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "running", Repo: "test"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(dir, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		forge:     nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	s.startIssueTracking(dir) // no panic, no issue to track
}

// --- broadcastStart / broadcastComplete ---

func TestBroadcastStart_Good_NoCore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := &PrepSubsystem{
		core:      nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	// Should not panic even without Core
	s.broadcastStart("codex", dir)
}

func TestBroadcastComplete_Good_NoCore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := &PrepSubsystem{
		core:      nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	s.broadcastComplete("codex", dir, "completed")
}

// --- onAgentComplete ---

func TestOnAgentComplete_Good_Completed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := filepath.Join(root, "ws-test")
	repoDir := filepath.Join(wsDir, "repo")
	metaDir := filepath.Join(wsDir, ".meta")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(metaDir, 0o755)

	// Write initial status
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(wsDir, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		core:      nil,
		forge:     nil,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	outputFile := filepath.Join(metaDir, "agent-codex.log")
	s.onAgentComplete("codex", wsDir, outputFile, 0, "completed", "test output")

	// Verify status was updated
	updated, err := ReadStatus(wsDir)
	assert.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, 0, updated.PID)
	assert.Empty(t, updated.Question)

	// Verify output was written
	content, _ := os.ReadFile(outputFile)
	assert.Equal(t, "test output", string(content))
}

func TestOnAgentComplete_Good_Failed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := filepath.Join(root, "ws-fail")
	repoDir := filepath.Join(wsDir, "repo")
	metaDir := filepath.Join(wsDir, ".meta")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(metaDir, 0o755)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(wsDir, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	s.onAgentComplete("codex", wsDir, filepath.Join(metaDir, "agent-codex.log"), 1, "failed", "error output")

	updated, _ := ReadStatus(wsDir)
	assert.Equal(t, "failed", updated.Status)
	assert.Contains(t, updated.Question, "code 1")
}

func TestOnAgentComplete_Good_Blocked(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := filepath.Join(root, "ws-blocked")
	repoDir := filepath.Join(wsDir, "repo")
	metaDir := filepath.Join(wsDir, ".meta")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(metaDir, 0o755)

	// Create BLOCKED.md
	os.WriteFile(filepath.Join(repoDir, "BLOCKED.md"), []byte("Need credentials"), 0o644)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(wsDir, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	s.onAgentComplete("codex", wsDir, filepath.Join(metaDir, "agent-codex.log"), 0, "completed", "")

	updated, _ := ReadStatus(wsDir)
	assert.Equal(t, "blocked", updated.Status)
	assert.Equal(t, "Need credentials", updated.Question)
}

func TestOnAgentComplete_Good_EmptyOutput(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := filepath.Join(root, "ws-empty")
	repoDir := filepath.Join(wsDir, "repo")
	metaDir := filepath.Join(wsDir, ".meta")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(metaDir, 0o755)

	st := &WorkspaceStatus{Status: "running", Repo: "test", Agent: "codex", StartedAt: time.Now()}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(wsDir, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	outputFile := filepath.Join(metaDir, "agent-codex.log")
	s.onAgentComplete("codex", wsDir, outputFile, 0, "completed", "")

	// Output file should NOT be created for empty output
	_, err := os.Stat(outputFile)
	assert.True(t, os.IsNotExist(err))
}
