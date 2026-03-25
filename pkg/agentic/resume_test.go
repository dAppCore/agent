// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- resume ---

func TestResume_Bad_EmptyWorkspace(t *testing.T) {
	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, _, err := s.resume(context.Background(), nil, ResumeInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace is required")
}

func TestResume_Bad_WorkspaceNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DIR_HOME", dir)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, _, err := s.resume(context.Background(), nil, ResumeInput{Workspace: "nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace not found")
}

func TestResume_Bad_NotResumableStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DIR_HOME", dir)

	wsRoot := WorkspaceRoot()
	ws := filepath.Join(wsRoot, "ws-running")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)

	// Init git repo
	exec.Command("git", "init", repoDir).Run()

	st := &WorkspaceStatus{Status: "running", Repo: "test", Agent: "codex"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, _, err := s.resume(context.Background(), nil, ResumeInput{Workspace: "ws-running"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not resumable")
}

func TestResume_Good_DryRun(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DIR_HOME", dir)

	wsRoot := WorkspaceRoot()
	ws := filepath.Join(wsRoot, "ws-blocked")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)

	// Init git repo
	exec.Command("git", "init", repoDir).Run()

	st := &WorkspaceStatus{
		Status: "blocked",
		Repo:   "go-io",
		Agent:  "codex",
		Task:   "Fix the tests",
	}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, out, err := s.resume(context.Background(), nil, ResumeInput{
		Workspace: "ws-blocked",
		Answer:    "Use the new Core API",
		DryRun:    true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "ws-blocked", out.Workspace)
	assert.Equal(t, "codex", out.Agent)
	assert.Contains(t, out.Prompt, "Fix the tests")
	assert.Contains(t, out.Prompt, "Use the new Core API")

	// Verify ANSWER.md was written
	answerContent, readErr := os.ReadFile(filepath.Join(repoDir, "ANSWER.md"))
	require.NoError(t, readErr)
	assert.Contains(t, string(answerContent), "Use the new Core API")
}

func TestResume_Good_AgentOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DIR_HOME", dir)

	wsRoot := WorkspaceRoot()
	ws := filepath.Join(wsRoot, "ws-failed")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)
	exec.Command("git", "init", repoDir).Run()

	st := &WorkspaceStatus{
		Status: "failed",
		Repo:   "go-crypt",
		Agent:  "codex",
		Task:   "Review code",
	}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	_, out, err := s.resume(context.Background(), nil, ResumeInput{
		Workspace: "ws-failed",
		Agent:     "claude:opus",
		DryRun:    true,
	})
	require.NoError(t, err)
	assert.Equal(t, "claude:opus", out.Agent, "should override agent")
}
