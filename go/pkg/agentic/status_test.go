// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
)

func mustReadStatus(t *testing.T, dir string) *WorkspaceStatus {
	t.Helper()

	result := ReadStatusResult(dir)
	core.RequireTrue(t, result.OK)

	status, ok := workspaceStatusValue(result)
	core.RequireTrue(t, ok)
	return status
}

func TestStatus_WriteStatus_Good_Case(t *testing.T) {
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
	core.RequireNoError(t, err)

	// Verify file was written via core.Fs
	r := fs.Read(core.JoinPath(dir, "status.json"))
	core.RequireTrue(t, r.OK)

	var read WorkspaceStatus
	ur := core.JSONUnmarshalString(r.Value.(string), &read)
	core.RequireTrue(t, ur.OK)

	core.AssertEqual(t, "running", read.Status)
	core.AssertEqual(t, "gemini", read.Agent)
	core.AssertEqual(t, "go-io", read.Repo)
	core.AssertEqual(t, "fix tests", read.Task)
	core.AssertEqual(t, 12345, read.PID)
	core.AssertEqual(t, 1, read.Runs)
	core.AssertFalse(t, read.UpdatedAt.IsZero(), "UpdatedAt should be set by writeStatus")
}

func TestStatus_WriteStatus_Good_UpdatesTimestamp(t *testing.T) {
	dir := t.TempDir()
	before := time.Now().Add(-time.Second)

	status := &WorkspaceStatus{
		Status: "running",
		Agent:  "claude",
	}

	err := writeStatus(dir, status)
	core.RequireNoError(t, err)

	core.AssertTrue(t, status.UpdatedAt.After(before), "UpdatedAt should be after the start time")
}

func TestStatus_ReadStatus_Good_Case(t *testing.T) {
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

	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	read := mustReadStatus(t, dir)

	core.AssertEqual(t, "completed", read.Status)
	core.AssertEqual(t, "codex", read.Agent)
	core.AssertEqual(t, "go-log", read.Repo)
	core.AssertEqual(t, "add logging", read.Task)
	core.AssertEqual(t, "agent/add-logging", read.Branch)
	core.AssertEqual(t, 2, read.Runs)
	core.AssertEqual(t, "https://forge.lthn.ai/core/go-log/pulls/5", read.PRURL)
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

	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	result := ReadStatusResult(dir)
	core.RequireTrue(t, result.OK)

	read, ok := result.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "completed", read.Status)
	core.AssertEqual(t, "codex", read.Agent)
	core.AssertEqual(t, "go-log", read.Repo)
	core.AssertEqual(t, "add logging", read.Task)
	core.AssertEqual(t, "agent/add-logging", read.Branch)
	core.AssertEqual(t, 2, read.Runs)
	core.AssertEqual(t, "https://forge.lthn.ai/core/go-log/pulls/5", read.PRURL)
}

func TestStatus_ReadStatusResult_Bad(t *testing.T) {
	result := ReadStatusResult(t.TempDir())
	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
}

func TestStatus_ReadStatusResult_Ugly(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), "{not-json").OK)

	result := ReadStatusResult(dir)
	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertError(t, err)
}

func TestStatus_ReadStatus_Bad_NoFile(t *testing.T) {
	dir := t.TempDir()
	result := ReadStatusResult(dir)
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.AssertTrue(t, ok)
}

func TestStatus_ReadStatus_Bad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), "not json{").OK)

	result := ReadStatusResult(dir)
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.AssertTrue(t, ok)
}

func TestStatus_ReadStatus_Good_BlockedWithQuestion(t *testing.T) {
	dir := t.TempDir()

	status := &WorkspaceStatus{
		Status:   "blocked",
		Agent:    "gemini",
		Repo:     "go-io",
		Question: "Which interface should I implement?",
	}

	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	read := mustReadStatus(t, dir)

	core.AssertEqual(t, "blocked", read.Status)
	core.AssertEqual(t, "Which interface should I implement?", read.Question)
}

func TestStatus_ReadStatus_Good(t *testing.T) {
	dir := t.TempDir()

	status := &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		Repo:   "go-io",
		Task:   "add logging",
	}

	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(status)).OK)

	read, err := ReadStatus(dir)
	core.RequireNoError(t, err)
	core.AssertNotNil(t, read)
	core.AssertEqual(t, "completed", read.Status)
	core.AssertEqual(t, "go-io", read.Repo)
}

func TestStatus_ReadStatus_Bad_NoFile_Wrapper(t *testing.T) {
	read, err := ReadStatus(t.TempDir())
	core.AssertNil(t, read)
	core.AssertError(t, err)
}

func TestStatus_ReadStatus_Bad(t *testing.T) {
	read, err := ReadStatus(t.TempDir())
	core.AssertNil(t, read)
	core.AssertError(t, err)
}

func TestStatus_ReadStatus_Ugly(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), "not json{").OK)

	read, err := ReadStatus(dir)
	core.AssertNil(t, read)
	core.AssertError(t, err)
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
	core.RequireNoError(t, err)

	read := mustReadStatus(t, dir)

	core.AssertEqual(t, original.Status, read.Status)
	core.AssertEqual(t, original.Agent, read.Agent)
	core.AssertEqual(t, original.Repo, read.Repo)
	core.AssertEqual(t, original.Org, read.Org)
	core.AssertEqual(t, original.Task, read.Task)
	core.AssertEqual(t, original.Branch, read.Branch)
	core.AssertEqual(t, original.Issue, read.Issue)
	core.AssertEqual(t, original.PID, read.PID)
	core.AssertEqual(t, original.Runs, read.Runs)
}

func TestStatus_WriteStatus_Good_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()

	first := &WorkspaceStatus{Status: "running", Agent: "gemini"}
	err := writeStatus(dir, first)
	core.RequireNoError(t, err)

	second := &WorkspaceStatus{Status: "completed", Agent: "gemini"}
	err = writeStatus(dir, second)
	core.RequireNoError(t, err)

	read := mustReadStatus(t, dir)
	core.AssertEqual(t, "completed", read.Status)
}

func TestStatus_ReadStatus_Ugly_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), "").OK)

	result := ReadStatusResult(dir)
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.AssertTrue(t, ok)
}

// --- status() dead PID detection ---

func TestStatus_Status_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Case 1: running + dead PID + BLOCKED.md → should detect as blocked
	ws1 := core.JoinPath(wsRoot, "dead-blocked")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(ws1, "repo")).OK)
	core.RequireNoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-io",
		Agent:  "codex",
		PID:    999999,
	}))
	core.RequireTrue(t, fs.Write(core.JoinPath(ws1, "repo", "BLOCKED.md"), "Need API credentials").OK)

	// Case 2: running + dead PID + agent log → completed
	ws2 := core.JoinPath(wsRoot, "dead-completed")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(ws2, "repo")).OK)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(ws2, ".meta")).OK)
	core.RequireNoError(t, writeStatus(ws2, &WorkspaceStatus{
		Status: "running",
		Repo:   "go-log",
		Agent:  "claude",
		PID:    999999,
	}))
	core.RequireTrue(t, fs.Write(core.JoinPath(ws2, ".meta", "agent-claude.log"), "agent finished ok").OK)

	// Case 3: running + dead PID + no log + no BLOCKED.md → failed
	ws3 := core.JoinPath(wsRoot, "dead-failed")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(ws3, "repo")).OK)
	core.RequireNoError(t, writeStatus(ws3, &WorkspaceStatus{
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

	_, out, err := s.status(context.TODO(), nil, StatusInput{})
	core.RequireNoError(t, err)
	core.AssertEqual(t, 3, out.Total)

	// Verify case 1: blocked
	core.AssertLen(t, out.Blocked, 1)
	core.AssertEqual(t, "go-io", out.Blocked[0].Repo)
	core.AssertEqual(t, "Need API credentials", out.Blocked[0].Question)

	// Verify case 2: completed
	core.AssertEqual(t, 1, out.Completed)

	// Verify case 3: failed
	core.AssertEqual(t, 1, out.Failed)

	// Verify statuses were persisted to disk
	st1 := mustReadStatus(t, ws1)
	core.AssertEqual(t, "blocked", st1.Status)

	st2 := mustReadStatus(t, ws2)
	core.AssertEqual(t, "completed", st2.Status)

	st3 := mustReadStatus(t, ws3)
	core.AssertEqual(t, "failed", st3.Status)
	core.AssertEqual(t, "Agent process died (no output log)", st3.Question)
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
	core.RequireNoError(t, err)

	// UpdatedAt should have been set by writeStatus
	core.AssertFalse(t, original.UpdatedAt.IsZero(), "writeStatus must set UpdatedAt")

	// Read back and verify every field
	read := mustReadStatus(t, dir)

	core.AssertEqual(t, "blocked", read.Status)
	core.AssertEqual(t, "gemini:flash", read.Agent)
	core.AssertEqual(t, "go-mcp", read.Repo)
	core.AssertEqual(t, "core", read.Org)
	core.AssertEqual(t, "Refactor IPC handler", read.Task)
	core.AssertEqual(t, "agent/refactor-ipc", read.Branch)
	core.AssertEqual(t, 77, read.Issue)
	core.AssertEqual(t, 999999, read.PID)
	core.AssertEqual(t, "Should I break backward compat?", read.Question)
	core.AssertEqual(t, 5, read.Runs)
	core.AssertEqual(t, "https://forge.lthn.ai/core/go-mcp/pulls/12", read.PRURL)
	core.AssertFalse(t, read.UpdatedAt.IsZero(), "UpdatedAt must survive roundtrip")
}

// --- writeStatus Bad ---

func TestStatus_WriteStatus_Bad_ReadOnlyPath(t *testing.T) {
	// go-io fs.Write auto-creates dirs, so test with /dev/null parent
	st := &WorkspaceStatus{Status: "running", Agent: "codex"}
	err := writeStatus("/dev/null/impossible", st)
	core.AssertError(t, err)
}

// --- status() MCP handler Good/Bad ---

func TestStatus_Status_Good_PopulatedWorkspaces(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsRoot := core.JoinPath(root, "workspace")

	// Create a running workspace with a live PID (our own PID)
	ws1 := core.JoinPath(wsRoot, "task-running")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(ws1, "repo")).OK)
	core.RequireNoError(t, writeStatus(ws1, &WorkspaceStatus{
		Status: "completed",
		Repo:   "go-io",
		Agent:  "codex",
		Task:   "fix tests",
	}))

	// Create a blocked workspace
	ws2 := core.JoinPath(wsRoot, "task-blocked")
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(ws2, "repo")).OK)
	core.RequireNoError(t, writeStatus(ws2, &WorkspaceStatus{
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
	core.RequireNoError(t, err)
	core.AssertEqual(t, 2, out.Total)
	core.AssertEqual(t, 1, out.Completed)
	core.AssertLen(t, out.Blocked, 1)
	core.AssertEqual(t, "go-log", out.Blocked[0].Repo)
	core.AssertEqual(t, "Which log format?", out.Blocked[0].Question)
}

func TestStatus_Status_Bad_EmptyWorkspaceRoot(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	// Do NOT create the workspace/ subdirectory

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.status(context.Background(), nil, StatusInput{})
	core.RequireNoError(t, err, "status on missing workspace dir should not error")
	core.AssertEqual(t, 0, out.Total)
	core.AssertEqual(t, 0, out.Running)
	core.AssertEqual(t, 0, out.Completed)
}
