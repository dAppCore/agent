// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_WriteStatus_Good(t *testing.T) {
	dir := t.TempDir()
	status := &WorkspaceStatus{
		Status:    "running",
		Agent:     "gemini",
		Repo:      "go-io",
		Task:      "fix tests",
		PID:       12345,
		StartedAt: time.Now(),
		Runs:      1,
	}

	err := writeStatus(dir, status)
	require.NoError(t, err)

	// Verify file was written via core.Fs
	r := fs.Read(core.JoinPath(dir, "status.json"))
	require.True(t, r.OK)

	var read WorkspaceStatus
	ur := core.JSONUnmarshalString(r.Value.(string), &read)
	require.True(t, ur.OK)

	assert.Equal(t, "running", read.Status)
	assert.Equal(t, "gemini", read.Agent)
	assert.Equal(t, "go-io", read.Repo)
	assert.Equal(t, "fix tests", read.Task)
	assert.Equal(t, 12345, read.PID)
	assert.Equal(t, 1, read.Runs)
	assert.False(t, read.UpdatedAt.IsZero(), "UpdatedAt should be set by writeStatus")
}

func TestStatus_WriteStatus_Good_UpdatesTimestamp(t *testing.T) {
	dir := t.TempDir()
	before := time.Now().Add(-time.Second)

	status := &WorkspaceStatus{
		Status: "running",
		Agent:  "claude",
	}

	err := writeStatus(dir, status)
	require.NoError(t, err)

	assert.True(t, status.UpdatedAt.After(before), "UpdatedAt should be after the start time")
}

func TestStatus_ReadStatus_Good(t *testing.T) {
	dir := t.TempDir()

	status := &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-log",
		Task:      "add logging",
		Branch:    "agent/add-logging",
		StartedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		Runs:      2,
		PRURL:     "https://forge.lthn.ai/core/go-log/pulls/5",
	}

	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	read, err := ReadStatus(dir)
	require.NoError(t, err)

	assert.Equal(t, "completed", read.Status)
	assert.Equal(t, "codex", read.Agent)
	assert.Equal(t, "go-log", read.Repo)
	assert.Equal(t, "add logging", read.Task)
	assert.Equal(t, "agent/add-logging", read.Branch)
	assert.Equal(t, 2, read.Runs)
	assert.Equal(t, "https://forge.lthn.ai/core/go-log/pulls/5", read.PRURL)
}

func TestStatus_ReadStatusResult_Good(t *testing.T) {
	dir := t.TempDir()

	status := &WorkspaceStatus{
		Status:    "completed",
		Agent:     "codex",
		Repo:      "go-log",
		Task:      "add logging",
		Branch:    "agent/add-logging",
		StartedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
		Runs:      2,
		PRURL:     "https://forge.lthn.ai/core/go-log/pulls/5",
	}

	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	result := ReadStatusResult(dir)
	require.True(t, result.OK)

	read, ok := result.Value.(*WorkspaceStatus)
	require.True(t, ok)
	assert.Equal(t, "completed", read.Status)
	assert.Equal(t, "codex", read.Agent)
	assert.Equal(t, "go-log", read.Repo)
	assert.Equal(t, "add logging", read.Task)
	assert.Equal(t, "agent/add-logging", read.Branch)
	assert.Equal(t, 2, read.Runs)
	assert.Equal(t, "https://forge.lthn.ai/core/go-log/pulls/5", read.PRURL)
}

func TestStatus_ReadStatusResult_Bad_NoFile(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	assert.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Error(t, err)
}

func TestStatus_ReadStatusResult_Ugly_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), "{not-json").OK)

	result := ReadStatusResult(dir)
	assert.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Error(t, err)
}

func TestStatus_ReadStatus_Bad_NoFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadStatus(dir)
	assert.Error(t, err)
}

func TestStatus_ReadStatus_Bad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), "not json{").OK)

	_, err := ReadStatus(dir)
	assert.Error(t, err)
}

func TestStatus_ReadStatus_Good_BlockedWithQuestion(t *testing.T) {
	dir := t.TempDir()

	status := &WorkspaceStatus{
		Status:   "blocked",
		Agent:    "gemini",
		Repo:     "go-io",
		Question: "Which interface should I implement?",
	}

	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	read, err := ReadStatus(dir)
	require.NoError(t, err)

	assert.Equal(t, "blocked", read.Status)
	assert.Equal(t, "Which interface should I implement?", read.Question)
}

func TestStatus_WriteRead_Good_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	original := &WorkspaceStatus{
		Status:    "running",
		Agent:     "claude:opus",
		Repo:      "agent",
		Org:       "core",
		Task:      "write tests for agentic package",
		Branch:    "agent/write-tests",
		Issue:     42,
		PID:       99999,
		StartedAt: time.Now().Truncate(time.Second),
		Runs:      3,
	}

	err := writeStatus(dir, original)
	require.NoError(t, err)

	read, err := ReadStatus(dir)
	require.NoError(t, err)

	assert.Equal(t, original.Status, read.Status)
	assert.Equal(t, original.Agent, read.Agent)
	assert.Equal(t, original.Repo, read.Repo)
	assert.Equal(t, original.Org, read.Org)
	assert.Equal(t, original.Task, read.Task)
	assert.Equal(t, original.Branch, read.Branch)
	assert.Equal(t, original.Issue, read.Issue)
	assert.Equal(t, original.PID, read.PID)
	assert.Equal(t, original.Runs, read.Runs)
}

func TestStatus_WriteStatus_Good_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()

	first := &WorkspaceStatus{Status: "running", Agent: "gemini"}
	err := writeStatus(dir, first)
	require.NoError(t, err)

	second := &WorkspaceStatus{Status: "completed", Agent: "gemini"}
	err = writeStatus(dir, second)
	require.NoError(t, err)

	read, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "completed", read.Status)
}

func TestStatus_ReadStatus_Ugly_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), "").OK)

	_, err := ReadStatus(dir)
	assert.Error(t, err)
}

// --- status() dead PID detection ---

func TestStatus_Status_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Case 1: running + dead PID + BLOCKED.md → should detect as blocked
	ws1 := core.JoinPath(wsRoot, "dead-blocked")
	require.True(t, fs.EnsureDir(core.JoinPath(ws1, "repo")).OK)
	require.NoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
		PID:    999999,
	}))
	require.True(t, fs.Write(core.JoinPath(ws1, "repo", "BLOCKED.md"), "Need API credentials").OK)

	// Case 2: running + dead PID + agent log → completed
	ws2 := core.JoinPath(wsRoot, "dead-completed")
	require.True(t, fs.EnsureDir(core.JoinPath(ws2, "repo")).OK)
	require.True(t, fs.EnsureDir(core.JoinPath(ws2, ".meta")).OK)
	require.NoError(t, writeStatus(ws2, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-log",
		Agent:  "claude",
		PID:    999999,
	}))
	require.True(t, fs.Write(core.JoinPath(ws2, ".meta", "agent-claude.log"), "agent finished ok").OK)

	// Case 3: running + dead PID + no log + no BLOCKED.md → failed
	ws3 := core.JoinPath(wsRoot, "dead-failed")
	require.True(t, fs.EnsureDir(core.JoinPath(ws3, "repo")).OK)
	require.NoError(t, writeStatus(ws3, &WorkspaceStatus{
		Status: "running",
		Repo:   "agent",
		Agent:  "gemini",
		PID:    999999,
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(nil, nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 3, out.Total)

	// Verify case 1: blocked
	assert.Len(t, out.Blocked, 1)
	assert.Equal(t, "go-io", out.Blocked[0].Repo)
	assert.Equal(t, "Need API credentials", out.Blocked[0].Question)

	// Verify case 2: completed
	assert.Equal(t, 1, out.Completed)

	// Verify case 3: failed
	assert.Equal(t, 1, out.Failed)

	// Verify statuses were persisted to disk
	st1, err := ReadStatus(ws1)
	require.NoError(t, err)
	assert.Equal(t, "blocked", st1.Status)

	st2, err := ReadStatus(ws2)
	require.NoError(t, err)
	assert.Equal(t, "completed", st2.Status)

	st3, err := ReadStatus(ws3)
	require.NoError(t, err)
	assert.Equal(t, "failed", st3.Status)
	assert.Equal(t, "Agent process died (no output log)", st3.Question)
}

// --- writeStatus (extended Ugly) ---

func TestStatus_WriteStatus_Ugly(t *testing.T) {
	// Write a status with all fields, read back, verify UpdatedAt is set and all fields preserved
	dir := t.TempDir()

	original := &WorkspaceStatus{
		Status:    "blocked",
		Agent:     "gemini:flash",
		Repo:      "go-mcp",
		Org:       "core",
		Task:      "Refactor IPC handler",
		Branch:    "agent/refactor-ipc",
		Issue:     77,
		PID:       999999, // dead PID — non-existent
		StartedAt: time.Now().Add(-10 * time.Minute).Truncate(time.Second),
		Question:  "Should I break backward compat?",
		Runs:      5,
		PRURL:     "https://forge.lthn.ai/core/go-mcp/pulls/12",
	}

	err := writeStatus(dir, original)
	require.NoError(t, err)

	// UpdatedAt should have been set by writeStatus
	assert.False(t, original.UpdatedAt.IsZero(), "writeStatus must set UpdatedAt")

	// Read back and verify every field
	read, err := ReadStatus(dir)
	require.NoError(t, err)

	assert.Equal(t, "blocked", read.Status)
	assert.Equal(t, "gemini:flash", read.Agent)
	assert.Equal(t, "go-mcp", read.Repo)
	assert.Equal(t, "core", read.Org)
	assert.Equal(t, "Refactor IPC handler", read.Task)
	assert.Equal(t, "agent/refactor-ipc", read.Branch)
	assert.Equal(t, 77, read.Issue)
	assert.Equal(t, 999999, read.PID)
	assert.Equal(t, "Should I break backward compat?", read.Question)
	assert.Equal(t, 5, read.Runs)
	assert.Equal(t, "https://forge.lthn.ai/core/go-mcp/pulls/12", read.PRURL)
	assert.False(t, read.UpdatedAt.IsZero(), "UpdatedAt must survive roundtrip")
}

// --- writeStatus Bad ---

func TestStatus_WriteStatus_Bad_ReadOnlyPath(t *testing.T) {
	// go-io fs.Write auto-creates dirs, so test with /dev/null parent
	st := &WorkspaceStatus{Status: "running", Agent: "codex"}
	err := writeStatus("/dev/null/impossible", st)
	assert.Error(t, err, "writeStatus to an impossible path should fail")
}

// --- status() MCP handler Good/Bad ---

func TestStatus_Status_Good_PopulatedWorkspaces(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create a running workspace with a live PID (our own PID)
	ws1 := core.JoinPath(wsRoot, "task-running")
	require.True(t, fs.EnsureDir(core.JoinPath(ws1, "repo")).OK)
	require.NoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
		Task:   "fix tests",
	}))

	// Create a blocked workspace
	ws2 := core.JoinPath(wsRoot, "task-blocked")
	require.True(t, fs.EnsureDir(core.JoinPath(ws2, "repo")).OK)
	require.NoError(t, writeStatus(ws2, &WorkspaceStatus{
		Status:   "blocked",
		Repo:     "go-log",
		Agent:    "gemini",
		Question: "Which log format?",
	}))

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 2, out.Total)
	assert.Equal(t, 1, out.Completed)
	assert.Len(t, out.Blocked, 1)
	assert.Equal(t, "go-log", out.Blocked[0].Repo)
	assert.Equal(t, "Which log format?", out.Blocked[0].Question)
}

func TestStatus_Status_Bad_EmptyWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	// Do NOT create the workspace/ subdirectory

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err, "status on missing workspace dir should not error")
	assert.Equal(t, 0, out.Total)
	assert.Equal(t, 0, out.Running)
	assert.Equal(t, 0, out.Completed)
}
