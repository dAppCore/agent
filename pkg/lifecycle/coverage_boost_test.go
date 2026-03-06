package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"forge.lthn.ai/core/go/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// service.go — NewService, OnStartup, handleTask, doCommit, doPrompt
// ============================================================================

func TestNewService_Good(t *testing.T) {
	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	factory := NewService(opts)

	result, err := factory(c)
	require.NoError(t, err)
	require.NotNil(t, result)

	svc, ok := result.(*Service)
	require.True(t, ok, "factory should return *Service")
	assert.Equal(t, opts.DefaultTools, svc.Opts().DefaultTools)
	assert.Equal(t, opts.AllowEdit, svc.Opts().AllowEdit)
}

func TestNewService_Good_CustomOpts(t *testing.T) {
	c, err := core.New()
	require.NoError(t, err)

	opts := ServiceOptions{
		DefaultTools: []string{"Bash", "Read", "Write", "Edit"},
		AllowEdit:    true,
	}
	factory := NewService(opts)

	result, err := factory(c)
	require.NoError(t, err)

	svc := result.(*Service)
	assert.True(t, svc.Opts().AllowEdit)
	assert.Len(t, svc.Opts().DefaultTools, 4)
}

func TestOnStartup_Good(t *testing.T) {
	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	err = svc.OnStartup(context.Background())
	assert.NoError(t, err)
}

// mockClaude creates a mock "claude" binary that exits with code 1 and
// prepends its directory to PATH, restoring PATH when the test finishes.
func mockClaude(t *testing.T) {
	t.Helper()
	mockBin := filepath.Join(t.TempDir(), "claude")
	err := os.WriteFile(mockBin, []byte("#!/bin/sh\nexit 1\n"), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(mockBin)+":"+origPath)
}

func TestHandleTask_Good_TaskCommit(t *testing.T) {
	mockClaude(t)

	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	task := TaskCommit{
		Path:    t.TempDir(),
		Name:    "test",
		CanEdit: false,
	}

	result, handled, err := svc.handleTask(c, task)
	assert.Nil(t, result)
	assert.True(t, handled, "TaskCommit should be handled")
	assert.Error(t, err, "mock claude should exit 1")
}

func TestHandleTask_Good_TaskCommitCanEdit(t *testing.T) {
	mockClaude(t)

	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	task := TaskCommit{
		Path:    t.TempDir(),
		Name:    "test-edit",
		CanEdit: true,
	}

	result, handled, err := svc.handleTask(c, task)
	assert.Nil(t, result)
	assert.True(t, handled)
	assert.Error(t, err, "mock claude should exit 1")
}

func TestHandleTask_Good_TaskPrompt(t *testing.T) {
	mockClaude(t)

	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	task := TaskPrompt{
		Prompt:  "test prompt",
		WorkDir: t.TempDir(),
	}

	result, handled, err := svc.handleTask(c, task)
	assert.Nil(t, result)
	assert.True(t, handled, "TaskPrompt should be handled")
	assert.Error(t, err, "mock claude should exit 1")
}

func TestHandleTask_Good_TaskPromptWithTaskID(t *testing.T) {
	mockClaude(t)

	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	task := TaskPrompt{
		Prompt:  "test prompt",
		WorkDir: t.TempDir(),
		taskID:  "task-123",
	}

	result, handled, err := svc.handleTask(c, task)
	assert.Nil(t, result)
	assert.True(t, handled)
	assert.Error(t, err, "mock claude should exit 1")
}

func TestHandleTask_Good_TaskPromptWithCustomTools(t *testing.T) {
	mockClaude(t)

	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	task := TaskPrompt{
		Prompt:       "test prompt",
		WorkDir:      t.TempDir(),
		AllowedTools: []string{"Bash", "Read"},
	}

	result, handled, err := svc.handleTask(c, task)
	assert.Nil(t, result)
	assert.True(t, handled)
	assert.Error(t, err, "mock claude should exit 1")
}

func TestHandleTask_Good_TaskPromptEmptyDefaultTools(t *testing.T) {
	mockClaude(t)

	c, err := core.New()
	require.NoError(t, err)

	opts := ServiceOptions{
		DefaultTools: nil, // empty tools
	}
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	task := TaskPrompt{
		Prompt:  "test prompt",
		WorkDir: t.TempDir(),
	}

	result, handled, err := svc.handleTask(c, task)
	assert.Nil(t, result)
	assert.True(t, handled)
	assert.Error(t, err, "mock claude should exit 1")
}

func TestHandleTask_Good_UnknownTask(t *testing.T) {
	c, err := core.New()
	require.NoError(t, err)

	opts := DefaultServiceOptions()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, opts),
	}

	// A string is not a recognised task type.
	result, handled, err := svc.handleTask(c, "unknown-task")
	assert.Nil(t, result)
	assert.False(t, handled, "unknown task should not be handled")
	assert.NoError(t, err)
}

// ============================================================================
// completion.go — CreatePR full coverage
// ============================================================================

func TestCreatePR_Good_WithGhMock(t *testing.T) {
	dir := initGitRepo(t)

	// Create a script that pretends to be "gh"
	mockBin := filepath.Join(t.TempDir(), "gh")
	mockScript := `#!/bin/sh
echo "https://github.com/owner/repo/pull/42"
`
	err := os.WriteFile(mockBin, []byte(mockScript), 0755)
	require.NoError(t, err)

	// Prepend mock bin directory to PATH
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Dir(mockBin)+":"+origPath)
	defer func() { _ = os.Setenv("PATH", origPath) }()

	task := &Task{
		ID:          "PR-10",
		Title:       "Test PR",
		Description: "Test PR description",
		Priority:    PriorityMedium,
	}

	prURL, err := CreatePR(context.Background(), task, dir, PROptions{})
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/owner/repo/pull/42", prURL)
}

func TestCreatePR_Good_WithAllOptions(t *testing.T) {
	dir := initGitRepo(t)

	mockBin := filepath.Join(t.TempDir(), "gh")
	mockScript := `#!/bin/sh
echo "https://github.com/owner/repo/pull/99"
`
	err := os.WriteFile(mockBin, []byte(mockScript), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Dir(mockBin)+":"+origPath)
	defer func() { _ = os.Setenv("PATH", origPath) }()

	task := &Task{
		ID:          "PR-20",
		Title:       "Full options PR",
		Description: "All options test",
		Priority:    PriorityHigh,
		Labels:      []string{"enhancement", "v2"},
	}

	opts := PROptions{
		Title:  "Custom title",
		Body:   "Custom body",
		Draft:  true,
		Labels: []string{"enhancement", "v2"},
		Base:   "develop",
	}

	prURL, err := CreatePR(context.Background(), task, dir, opts)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/owner/repo/pull/99", prURL)
}

func TestCreatePR_Good_DefaultTitleAndBody(t *testing.T) {
	dir := initGitRepo(t)

	mockBin := filepath.Join(t.TempDir(), "gh")
	mockScript := `#!/bin/sh
echo "https://github.com/owner/repo/pull/55"
`
	err := os.WriteFile(mockBin, []byte(mockScript), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Dir(mockBin)+":"+origPath)
	defer func() { _ = os.Setenv("PATH", origPath) }()

	task := &Task{
		ID:          "PR-30",
		Title:       "Default title from task",
		Description: "Default body from task description",
		Priority:    PriorityCritical,
	}

	// Empty PROptions — title and body should default from task.
	prURL, err := CreatePR(context.Background(), task, dir, PROptions{})
	require.NoError(t, err)
	assert.Contains(t, prURL, "pull/55")
}

func TestCreatePR_Bad_GhFails(t *testing.T) {
	dir := initGitRepo(t)

	mockBin := filepath.Join(t.TempDir(), "gh")
	mockScript := `#!/bin/sh
echo "error: not logged in" >&2
exit 1
`
	err := os.WriteFile(mockBin, []byte(mockScript), 0755)
	require.NoError(t, err)

	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Dir(mockBin)+":"+origPath)
	defer func() { _ = os.Setenv("PATH", origPath) }()

	task := &Task{
		ID:    "PR-40",
		Title: "Failing PR",
	}

	prURL, err := CreatePR(context.Background(), task, dir, PROptions{})
	assert.Error(t, err)
	assert.Empty(t, prURL)
	assert.Contains(t, err.Error(), "failed to create PR")
}

func TestCreatePR_Bad_GhNotFound(t *testing.T) {
	dir := initGitRepo(t)

	// Set PATH to an empty directory so "gh" is not found.
	emptyDir := t.TempDir()
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", emptyDir)
	defer func() { _ = os.Setenv("PATH", origPath) }()

	task := &Task{
		ID:    "PR-50",
		Title: "No gh binary",
	}

	prURL, err := CreatePR(context.Background(), task, dir, PROptions{})
	assert.Error(t, err)
	assert.Empty(t, prURL)
}

// ============================================================================
// completion.go — PushChanges success path
// ============================================================================

func TestPushChanges_Good_WithRemote(t *testing.T) {
	// Create a bare remote repo and a working repo that pushes to it.
	remoteDir := t.TempDir()
	_, err := runCommandCtx(context.Background(), remoteDir, "git", "init", "--bare")
	require.NoError(t, err)

	dir := initGitRepo(t)
	_, err = runCommandCtx(context.Background(), dir, "git", "remote", "add", "origin", remoteDir)
	require.NoError(t, err)

	// Push the initial commit to set up upstream.
	branch, err := GetCurrentBranch(context.Background(), dir)
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "push", "-u", "origin", branch)
	require.NoError(t, err)

	// Create a new commit to push.
	err = os.WriteFile(filepath.Join(dir, "push-test.txt"), []byte("push me\n"), 0644)
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "add", "-A")
	require.NoError(t, err)
	_, err = runCommandCtx(context.Background(), dir, "git", "commit", "-m", "push test")
	require.NoError(t, err)

	err = PushChanges(context.Background(), dir)
	assert.NoError(t, err)
}

// ============================================================================
// completion.go — CreateBranch in non-git dir
// ============================================================================

func TestCreateBranch_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	task := &Task{
		ID:    "BR-99",
		Title: "Not a repo",
	}

	branchName, err := CreateBranch(context.Background(), task, dir)
	assert.Error(t, err)
	assert.Empty(t, branchName)
	assert.Contains(t, err.Error(), "failed to create branch")
}

// ============================================================================
// config.go — SaveConfig error paths, ConfigPath
// ============================================================================

func TestSaveConfig_Good_CreatesConfigDir(t *testing.T) {
	tmpHome := t.TempDir()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	cfg := &Config{
		BaseURL: "https://test.example.com",
		Token:   "test-token-123",
	}

	err := SaveConfig(cfg)
	require.NoError(t, err)

	// Verify .core directory was created.
	info, err := os.Stat(filepath.Join(tmpHome, ".core"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestSaveConfig_Good_OverwritesExisting(t *testing.T) {
	tmpHome := t.TempDir()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Write first config.
	cfg1 := &Config{Token: "first-token"}
	err := SaveConfig(cfg1)
	require.NoError(t, err)

	// Overwrite with second config.
	cfg2 := &Config{Token: "second-token"}
	err = SaveConfig(cfg2)
	require.NoError(t, err)

	// Verify second config is saved.
	data, err := os.ReadFile(filepath.Join(tmpHome, ".core", "agentic.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "second-token")
	assert.NotContains(t, string(data), "first-token")
}

func TestConfigPath_Good_ContainsExpectedComponents(t *testing.T) {
	path, err := ConfigPath()
	require.NoError(t, err)

	// Path should end with .core/agentic.yaml.
	assert.True(t, filepath.IsAbs(path), "path should be absolute")
	dir, file := filepath.Split(path)
	assert.Equal(t, "agentic.yaml", file)
	assert.Contains(t, dir, ".core")
}

// ============================================================================
// allowance_service.go — ResetAgent error path
// ============================================================================

// resetErrorStore extends errorStore with a ResetUsage failure mode.
type resetErrorStore struct {
	*MemoryStore
	failReset bool
}

func (e *resetErrorStore) ResetUsage(agentID string) error {
	if e.failReset {
		return errors.New("simulated ResetUsage error")
	}
	return e.MemoryStore.ResetUsage(agentID)
}

func TestResetAgent_Bad_StoreError(t *testing.T) {
	store := &resetErrorStore{
		MemoryStore: NewMemoryStore(),
		failReset:   true,
	}
	svc := NewAllowanceService(store)

	err := svc.ResetAgent("agent-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reset usage")
}

func TestResetAgent_Good_Success(t *testing.T) {
	store := &resetErrorStore{
		MemoryStore: NewMemoryStore(),
		failReset:   false,
	}
	svc := NewAllowanceService(store)

	// Pre-populate some usage.
	_ = store.IncrementUsage("agent-1", 5000, 3)

	err := svc.ResetAgent("agent-1")
	require.NoError(t, err)

	usage, _ := store.GetUsage("agent-1")
	assert.Equal(t, int64(0), usage.TokensUsed)
	assert.Equal(t, 0, usage.JobsStarted)
}

// ============================================================================
// client.go — error paths for ListTasks, GetTask, ClaimTask, UpdateTask,
// CompleteTask, Ping
// ============================================================================

func TestClient_ListTasks_Bad_ConnectionRefused(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test-token")
	client.HTTPClient.Timeout = 100 * 1000000 // 100ms in nanoseconds

	tasks, err := client.ListTasks(context.Background(), ListOptions{})
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClient_ListTasks_Bad_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	tasks, err := client.ListTasks(context.Background(), ListOptions{})
	assert.Error(t, err)
	assert.Nil(t, tasks)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestClient_GetTask_Bad_ConnectionRefused(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test-token")
	client.HTTPClient.Timeout = 100 * 1000000

	task, err := client.GetTask(context.Background(), "task-1")
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClient_GetTask_Bad_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	task, err := client.GetTask(context.Background(), "task-1")
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestClient_ClaimTask_Bad_ConnectionRefused(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test-token")
	client.HTTPClient.Timeout = 100 * 1000000

	task, err := client.ClaimTask(context.Background(), "task-1")
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClient_ClaimTask_Bad_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("completely broken json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	client.AgentID = "agent-1"
	task, err := client.ClaimTask(context.Background(), "task-1")
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestClient_UpdateTask_Bad_ConnectionRefused(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test-token")
	client.HTTPClient.Timeout = 100 * 1000000

	err := client.UpdateTask(context.Background(), "task-1", TaskUpdate{
		Status: StatusInProgress,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClient_UpdateTask_Bad_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(APIError{Message: "server error"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.UpdateTask(context.Background(), "task-1", TaskUpdate{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
}

func TestClient_CompleteTask_Bad_ConnectionRefused(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", "test-token")
	client.HTTPClient.Timeout = 100 * 1000000

	err := client.CompleteTask(context.Background(), "task-1", TaskResult{
		Success: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestClient_CompleteTask_Bad_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(APIError{Message: "bad request"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.CompleteTask(context.Background(), "task-1", TaskResult{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad request")
}

func TestClient_Ping_Bad_ServerReturns5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 503")
}

// ============================================================================
// context.go — BuildTaskContext edge cases
// ============================================================================

func TestBuildTaskContext_Good_FilesGatherError(t *testing.T) {
	// Task with files but in a non-existent directory.
	task := &Task{
		ID:    "ctx-err-1",
		Title: "Files error test",
		Files: []string{"nonexistent.go"},
	}

	dir := t.TempDir()
	ctx, err := BuildTaskContext(task, dir)
	require.NoError(t, err, "BuildTaskContext should not fail even if files are missing")
	assert.NotNil(t, ctx)
	assert.Empty(t, ctx.Files, "no files should be gathered")
}

// ============================================================================
// completion.go — generateBranchName edge cases
// ============================================================================

func TestGenerateBranchName_Good_TestsLabel(t *testing.T) {
	task := &Task{
		ID:     "GEN-1",
		Title:  "Add tests for core",
		Labels: []string{"tests"},
	}
	name := generateBranchName(task)
	assert.Equal(t, "test/GEN-1-add-tests-for-core", name)
}

func TestGenerateBranchName_Good_EmptyTitle(t *testing.T) {
	task := &Task{
		ID:     "GEN-2",
		Title:  "",
		Labels: nil,
	}
	name := generateBranchName(task)
	assert.Equal(t, "feat/GEN-2-", name)
}

func TestGenerateBranchName_Good_BugfixLabel(t *testing.T) {
	task := &Task{
		ID:     "GEN-3",
		Title:  "Fix memory leak",
		Labels: []string{"bugfix"},
	}
	name := generateBranchName(task)
	assert.Equal(t, "fix/GEN-3-fix-memory-leak", name)
}

func TestGenerateBranchName_Good_DocsLabel(t *testing.T) {
	task := &Task{
		ID:     "GEN-4",
		Title:  "Update docs",
		Labels: []string{"docs"},
	}
	name := generateBranchName(task)
	assert.Equal(t, "docs/GEN-4-update-docs", name)
}

func TestGenerateBranchName_Good_FixLabel(t *testing.T) {
	task := &Task{
		ID:     "GEN-5",
		Title:  "Fix something",
		Labels: []string{"fix"},
	}
	name := generateBranchName(task)
	assert.Equal(t, "fix/GEN-5-fix-something", name)
}

// ============================================================================
// AutoCommit additional edge cases
// ============================================================================

func TestAutoCommit_Bad_NotAGitRepo(t *testing.T) {
	dir := t.TempDir()

	task := &Task{ID: "AC-1", Title: "Not a repo"}
	err := AutoCommit(context.Background(), task, dir, "feat: test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to stage changes")
}
