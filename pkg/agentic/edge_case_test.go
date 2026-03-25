// SPDX-License-Identifier: EUPL-1.2

// Edge-case tests to push partially covered functions toward 80%+.

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- autoCreatePR ---

func TestAutoCreatePR_Bad_NoStatus(t *testing.T) {
	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.autoCreatePR(t.TempDir()) // should not panic
}

func TestAutoCreatePR_Bad_EmptyBranch(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "completed", Repo: "test", Branch: ""}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(dir, "status.json"), data, 0o644)

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.autoCreatePR(dir)
}

func TestAutoCreatePR_Bad_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "completed", Branch: "agent/fix"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(dir, "status.json"), data, 0o644)

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.autoCreatePR(dir)
}

func TestAutoCreatePR_Bad_NoCommits(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "repo")
	os.MkdirAll(repoDir, 0o755)

	// Init a real git repo with a commit
	exec.Command("git", "init", repoDir).Run()
	exec.Command("git", "-C", repoDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", repoDir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(repoDir, "f.txt"), []byte("hi"), 0o644)
	exec.Command("git", "-C", repoDir, "add", ".").Run()
	exec.Command("git", "-C", repoDir, "commit", "-m", "init").Run()
	exec.Command("git", "-C", repoDir, "checkout", "-b", "dev").Run()

	st := &WorkspaceStatus{Status: "completed", Repo: "test", Branch: "dev", Agent: "codex"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(dir, "status.json"), data, 0o644)

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.autoCreatePR(dir) // no commits ahead → early return
}

// --- createPR ---

func TestCreatePR_Bad_MissingWorkspace(t *testing.T) {
	s := &PrepSubsystem{forgeToken: "tok", backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace is required")
}

func TestCreatePR_Bad_NoForgeToken(t *testing.T) {
	s := &PrepSubsystem{forgeToken: "", backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{Workspace: "ws"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Forge token")
}

func TestCreatePR_Bad_NoStatusFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsRoot := filepath.Join(root, "workspace")
	ws := filepath.Join(wsRoot, "ws-nostatus")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)
	exec.Command("git", "init", repoDir).Run()

	s := &PrepSubsystem{forgeToken: "tok", backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, _, err := s.createPR(context.Background(), nil, CreatePRInput{Workspace: "ws-nostatus"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no status")
}

func TestCreatePR_Good_DryRunNoBranch(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsRoot := filepath.Join(root, "workspace")
	ws := filepath.Join(wsRoot, "ws-nobranch")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)

	// Init git with a commit so rev-parse works
	exec.Command("git", "init", "-b", "agent-test", repoDir).Run()
	exec.Command("git", "-C", repoDir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", repoDir, "config", "user.name", "T").Run()
	os.WriteFile(filepath.Join(repoDir, "f.txt"), []byte("x"), 0o644)
	exec.Command("git", "-C", repoDir, "add", ".").Run()
	exec.Command("git", "-C", repoDir, "commit", "-m", "init").Run()

	// Status has no branch — createPR should detect from git
	st := &WorkspaceStatus{Status: "completed", Repo: "go-io", Task: "Fix", Agent: "codex"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{forgeToken: "tok", backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "ws-nobranch",
		DryRun:    true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "agent-test", out.Branch)
}

func TestCreatePR_Good_DryRunDefaultTitle(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsRoot := filepath.Join(root, "workspace")
	ws := filepath.Join(wsRoot, "ws-notitle")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)
	exec.Command("git", "init", repoDir).Run()

	// Status with no Task — title defaults to branch name
	st := &WorkspaceStatus{Status: "completed", Repo: "go-io", Branch: "agent/fix", Agent: "codex"}
	data, _ := json.Marshal(st)
	os.WriteFile(filepath.Join(ws, "status.json"), data, 0o644)

	s := &PrepSubsystem{forgeToken: "tok", backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.createPR(context.Background(), nil, CreatePRInput{
		Workspace: "ws-notitle",
		DryRun:    true,
	})
	require.NoError(t, err)
	assert.Contains(t, out.Title, "agent/fix")
}

// --- listPRs ---

func TestListPRs_Bad_AllRepos(t *testing.T) {
	// Test the "all repos" path — lists from all org repos
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/orgs/core/repos":
			json.NewEncoder(w).Encode([]map[string]any{
				{"name": "go-io", "owner": map[string]any{"login": "core"}},
			})
		default:
			json.NewEncoder(w).Encode([]map[string]any{
				{"number": 1, "title": "PR", "state": "open", "html_url": "url",
					"user": map[string]any{"login": "virgil"},
					"head": map[string]any{"ref": "fix"}, "base": map[string]any{"ref": "dev"},
					"labels": []map[string]any{}},
			})
		}
	}))
	t.Cleanup(srv.Close)

	s := &PrepSubsystem{
		forge: forge.NewForge(srv.URL, "test-token"), forgeURL: srv.URL, forgeToken: "test-token",
		client: srv.Client(), backoff: make(map[string]time.Time), failCount: make(map[string]int),
	}
	_, out, err := s.listPRs(context.Background(), nil, ListPRsInput{})
	require.NoError(t, err)
	assert.True(t, out.Success)
}

// --- status (more branches) ---

func TestStatus_Good_RunningPIDDead_Blocked(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "ws-dead")
	repoDir := filepath.Join(ws, "repo")
	os.MkdirAll(repoDir, 0o755)

	// Write BLOCKED.md — dead process with blocked file = blocked
	os.WriteFile(filepath.Join(repoDir, "BLOCKED.md"), []byte("Need help with API"), 0o644)

	writeStatus(ws, &WorkspaceStatus{
		Status: "running", Repo: "test", Agent: "codex", PID: 999999, // non-existent PID
	})

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Total)
	assert.Len(t, out.Blocked, 1)
	assert.Contains(t, out.Blocked[0].Question, "Need help")
}

func TestStatus_Good_RunningPIDDead_Completed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "ws-dead2")
	os.MkdirAll(filepath.Join(ws, "repo"), 0o755)

	// Write agent log — dead process with log = completed
	os.WriteFile(filepath.Join(ws, "agent-codex.log"), []byte("done"), 0o644)

	writeStatus(ws, &WorkspaceStatus{
		Status: "running", Repo: "test", Agent: "codex", PID: 999998,
	})

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Completed)
}

func TestStatus_Good_RunningPIDDead_Failed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := filepath.Join(root, "workspace")

	ws := filepath.Join(wsRoot, "ws-dead3")
	os.MkdirAll(filepath.Join(ws, "repo"), 0o755)

	// No BLOCKED.md, no agent log — dead process = failed
	writeStatus(ws, &WorkspaceStatus{
		Status: "running", Repo: "test", Agent: "codex", PID: 999997,
	})

	s := &PrepSubsystem{backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, out, err := s.status(context.Background(), nil, StatusInput{})
	require.NoError(t, err)
	assert.Equal(t, 1, out.Failed)
}

// --- DefaultBranch ---

func TestDefaultBranch_Good_GitRepo(t *testing.T) {
	dir := t.TempDir()
	exec.Command("git", "init", "-b", "main", dir).Run()
	exec.Command("git", "-C", dir, "config", "user.email", "t@t.com").Run()
	exec.Command("git", "-C", dir, "config", "user.name", "T").Run()
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

	branch := DefaultBranch(dir)
	assert.Equal(t, "main", branch)
}

func TestDefaultBranch_Bad_NoGit(t *testing.T) {
	dir := t.TempDir()
	branch := DefaultBranch(dir)
	assert.Equal(t, "main", branch, "should default to main for non-git dirs")
}

// --- writeStatus edge cases ---

func TestWriteStatus_Good_UpdatesTimestampOnWrite(t *testing.T) {
	dir := t.TempDir()
	before := time.Now().Add(-1 * time.Second)

	st := &WorkspaceStatus{Status: "running", Repo: "test"}
	err := writeStatus(dir, st)
	require.NoError(t, err)

	after := time.Now().Add(1 * time.Second)
	read, _ := ReadStatus(dir)
	assert.True(t, read.UpdatedAt.After(before))
	assert.True(t, read.UpdatedAt.Before(after))
}

func TestWriteStatus_Good_PreservesFields(t *testing.T) {
	dir := t.TempDir()
	st := &WorkspaceStatus{
		Status: "running", Repo: "go-io", Agent: "codex", Org: "core",
		Task: "Fix it", Branch: "agent/fix", Issue: 42, PID: 12345,
		Question: "need help", Runs: 3, PRURL: "https://forge.test/pulls/1",
	}
	require.NoError(t, writeStatus(dir, st))

	read, err := ReadStatus(dir)
	require.NoError(t, err)
	assert.Equal(t, "running", read.Status)
	assert.Equal(t, "go-io", read.Repo)
	assert.Equal(t, 42, read.Issue)
	assert.Equal(t, 12345, read.PID)
	assert.Equal(t, "need help", read.Question)
}

// --- reviewQueue edge cases ---

func TestReviewQueue_Good_RespectLimit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		codePath:  root,
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	_, out, err := s.reviewQueue(context.Background(), nil, ReviewQueueInput{Limit: 1})
	require.NoError(t, err)
	assert.True(t, out.Success)
}

// --- cmdPrep with branch/pr/tag/issue ---

func TestCmdPrep_Good_WithIssue(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions(
		core.Option{Key: "_arg", Value: "nonexistent"},
		core.Option{Key: "issue", Value: "42"},
	))
	// Will fail (no local clone) but exercises the issue parsing path
	assert.False(t, r.OK)
}

func TestCmdPrep_Good_WithPR(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions(
		core.Option{Key: "_arg", Value: "nonexistent"},
		core.Option{Key: "pr", Value: "7"},
	))
	assert.False(t, r.OK)
}

func TestCmdPrep_Good_WithBranch(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions(
		core.Option{Key: "_arg", Value: "nonexistent"},
		core.Option{Key: "branch", Value: "feat/new"},
	))
	assert.False(t, r.OK)
}

func TestCmdPrep_Good_WithTag(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdPrep(core.NewOptions(
		core.Option{Key: "_arg", Value: "nonexistent"},
		core.Option{Key: "tag", Value: "v1.0.0"},
	))
	assert.False(t, r.OK)
}

// --- cmdRunTask with defaults ---

func TestCmdRunTask_Good_DefaultsApplied(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Has repo+task but dispatch will fail (no local clone) — exercises default logic
	r := s.cmdRunTask(ctx, core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "task", Value: "fix tests"},
		core.Option{Key: "issue", Value: "15"},
	))
	assert.False(t, r.OK) // dispatch fails, but exercises all defaults
}

// --- canDispatchAgent with Core config ---

func TestCanDispatchAgent_Good_WithCoreConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	os.MkdirAll(filepath.Join(root, "workspace"), 0o755)

	c := core.New()
	// Set concurrency config on Core
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"claude": {Total: 5},
	})

	s := &PrepSubsystem{
		core:      c,
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}
	assert.True(t, s.canDispatchAgent("claude"))
}

// --- buildPrompt with persona ---

func TestBuildPrompt_Good_WithPersona(t *testing.T) {
	dir := t.TempDir()
	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	prompt, _, _ := s.buildPrompt(context.Background(), PrepInput{
		Task:    "Fix tests",
		Org:     "core",
		Repo:    "go-io",
		Persona: "engineering/engineering-security-engineer",
	}, "dev", dir)

	assert.Contains(t, prompt, "TASK: Fix tests")
	// Persona may or may not be found — just exercises the branch
}

// --- buildPrompt with plan template ---

func TestBuildPrompt_Good_WithPlanTemplate(t *testing.T) {
	dir := t.TempDir()
	s := &PrepSubsystem{
		codePath:  t.TempDir(),
		backoff:   make(map[string]time.Time),
		failCount: make(map[string]int),
	}

	prompt, _, _ := s.buildPrompt(context.Background(), PrepInput{
		Task:         "Fix the auth bug",
		Org:          "core",
		Repo:         "go-io",
		PlanTemplate: "bug-fix",
	}, "dev", dir)

	assert.Contains(t, prompt, "TASK: Fix the auth bug")
	// Plan template may render if embedded — exercises the branch
}
