// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteStatus_Good(t *testing.T) {
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
	r := fs.Read(filepath.Join(dir, "status.json"))
	require.True(t, r.OK)

	var read WorkspaceStatus
	err = json.Unmarshal([]byte(r.Value.(string)), &read)
	require.NoError(t, err)

	assert.Equal(t, "running", read.Status)
	assert.Equal(t, "gemini", read.Agent)
	assert.Equal(t, "go-io", read.Repo)
	assert.Equal(t, "fix tests", read.Task)
	assert.Equal(t, 12345, read.PID)
	assert.Equal(t, 1, read.Runs)
	assert.False(t, read.UpdatedAt.IsZero(), "UpdatedAt should be set by writeStatus")
}

func TestWriteStatus_Good_UpdatesTimestamp(t *testing.T) {
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

func TestReadStatus_Good(t *testing.T) {
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

	data, err := json.MarshalIndent(status, "", "  ")
	require.NoError(t, err)
	require.True(t, fs.Write(filepath.Join(dir, "status.json"), string(data)).OK)

	read, err := readStatus(dir)
	require.NoError(t, err)

	assert.Equal(t, "completed", read.Status)
	assert.Equal(t, "codex", read.Agent)
	assert.Equal(t, "go-log", read.Repo)
	assert.Equal(t, "add logging", read.Task)
	assert.Equal(t, "agent/add-logging", read.Branch)
	assert.Equal(t, 2, read.Runs)
	assert.Equal(t, "https://forge.lthn.ai/core/go-log/pulls/5", read.PRURL)
}

func TestReadStatus_Bad_NoFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readStatus(dir)
	assert.Error(t, err)
}

func TestReadStatus_Bad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "status.json"), "not json{").OK)

	_, err := readStatus(dir)
	assert.Error(t, err)
}

func TestReadStatus_Good_BlockedWithQuestion(t *testing.T) {
	dir := t.TempDir()

	status := &WorkspaceStatus{
		Status:   "blocked",
		Agent:    "gemini",
		Repo:     "go-io",
		Question: "Which interface should I implement?",
	}

	data, err := json.MarshalIndent(status, "", "  ")
	require.NoError(t, err)
	require.True(t, fs.Write(filepath.Join(dir, "status.json"), string(data)).OK)

	read, err := readStatus(dir)
	require.NoError(t, err)

	assert.Equal(t, "blocked", read.Status)
	assert.Equal(t, "Which interface should I implement?", read.Question)
}

func TestWriteReadStatus_Good_Roundtrip(t *testing.T) {
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

	read, err := readStatus(dir)
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

func TestWriteStatus_Good_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()

	first := &WorkspaceStatus{Status: "running", Agent: "gemini"}
	err := writeStatus(dir, first)
	require.NoError(t, err)

	second := &WorkspaceStatus{Status: "completed", Agent: "gemini"}
	err = writeStatus(dir, second)
	require.NoError(t, err)

	read, err := readStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "completed", read.Status)
}

func TestReadStatus_Ugly_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "status.json"), "").OK)

	_, err := readStatus(dir)
	assert.Error(t, err)
}
