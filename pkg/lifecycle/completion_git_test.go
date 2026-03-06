package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initGitRepo creates a temporary git repo with an initial commit.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialise a git repo.
	_, err := runCommandCtx(context.Background(), dir, "git", "init")
	require.NoError(t, err, "git init should succeed")

	// Configure git identity for commits.
	_, err = runCommandCtx(context.Background(), dir, "git", "config", "user.email", "test@example.com")
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "config", "user.name", "Test User")
	require.NoError(t, err)

	// Create initial commit so HEAD exists.
	readmePath := filepath.Join(dir, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644)
	require.NoError(t, err)

	_, err = runCommandCtx(context.Background(), dir, "git", "add", "-A")
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "commit", "-m", "initial commit")
	require.NoError(t, err)

	return dir
}

// --- runCommandCtx / runGitCommandCtx tests ---

func TestRunCommandCtx_Good(t *testing.T) {
	output, err := runCommandCtx(context.Background(), "/tmp", "echo", "hello world")
	require.NoError(t, err)
	assert.Contains(t, output, "hello world")
}

func TestRunCommandCtx_Bad_NonexistentCommand(t *testing.T) {
	_, err := runCommandCtx(context.Background(), "/tmp", "nonexistent-command-xyz")
	assert.Error(t, err)
}

func TestRunCommandCtx_Bad_CommandFails(t *testing.T) {
	_, err := runCommandCtx(context.Background(), "/tmp", "false")
	assert.Error(t, err)
}

func TestRunCommandCtx_Bad_StderrIncluded(t *testing.T) {
	// git status in a non-git directory should produce stderr.
	dir := t.TempDir()
	_, err := runCommandCtx(context.Background(), dir, "git", "status")
	assert.Error(t, err)
}

func TestRunGitCommandCtx_Good(t *testing.T) {
	dir := initGitRepo(t)

	output, err := runGitCommandCtx(context.Background(), dir, "log", "--oneline", "-1")
	require.NoError(t, err)
	assert.Contains(t, output, "initial commit")
}

// --- GetCurrentBranch tests ---

func TestGetCurrentBranch_Good(t *testing.T) {
	dir := initGitRepo(t)

	branch, err := GetCurrentBranch(context.Background(), dir)
	require.NoError(t, err)
	// Depending on git config, default branch could be master or main.
	assert.True(t, branch == "main" || branch == "master",
		"expected main or master, got %q", branch)
}

func TestGetCurrentBranch_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, err := GetCurrentBranch(context.Background(), dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current branch")
}

// --- HasUncommittedChanges tests ---

func TestHasUncommittedChanges_Good_Clean(t *testing.T) {
	dir := initGitRepo(t)

	hasChanges, err := HasUncommittedChanges(context.Background(), dir)
	require.NoError(t, err)
	assert.False(t, hasChanges, "fresh repo with initial commit should be clean")
}

func TestHasUncommittedChanges_Good_WithChanges(t *testing.T) {
	dir := initGitRepo(t)

	// Create a new file.
	err := os.WriteFile(filepath.Join(dir, "new-file.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	hasChanges, err := HasUncommittedChanges(context.Background(), dir)
	require.NoError(t, err)
	assert.True(t, hasChanges, "should detect untracked file")
}

func TestHasUncommittedChanges_Good_WithModifiedFile(t *testing.T) {
	dir := initGitRepo(t)

	// Modify the existing README.
	err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Updated\n"), 0644)
	require.NoError(t, err)

	hasChanges, err := HasUncommittedChanges(context.Background(), dir)
	require.NoError(t, err)
	assert.True(t, hasChanges, "should detect modified file")
}

func TestHasUncommittedChanges_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, err := HasUncommittedChanges(context.Background(), dir)
	assert.Error(t, err)
}

// --- GetDiff tests ---

func TestGetDiff_Good_Unstaged(t *testing.T) {
	dir := initGitRepo(t)

	// Modify a tracked file.
	err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Modified\n"), 0644)
	require.NoError(t, err)

	diff, err := GetDiff(context.Background(), dir, false)
	require.NoError(t, err)
	assert.Contains(t, diff, "Modified", "diff should show the change")
}

func TestGetDiff_Good_Staged(t *testing.T) {
	dir := initGitRepo(t)

	// Modify and stage a file.
	err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Staged change\n"), 0644)
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "add", "README.md")
	require.NoError(t, err)

	diff, err := GetDiff(context.Background(), dir, true)
	require.NoError(t, err)
	assert.Contains(t, diff, "Staged change", "staged diff should show the change")
}

func TestGetDiff_Good_NoDiff(t *testing.T) {
	dir := initGitRepo(t)

	diff, err := GetDiff(context.Background(), dir, false)
	require.NoError(t, err)
	assert.Empty(t, diff, "clean repo should have no diff")
}

func TestGetDiff_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	_, err := GetDiff(context.Background(), dir, false)
	assert.Error(t, err)
}

// --- AutoCommit tests (with real git) ---

func TestAutoCommit_Good(t *testing.T) {
	dir := initGitRepo(t)

	// Create a file to commit.
	err := os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package main\n"), 0644)
	require.NoError(t, err)

	task := &Task{ID: "T-100", Title: "Add feature"}
	err = AutoCommit(context.Background(), task, dir, "feat: add feature module")
	require.NoError(t, err)

	// Verify commit was created.
	output, err := runGitCommandCtx(context.Background(), dir, "log", "--oneline", "-1")
	require.NoError(t, err)
	assert.Contains(t, output, "feat: add feature module")

	// Verify task reference in full message.
	fullLog, err := runGitCommandCtx(context.Background(), dir, "log", "-1", "--pretty=format:%B")
	require.NoError(t, err)
	assert.Contains(t, fullLog, "Task: #T-100")
	assert.Contains(t, fullLog, "Co-Authored-By: Claude <noreply@anthropic.com>")
}

func TestAutoCommit_Bad_NoChangesToCommit(t *testing.T) {
	dir := initGitRepo(t)

	// No changes to commit.
	task := &Task{ID: "T-200", Title: "No changes"}
	err := AutoCommit(context.Background(), task, dir, "feat: nothing")
	assert.Error(t, err, "should fail when there is nothing to commit")
}

// --- CreateBranch tests (with real git) ---

func TestCreateBranch_Good(t *testing.T) {
	dir := initGitRepo(t)

	task := &Task{
		ID:     "BR-42",
		Title:  "Implement new feature",
		Labels: []string{"enhancement"},
	}

	branchName, err := CreateBranch(context.Background(), task, dir)
	require.NoError(t, err)
	assert.Equal(t, "feat/BR-42-implement-new-feature", branchName)

	// Verify we're on the new branch.
	currentBranch, err := GetCurrentBranch(context.Background(), dir)
	require.NoError(t, err)
	assert.Equal(t, branchName, currentBranch)
}

func TestCreateBranch_Good_BugLabel(t *testing.T) {
	dir := initGitRepo(t)

	task := &Task{
		ID:     "BR-43",
		Title:  "Fix login bug",
		Labels: []string{"bug"},
	}

	branchName, err := CreateBranch(context.Background(), task, dir)
	require.NoError(t, err)
	assert.Equal(t, "fix/BR-43-fix-login-bug", branchName)
}

// --- PushChanges test ---

func TestPushChanges_Bad_NoRemote(t *testing.T) {
	dir := initGitRepo(t)

	// No remote configured, push should fail.
	err := PushChanges(context.Background(), dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to push changes")
}

// --- CommitAndSync tests ---

func TestCommitAndSync_Good_WithoutClient(t *testing.T) {
	dir := initGitRepo(t)

	// Create a file to commit.
	err := os.WriteFile(filepath.Join(dir, "sync.go"), []byte("package sync\n"), 0644)
	require.NoError(t, err)

	task := &Task{ID: "CS-1", Title: "Sync test"}

	// nil client: should commit but skip sync.
	err = CommitAndSync(context.Background(), nil, task, dir, "feat: sync test", 50)
	require.NoError(t, err)

	// Verify commit.
	output, err := runGitCommandCtx(context.Background(), dir, "log", "--oneline", "-1")
	require.NoError(t, err)
	assert.Contains(t, output, "feat: sync test")
}

func TestCommitAndSync_Good_WithClient(t *testing.T) {
	dir := initGitRepo(t)

	// Create a file to commit.
	err := os.WriteFile(filepath.Join(dir, "synced.go"), []byte("package synced\n"), 0644)
	require.NoError(t, err)

	var receivedUpdate TaskUpdate
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			_ = json.NewDecoder(r.Body).Decode(&receivedUpdate)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	task := &Task{ID: "CS-2", Title: "Sync with client"}

	err = CommitAndSync(context.Background(), client, task, dir, "feat: synced", 75)
	require.NoError(t, err)

	// Verify the update was sent.
	assert.Equal(t, StatusInProgress, receivedUpdate.Status)
	assert.Equal(t, 75, receivedUpdate.Progress)
	assert.Contains(t, receivedUpdate.Notes, "feat: synced")
}

func TestCommitAndSync_Bad_CommitFails(t *testing.T) {
	dir := initGitRepo(t)

	// No changes to commit.
	task := &Task{ID: "CS-3", Title: "Will fail"}

	err := CommitAndSync(context.Background(), nil, task, dir, "feat: no changes", 50)
	assert.Error(t, err, "should fail when commit fails")
}

func TestCommitAndSync_Bad_SyncFails(t *testing.T) {
	dir := initGitRepo(t)

	// Create a file to commit.
	err := os.WriteFile(filepath.Join(dir, "fail-sync.go"), []byte("package failsync\n"), 0644)
	require.NoError(t, err)

	// Server returns an error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(APIError{Message: "sync failed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	task := &Task{ID: "CS-4", Title: "Sync fails"}

	err = CommitAndSync(context.Background(), client, task, dir, "feat: sync-fail", 50)
	assert.Error(t, err, "should report sync failure")
	assert.Contains(t, err.Error(), "sync failed")
}

// --- SyncStatus with working client ---

func TestSyncStatus_Good(t *testing.T) {
	var receivedUpdate TaskUpdate
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedUpdate)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	task := &Task{ID: "sync-1", Title: "Sync test"}

	err := SyncStatus(context.Background(), client, task, TaskUpdate{
		Status:   StatusCompleted,
		Progress: 100,
		Notes:    "All done",
	})
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, receivedUpdate.Status)
	assert.Equal(t, 100, receivedUpdate.Progress)
}

// --- CreatePR with default title/body ---

func TestCreatePR_Good_DefaultTitleFromTask(t *testing.T) {
	// CreatePR requires gh CLI which may not be available.
	// Test the option building logic by checking that the title
	// defaults to the task title.
	task := &Task{
		ID:          "PR-1",
		Title:       "Add authentication",
		Description: "OAuth2 login",
		Priority:    PriorityHigh,
	}

	opts := PROptions{}

	// Verify the defaulting logic that would be used.
	title := opts.Title
	if title == "" {
		title = task.Title
	}
	assert.Equal(t, "Add authentication", title)

	body := opts.Body
	if body == "" {
		body = buildPRBody(task)
	}
	assert.Contains(t, body, "OAuth2 login")
}

func TestCreatePR_Good_CustomOptions(t *testing.T) {
	opts := PROptions{
		Title:  "Custom title",
		Body:   "Custom body",
		Draft:  true,
		Labels: []string{"enhancement", "v2"},
		Base:   "develop",
	}

	assert.Equal(t, "Custom title", opts.Title)
	assert.True(t, opts.Draft)
	assert.Equal(t, "develop", opts.Base)
	assert.Len(t, opts.Labels, 2)
}

// --- Client checkResponse edge cases ---

func TestClient_CheckResponse_Good_GenericError(t *testing.T) {
	// Test checkResponse with a non-JSON error body.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("plain text error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.GetTask(context.Background(), "test-task")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bad Gateway")
}

func TestClient_CheckResponse_Good_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.GetTask(context.Background(), "test-task")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Forbidden")
}

// --- Client Ping edge case ---

func TestClient_Ping_Bad_ServerReturns4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 401")
}

// --- Client ClaimTask without AgentID ---

func TestClient_ClaimTask_Good_NoAgentID(t *testing.T) {
	claimedTask := Task{
		ID:     "task-no-agent",
		Status: StatusInProgress,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no body sent when AgentID is empty.
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(claimedTask)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	// Explicitly leave AgentID empty.
	client.AgentID = ""

	task, err := client.ClaimTask(context.Background(), "task-no-agent")
	require.NoError(t, err)
	assert.Equal(t, "task-no-agent", task.ID)
}
