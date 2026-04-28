// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

// --- ReadStatusResult ---

func TestStatus_ReadStatus_Good_AllFields(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().Truncate(time.Second)

	original := WorkspaceStatus{
		Status:    "running",
		Agent:     "claude:opus",
		Repo:      "go-io",
		Org:       "core",
		Task:      "add observability",
		Branch:    "agent/add-observability",
		Issue:     7,
		PID:       42100,
		StartedAt: now,
		UpdatedAt: now,
		Question:  "",
		Runs:      2,
		PRURL:     "",
	}
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(original)).OK)

	st := mustReadStatus(t, dir)

	core.AssertEqual(t, original.Status, st.Status)
	core.AssertEqual(t, original.Agent, st.Agent)
	core.AssertEqual(t, original.Repo, st.Repo)
	core.AssertEqual(t, original.Org, st.Org)
	core.AssertEqual(t, original.Task, st.Task)
	core.AssertEqual(t, original.Branch, st.Branch)
	core.AssertEqual(t, original.Issue, st.Issue)
	core.AssertEqual(t, original.PID, st.PID)
	core.AssertEqual(t, original.Runs, st.Runs)
}

func TestStatus_ReadStatus_Bad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	result := ReadStatusResult(dir)
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.RequireTrue(t, ok)
}

func TestStatus_ReadStatus_Bad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), `{"status": "running", broken`).OK)

	result := ReadStatusResult(dir)
	core.AssertFalse(t, result.OK)
	_, ok := result.Value.(error)
	core.RequireTrue(t, ok)
}

func TestStatus_ReadStatus_Bad_NullJSON(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "status.json"), "null").OK)

	// null is valid JSON — ReadStatusResult returns a zero-value struct, not an error
	st := mustReadStatus(t, dir)
	core.AssertEqual(t, "", st.Status)
}

// --- writeStatus ---

func TestStatus_WriteStatus_Good_WritesAndReadsBack(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{
		Status: "queued",
		Agent:  "gemini:pro",
		Repo:   "go-log",
		Task:   "improve logging",
		Runs:   0,
	}

	err := writeStatus(dir, st)
	core.RequireNoError(t, err)

	read := mustReadStatus(t, dir)
	core.AssertEqual(t, "queued", read.Status)
	core.AssertEqual(t, "gemini:pro", read.Agent)
	core.AssertEqual(t, "go-log", read.Repo)
	core.AssertEqual(t, "improve logging", read.Task)
}

func TestStatus_WriteStatus_Good_SetsUpdatedAt(t *testing.T) {
	dir := t.TempDir()
	before := time.Now().Add(-time.Millisecond)

	st := &WorkspaceStatus{Status: "failed", Agent: "codex"}
	err := writeStatus(dir, st)
	core.RequireNoError(t, err)

	core.AssertTrue(t, st.UpdatedAt.After(before), "writeStatus must set UpdatedAt to a recent time")
}

func TestStatus_WriteStatus_Good_Overwrites(t *testing.T) {
	dir := t.TempDir()

	core.RequireNoError(t, writeStatus(dir, &WorkspaceStatus{Status: "running", Agent: "gemini"}))
	core.RequireNoError(t, writeStatus(dir, &WorkspaceStatus{Status: "completed", Agent: "gemini"}))

	st := mustReadStatus(t, dir)
	core.AssertEqual(t, "completed", st.Status)
}

// --- WorkspaceStatus JSON round-trip ---

func TestStatus_WorkspaceStatus_Good_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := WorkspaceStatus{
		Status:    "blocked",
		Agent:     "codex:gpt-5.4",
		Repo:      "agent",
		Org:       "core",
		Task:      "write more tests",
		Branch:    "agent/write-more-tests",
		Issue:     15,
		PID:       99001,
		StartedAt: now,
		UpdatedAt: now,
		Question:  "Which pattern should I use?",
		Runs:      3,
		PRURL:     "https://forge.lthn.ai/core/agent/pulls/10",
	}

	jsonStr := core.JSONMarshalString(original)

	var decoded WorkspaceStatus
	core.RequireTrue(t, core.JSONUnmarshalString(jsonStr, &decoded).OK)

	core.AssertEqual(t, original.Status, decoded.Status)
	core.AssertEqual(t, original.Agent, decoded.Agent)
	core.AssertEqual(t, original.Repo, decoded.Repo)
	core.AssertEqual(t, original.Org, decoded.Org)
	core.AssertEqual(t, original.Task, decoded.Task)
	core.AssertEqual(t, original.Branch, decoded.Branch)
	core.AssertEqual(t, original.Issue, decoded.Issue)
	core.AssertEqual(t, original.PID, decoded.PID)
	core.AssertEqual(t, original.Question, decoded.Question)
	core.AssertEqual(t, original.Runs, decoded.Runs)
	core.AssertEqual(t, original.PRURL, decoded.PRURL)
}

func TestStatus_WorkspaceStatus_Good_OmitemptyFields(t *testing.T) {
	st := WorkspaceStatus{Status: "queued", Agent: "claude"}

	// Optional fields with omitempty must be absent when zero
	jsonStr := core.JSONMarshalString(st)
	core.AssertNotContains(t, jsonStr, `"org"`)
	core.AssertNotContains(t, jsonStr, `"branch"`)
	core.AssertNotContains(t, jsonStr, `"question"`)
	core.AssertNotContains(t, jsonStr, `"pr_url"`)
	core.AssertNotContains(t, jsonStr, `"pid"`)
	core.AssertNotContains(t, jsonStr, `"issue"`)
}
