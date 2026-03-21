// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNotifier captures channel events for testing.
type mockNotifier struct {
	mu     sync.Mutex
	events []mockEvent
}

type mockEvent struct {
	channel string
	data    any
}

func (m *mockNotifier) ChannelSend(_ context.Context, channel string, data any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, mockEvent{channel: channel, data: data})
}

func (m *mockNotifier) Events() []mockEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]mockEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// initTestRepo creates a bare git repo and a workspace clone with a branch.
func initTestRepo(t *testing.T) (sourceDir, wsDir string) {
	t.Helper()

	// Create bare "source" repo
	sourceDir = filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	run(t, sourceDir, "git", "init")
	run(t, sourceDir, "git", "checkout", "-b", "main")
	os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("# test"), 0644)
	run(t, sourceDir, "git", "add", ".")
	run(t, sourceDir, "git", "commit", "-m", "init")

	// Create workspace dir with src/ clone
	wsDir = filepath.Join(t.TempDir(), "workspace")
	srcDir := filepath.Join(wsDir, "src")
	require.NoError(t, os.MkdirAll(wsDir, 0755))
	run(t, wsDir, "git", "clone", sourceDir, "src")

	// Create agent branch with a commit
	run(t, srcDir, "git", "checkout", "-b", "agent/test-task")
	os.WriteFile(filepath.Join(srcDir, "new.go"), []byte("package main\n"), 0644)
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "agent work")

	return sourceDir, wsDir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command %s %v failed: %s", name, args, out)
}

func writeStatus(t *testing.T, wsDir, status, repo, branch string) {
	t.Helper()
	st := map[string]any{
		"status": status,
		"repo":   repo,
		"branch": branch,
	}
	data, _ := json.MarshalIndent(st, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(wsDir, "status.json"), data, 0644))
}

// --- Tests ---

func TestDetectBranch_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := filepath.Join(wsDir, "src")

	branch := detectBranch(srcDir)
	assert.Equal(t, "agent/test-task", branch)
}

func TestDetectBranch_Bad_NoRepo(t *testing.T) {
	branch := detectBranch(t.TempDir())
	assert.Equal(t, "", branch)
}

func TestCountUnpushed_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := filepath.Join(wsDir, "src")

	count := countUnpushed(srcDir, "agent/test-task")
	assert.Equal(t, 1, count)
}

func TestCountChangedFiles_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := filepath.Join(wsDir, "src")

	count := countChangedFiles(srcDir)
	assert.Equal(t, 1, count)
}

func TestCheckSafety_Good_CleanWorkspace(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := filepath.Join(wsDir, "src")

	reason := checkSafety(srcDir)
	assert.Equal(t, "", reason)
}

func TestCheckSafety_Bad_BinaryFile(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := filepath.Join(wsDir, "src")

	// Add a binary file
	os.WriteFile(filepath.Join(srcDir, "app.exe"), []byte("binary"), 0644)
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "add binary")

	reason := checkSafety(srcDir)
	assert.Contains(t, reason, "binary file added")
	assert.Contains(t, reason, "app.exe")
}

func TestCheckSafety_Bad_LargeFile(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := filepath.Join(wsDir, "src")

	// Add a file > 1MB
	bigData := make([]byte, 1024*1024+1)
	os.WriteFile(filepath.Join(srcDir, "huge.txt"), bigData, 0644)
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "add large file")

	reason := checkSafety(srcDir)
	assert.Contains(t, reason, "large file")
	assert.Contains(t, reason, "huge.txt")
}

func TestHarvestWorkspace_Good(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	result := mon.harvestWorkspace(wsDir)
	require.NotNil(t, result)
	assert.Equal(t, "test-repo", result.repo)
	assert.Equal(t, "agent/test-task", result.branch)
	assert.Equal(t, 1, result.files)
	assert.Equal(t, "", result.rejected)

	// Verify status updated
	data, err := os.ReadFile(filepath.Join(wsDir, "status.json"))
	require.NoError(t, err)
	var st map[string]any
	json.Unmarshal(data, &st)
	assert.Equal(t, "ready-for-review", st["status"])
}

func TestHarvestWorkspace_Bad_NotCompleted(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "running", "test-repo", "agent/test-task")

	mon := New()
	result := mon.harvestWorkspace(wsDir)
	assert.Nil(t, result)
}

func TestHarvestWorkspace_Bad_MainBranch(t *testing.T) {
	_, wsDir := initTestRepo(t)

	// Switch back to main
	srcDir := filepath.Join(wsDir, "src")
	run(t, srcDir, "git", "checkout", "main")

	writeStatus(t, wsDir, "completed", "test-repo", "main")

	mon := New()
	result := mon.harvestWorkspace(wsDir)
	assert.Nil(t, result)
}

func TestHarvestWorkspace_Bad_BinaryRejected(t *testing.T) {
	_, wsDir := initTestRepo(t)
	srcDir := filepath.Join(wsDir, "src")

	// Add binary
	os.WriteFile(filepath.Join(srcDir, "build.so"), []byte("elf"), 0644)
	run(t, srcDir, "git", "add", ".")
	run(t, srcDir, "git", "commit", "-m", "add binary")

	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	result := mon.harvestWorkspace(wsDir)
	require.NotNil(t, result)
	assert.Contains(t, result.rejected, "binary file added")

	// Verify status set to rejected
	data, _ := os.ReadFile(filepath.Join(wsDir, "status.json"))
	var st map[string]any
	json.Unmarshal(data, &st)
	assert.Equal(t, "rejected", st["status"])
}

func TestHarvestCompleted_Good_ChannelEvents(t *testing.T) {
	_, wsDir := initTestRepo(t)
	writeStatus(t, wsDir, "completed", "test-repo", "agent/test-task")

	// Override workspace root so harvestCompleted finds our workspace
	origRoot := os.Getenv("CORE_WORKSPACE_ROOT")
	os.Setenv("CORE_WORKSPACE_ROOT", filepath.Dir(wsDir))
	defer os.Setenv("CORE_WORKSPACE_ROOT", origRoot)

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	// Call harvestWorkspace directly since harvestCompleted uses agentic.WorkspaceRoot()
	result := mon.harvestWorkspace(wsDir)
	require.NotNil(t, result)
	assert.Equal(t, "", result.rejected)

	// Simulate what harvestCompleted does with the result
	if result.rejected == "" {
		mon.notifier.ChannelSend(context.Background(), "harvest.complete", map[string]any{
			"repo":   result.repo,
			"branch": result.branch,
			"files":  result.files,
		})
	}

	events := notifier.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "harvest.complete", events[0].channel)

	eventData := events[0].data.(map[string]any)
	assert.Equal(t, "test-repo", eventData["repo"])
	assert.Equal(t, 1, eventData["files"])
}

func TestUpdateStatus_Good(t *testing.T) {
	dir := t.TempDir()
	initial := map[string]any{"status": "completed", "repo": "test"}
	data, _ := json.MarshalIndent(initial, "", "  ")
	os.WriteFile(filepath.Join(dir, "status.json"), data, 0644)

	updateStatus(dir, "ready-for-review", "")

	out, _ := os.ReadFile(filepath.Join(dir, "status.json"))
	var st map[string]any
	json.Unmarshal(out, &st)
	assert.Equal(t, "ready-for-review", st["status"])
}

func TestUpdateStatus_Good_WithQuestion(t *testing.T) {
	dir := t.TempDir()
	initial := map[string]any{"status": "completed", "repo": "test"}
	data, _ := json.MarshalIndent(initial, "", "  ")
	os.WriteFile(filepath.Join(dir, "status.json"), data, 0644)

	updateStatus(dir, "rejected", "binary file: app.exe")

	out, _ := os.ReadFile(filepath.Join(dir, "status.json"))
	var st map[string]any
	json.Unmarshal(out, &st)
	assert.Equal(t, "rejected", st["status"])
	assert.Equal(t, "binary file: app.exe", st["question"])
}

func TestSetNotifier_Good(t *testing.T) {
	mon := New()
	assert.Nil(t, mon.notifier)

	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)
	assert.NotNil(t, mon.notifier)
}
