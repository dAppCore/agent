// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ReadStatus ---

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
	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(original)).OK)

	st, err := ReadStatus(dir)
	require.NoError(t, err)

	assert.Equal(t, original.Status, st.Status)
	assert.Equal(t, original.Agent, st.Agent)
	assert.Equal(t, original.Repo, st.Repo)
	assert.Equal(t, original.Org, st.Org)
	assert.Equal(t, original.Task, st.Task)
	assert.Equal(t, original.Branch, st.Branch)
	assert.Equal(t, original.Issue, st.Issue)
	assert.Equal(t, original.PID, st.PID)
	assert.Equal(t, original.Runs, st.Runs)
}

func TestStatus_ReadStatus_Bad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadStatus(dir)
	assert.Error(t, err, "missing status.json must return an error")
}

func TestStatus_ReadStatus_Bad_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), `{"status": "running", broken`).OK)

	_, err := ReadStatus(dir)
	assert.Error(t, err, "corrupt JSON must return an error")
}

func TestStatus_ReadStatus_Bad_NullJSON(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "status.json"), "null").OK)

	// null is valid JSON — ReadStatus returns a zero-value struct, not an error
	st, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "", st.Status)
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
	require.NoError(t, err)

	read, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "queued", read.Status)
	assert.Equal(t, "gemini:pro", read.Agent)
	assert.Equal(t, "go-log", read.Repo)
	assert.Equal(t, "improve logging", read.Task)
}

func TestStatus_WriteStatus_Good_SetsUpdatedAt(t *testing.T) {
	dir := t.TempDir()
	before := time.Now().Add(-time.Millisecond)

	st := &WorkspaceStatus{Status: "failed", Agent: "codex"}
	err := writeStatus(dir, st)
	require.NoError(t, err)

	assert.True(t, st.UpdatedAt.After(before), "writeStatus must set UpdatedAt to a recent time")
}

func TestStatus_WriteStatus_Good_Overwrites(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, writeStatus(dir, &WorkspaceStatus{Status: "running", Agent: "gemini"}))
	require.NoError(t, writeStatus(dir, &WorkspaceStatus{Status: "completed", Agent: "gemini"}))

	st, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "completed", st.Status)
}

// --- WorkspaceStatus JSON round-trip ---

func TestWorkspaceStatus_Good_JSONRoundTrip(t *testing.T) {
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
	require.True(t, core.JSONUnmarshalString(jsonStr, &decoded).OK)

	assert.Equal(t, original.Status, decoded.Status)
	assert.Equal(t, original.Agent, decoded.Agent)
	assert.Equal(t, original.Repo, decoded.Repo)
	assert.Equal(t, original.Org, decoded.Org)
	assert.Equal(t, original.Task, decoded.Task)
	assert.Equal(t, original.Branch, decoded.Branch)
	assert.Equal(t, original.Issue, decoded.Issue)
	assert.Equal(t, original.PID, decoded.PID)
	assert.Equal(t, original.Question, decoded.Question)
	assert.Equal(t, original.Runs, decoded.Runs)
	assert.Equal(t, original.PRURL, decoded.PRURL)
}

func TestWorkspaceStatus_Good_OmitemptyFields(t *testing.T) {
	st := WorkspaceStatus{Status: "queued", Agent: "claude"}

	// Optional fields with omitempty must be absent when zero
	jsonStr := core.JSONMarshalString(st)
	assert.NotContains(t, jsonStr, `"org"`)
	assert.NotContains(t, jsonStr, `"branch"`)
	assert.NotContains(t, jsonStr, `"question"`)
	assert.NotContains(t, jsonStr, `"pr_url"`)
	assert.NotContains(t, jsonStr, `"pid"`)
	assert.NotContains(t, jsonStr, `"issue"`)
}
