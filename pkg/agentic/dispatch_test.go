// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- agentCommand ---

// Good: tested in logic_test.go (TestAgentCommand_Good_*)
// Bad: tested in logic_test.go (TestAgentCommand_Bad_Unknown)
// Ugly: tested in logic_test.go (TestAgentCommand_Ugly_EmptyAgent)

// --- containerCommand ---

// Good: tested in logic_test.go (TestContainerCommand_Good_*)

// --- agentOutputFile ---

func TestDispatch_AgentOutputFile_Good(t *testing.T) {
	assert.Contains(t, agentOutputFile("/ws", "codex"), ".meta/agent-codex.log")
	assert.Contains(t, agentOutputFile("/ws", "claude:opus"), ".meta/agent-claude.log")
	assert.Contains(t, agentOutputFile("/ws", "gemini:flash"), ".meta/agent-gemini.log")
}

func TestDispatch_AgentOutputFile_Bad(t *testing.T) {
	// Empty agent — still produces a path (no crash)
	result := agentOutputFile("/ws", "")
	assert.Contains(t, result, ".meta/agent-.log")
}

func TestDispatch_AgentOutputFile_Ugly(t *testing.T) {
	// Agent with multiple colons — only splits on first
	result := agentOutputFile("/ws", "claude:opus:latest")
	assert.Contains(t, result, "agent-claude.log")
}

// --- detectFinalStatus ---

func TestDispatch_DetectFinalStatus_Good(t *testing.T) {
	dir := t.TempDir()

	// Clean exit = completed
	status, question := detectFinalStatus(dir, 0, "completed")
	assert.Equal(t, "completed", status)
	assert.Empty(t, question)
}

func TestDispatch_DetectFinalStatus_Bad(t *testing.T) {
	dir := t.TempDir()

	// Non-zero exit code
	status, question := detectFinalStatus(dir, 1, "completed")
	assert.Equal(t, "failed", status)
	assert.Contains(t, question, "code 1")

	// Process killed
	status2, _ := detectFinalStatus(dir, 0, "killed")
	assert.Equal(t, "failed", status2)

	// Process status "failed"
	status3, _ := detectFinalStatus(dir, 0, "failed")
	assert.Equal(t, "failed", status3)
}

func TestDispatch_DetectFinalStatus_Ugly(t *testing.T) {
	dir := t.TempDir()

	// BLOCKED.md exists but is whitespace only — NOT blocked
	os.WriteFile(core.JoinPath(dir, "BLOCKED.md"), []byte("  \n  "), 0o644)
	status, _ := detectFinalStatus(dir, 0, "completed")
	assert.Equal(t, "completed", status)

	// BLOCKED.md takes precedence over non-zero exit
	os.WriteFile(core.JoinPath(dir, "BLOCKED.md"), []byte("Need credentials"), 0o644)
	status2, question2 := detectFinalStatus(dir, 1, "failed")
	assert.Equal(t, "blocked", status2)
	assert.Equal(t, "Need credentials", question2)
}

// --- trackFailureRate ---

func TestDispatch_TrackFailureRate_Good(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: map[string]int{"codex": 2}}

	// Success resets count
	triggered := s.trackFailureRate("codex", "completed", time.Now().Add(-10*time.Second))
	assert.False(t, triggered)
	assert.Equal(t, 0, s.failCount["codex"])
}

func TestDispatch_TrackFailureRate_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: map[string]int{"codex": 2}}

	// 3rd fast failure triggers backoff
	triggered := s.trackFailureRate("codex", "failed", time.Now().Add(-10*time.Second))
	assert.True(t, triggered)
	assert.True(t, time.Now().Before(s.backoff["codex"]))
}

func TestDispatch_TrackFailureRate_Ugly(t *testing.T) {
	s := newPrepWithProcess()

	// Slow failure (>60s) resets count instead of incrementing
	s.failCount["codex"] = 2
	s.trackFailureRate("codex", "failed", time.Now().Add(-5*time.Minute))
	assert.Equal(t, 0, s.failCount["codex"])

	// Model variant tracks by base pool
	s.trackFailureRate("codex:gpt-5.4", "failed", time.Now().Add(-10*time.Second))
	assert.Equal(t, 1, s.failCount["codex"])
}

// --- startIssueTracking ---

func TestDispatch_StartIssueTracking_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Org: "core", Issue: 15}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: forge.NewForge(srv.URL, "tok"), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.startIssueTracking(dir)
}

func TestDispatch_StartIssueTracking_Bad(t *testing.T) {
	// No forge — returns early
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.startIssueTracking(t.TempDir())

	// No status file
	s2 := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s2.startIssueTracking(t.TempDir())
}

func TestDispatch_StartIssueTracking_Ugly(t *testing.T) {
	// Status has no issue — early return
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "running", Repo: "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: forge.NewForge("http://invalid", "tok"), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.startIssueTracking(dir) // no issue → skips API call
}

// --- stopIssueTracking ---

func TestDispatch_StopIssueTracking_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "completed", Repo: "go-io", Issue: 10}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: forge.NewForge(srv.URL, "tok"), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.stopIssueTracking(dir)
}

func TestDispatch_StopIssueTracking_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.stopIssueTracking(t.TempDir())
}

func TestDispatch_StopIssueTracking_Ugly(t *testing.T) {
	// Status has no issue
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "completed", Repo: "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: forge.NewForge("http://invalid", "tok"), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.stopIssueTracking(dir)
}

// --- broadcastStart ---

func TestDispatch_BroadcastStart_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "ws-test")
	fs.EnsureDir(wsDir)
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(WorkspaceStatus{Repo: "go-io", Agent: "codex"}))

	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastStart("codex", wsDir)
}

func TestDispatch_BroadcastStart_Bad(t *testing.T) {
	// No Core — should not panic
	s := &PrepSubsystem{ServiceRuntime: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastStart("codex", t.TempDir())
}

func TestDispatch_BroadcastStart_Ugly(t *testing.T) {
	// No status file — broadcasts with empty repo
	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastStart("codex", t.TempDir())
}

// --- broadcastComplete ---

func TestDispatch_BroadcastComplete_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "workspace", "ws-test")
	os.MkdirAll(wsDir, 0o755)
	data, _ := json.Marshal(WorkspaceStatus{Repo: "go-io", Agent: "codex"})
	os.WriteFile(core.JoinPath(wsDir, "status.json"), data, 0o644)

	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastComplete("codex", wsDir, "completed")
}

func TestDispatch_BroadcastComplete_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastComplete("codex", t.TempDir(), "failed")
}

func TestDispatch_BroadcastComplete_Ugly(t *testing.T) {
	// No status file
	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastComplete("codex", t.TempDir(), "completed")
}

// --- onAgentComplete ---

func TestDispatch_OnAgentComplete_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "ws-test")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(metaDir, 0o755)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	data, _ := json.Marshal(st)
	os.WriteFile(core.JoinPath(wsDir, "status.json"), data, 0o644)

	s := newPrepWithProcess()
	outputFile := core.JoinPath(metaDir, "agent-codex.log")
	s.onAgentComplete("codex", wsDir, outputFile, 0, "completed", "test output")

	updated, err := ReadStatus(wsDir)
	require.NoError(t, err)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, 0, updated.PID)

	content, _ := os.ReadFile(outputFile)
	assert.Equal(t, "test output", string(content))
}

func TestDispatch_OnAgentComplete_Bad(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "ws-fail")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(metaDir, 0o755)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	data, _ := json.Marshal(st)
	os.WriteFile(core.JoinPath(wsDir, "status.json"), data, 0o644)

	s := newPrepWithProcess()
	s.onAgentComplete("codex", wsDir, core.JoinPath(metaDir, "agent-codex.log"), 1, "failed", "error")

	updated, _ := ReadStatus(wsDir)
	assert.Equal(t, "failed", updated.Status)
	assert.Contains(t, updated.Question, "code 1")
}

func TestDispatch_OnAgentComplete_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	wsDir := core.JoinPath(root, "ws-blocked")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	os.MkdirAll(repoDir, 0o755)
	os.MkdirAll(metaDir, 0o755)

	os.WriteFile(core.JoinPath(repoDir, "BLOCKED.md"), []byte("Need credentials"), 0o644)
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	data, _ := json.Marshal(st)
	os.WriteFile(core.JoinPath(wsDir, "status.json"), data, 0o644)

	s := newPrepWithProcess()
	s.onAgentComplete("codex", wsDir, core.JoinPath(metaDir, "agent-codex.log"), 0, "completed", "")

	updated, _ := ReadStatus(wsDir)
	assert.Equal(t, "blocked", updated.Status)
	assert.Equal(t, "Need credentials", updated.Question)

	// Empty output should NOT create log file
	_, err := os.Stat(core.JoinPath(metaDir, "agent-codex.log"))
	assert.True(t, os.IsNotExist(err))
}

// --- runQA ---

func TestDispatch_RunQA_Good(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	os.MkdirAll(repoDir, 0o755)
	os.WriteFile(core.JoinPath(repoDir, "go.mod"), []byte("module testmod\n\ngo 1.22\n"), 0o644)
	os.WriteFile(core.JoinPath(repoDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	s := newPrepWithProcess()
	assert.True(t, s.runQA(wsDir))
}

func TestDispatch_RunQA_Bad(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	os.MkdirAll(repoDir, 0o755)

	// Broken Go code
	os.WriteFile(core.JoinPath(repoDir, "go.mod"), []byte("module testmod\n\ngo 1.22\n"), 0o644)
	os.WriteFile(core.JoinPath(repoDir, "main.go"), []byte("package main\nfunc main( {\n}\n"), 0o644)

	s := newPrepWithProcess()
	assert.False(t, s.runQA(wsDir))

	// PHP project — composer not available
	wsDir2 := t.TempDir()
	repoDir2 := core.JoinPath(wsDir2, "repo")
	os.MkdirAll(repoDir2, 0o755)
	os.WriteFile(core.JoinPath(repoDir2, "composer.json"), []byte(`{"name":"test"}`), 0o644)

	assert.False(t, s.runQA(wsDir2))
}

func TestDispatch_RunQA_Ugly(t *testing.T) {
	// Unknown language — passes QA (no checks)
	wsDir := t.TempDir()
	os.MkdirAll(core.JoinPath(wsDir, "repo"), 0o755)

	s := newPrepWithProcess()
	assert.True(t, s.runQA(wsDir))

	// Go vet failure (compiles but bad printf)
	wsDir2 := t.TempDir()
	repoDir2 := core.JoinPath(wsDir2, "repo")
	os.MkdirAll(repoDir2, 0o755)
	os.WriteFile(core.JoinPath(repoDir2, "go.mod"), []byte("module testmod\n\ngo 1.22\n"), 0o644)
	os.WriteFile(core.JoinPath(repoDir2, "main.go"), []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Printf(\"%d\", \"x\") }\n"), 0o644)
	assert.False(t, s.runQA(wsDir2))

	// Node project — npm install likely fails
	wsDir3 := t.TempDir()
	repoDir3 := core.JoinPath(wsDir3, "repo")
	os.MkdirAll(repoDir3, 0o755)
	os.WriteFile(core.JoinPath(repoDir3, "package.json"), []byte(`{"name":"test","scripts":{"test":"echo ok"}}`), 0o644)
	_ = s.runQA(wsDir3) // exercises the node path
}

// --- dispatch ---

func TestDispatch_Dispatch_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	forgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"title": "Issue", "body": "Fix"})
	}))
	t.Cleanup(forgeSrv.Close)

	srcRepo := core.JoinPath(t.TempDir(), "core", "go-io")
	exec.Command("git", "init", "-b", "main", srcRepo).Run()
	exec.Command("git", "-C", srcRepo, "config", "user.name", "T").Run()
	exec.Command("git", "-C", srcRepo, "config", "user.email", "t@t.com").Run()
	os.WriteFile(core.JoinPath(srcRepo, "go.mod"), []byte("module test\ngo 1.22\n"), 0o644)
	exec.Command("git", "-C", srcRepo, "add", ".").Run()
	exec.Command("git", "-C", srcRepo, "commit", "-m", "init").Run()

	s := newPrepWithProcess()
	s.forge = forge.NewForge(forgeSrv.URL, "tok")
	s.codePath = core.PathDir(core.PathDir(srcRepo))

	_, out, err := s.dispatch(context.Background(), nil, DispatchInput{
		Repo: "go-io", Task: "Fix stuff", Issue: 42, DryRun: true,
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.Equal(t, "codex", out.Agent)
	assert.NotEmpty(t, out.Prompt)
}

func TestDispatch_Dispatch_Bad(t *testing.T) {
	s := newPrepWithProcess()

	// No repo
	_, _, err := s.dispatch(context.Background(), nil, DispatchInput{Task: "do"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")

	// No task
	_, _, err = s.dispatch(context.Background(), nil, DispatchInput{Repo: "go-io"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task is required")
}

func TestDispatch_Dispatch_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Prep fails (no local clone)
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir(), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, _, err := s.dispatch(context.Background(), nil, DispatchInput{
		Repo: "nonexistent", Task: "do", Issue: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prep workspace failed")
}

// --- workspaceDir ---

func TestDispatch_WorkspaceDir_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	dir, err := workspaceDir("core", "go-io", PrepInput{Issue: 42})
	require.NoError(t, err)
	assert.Contains(t, dir, "task-42")

	dir2, _ := workspaceDir("core", "go-io", PrepInput{PR: 7})
	assert.Contains(t, dir2, "pr-7")

	dir3, _ := workspaceDir("core", "go-io", PrepInput{Branch: "feat/new"})
	assert.Contains(t, dir3, "feat/new")

	dir4, _ := workspaceDir("core", "go-io", PrepInput{Tag: "v1.0.0"})
	assert.Contains(t, dir4, "v1.0.0")
}

func TestDispatch_WorkspaceDir_Bad(t *testing.T) {
	_, err := workspaceDir("core", "go-io", PrepInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "one of issue, pr, branch, or tag")
}

func TestDispatch_WorkspaceDir_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// PR takes precedence when multiple set (first match)
	dir, err := workspaceDir("core", "go-io", PrepInput{PR: 3, Issue: 5})
	require.NoError(t, err)
	assert.Contains(t, dir, "pr-3")
}

// --- containerCommand ---

func TestDispatch_ContainerCommand_Bad(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	// Empty command string — docker still runs, just with no command after image
	cmd, args := containerCommand("codex", "", []string{}, "/ws/repo", "/ws/.meta")
	assert.Equal(t, "docker", cmd)
	assert.Contains(t, args, "run")
	// The image should still be present in args
	assert.Contains(t, args, defaultDockerImage)
}

// --- canDispatchAgent ---
// Good: tested in queue_test.go
// Bad: tested in queue_test.go
// Ugly: see queue_extra_test.go
