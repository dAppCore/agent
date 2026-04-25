// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"dappco.re/go/forge"
	"dappco.re/go/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCompletionProcess struct {
	done   chan struct{}
	info   process.Info
	output string
}

func (p *fakeCompletionProcess) Done() <-chan struct{} { return p.done }
func (p *fakeCompletionProcess) Info() process.Info    { return p.info }
func (p *fakeCompletionProcess) Output() string        { return p.output }

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
	fs.Write(core.JoinPath(dir, "BLOCKED.md"), "  \n  ")
	status, _ := detectFinalStatus(dir, 0, "completed")
	assert.Equal(t, "completed", status)

	// BLOCKED.md takes precedence over non-zero exit
	fs.Write(core.JoinPath(dir, "BLOCKED.md"), "Need credentials")
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
	c := core.New()
	var captured []messages.RateLimitDetected
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.RateLimitDetected); ok {
			captured = append(captured, ev)
		}
		return core.Result{OK: true}
	})

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: map[string]int{"codex": 2}}

	// 3rd fast failure triggers backoff
	triggered := s.trackFailureRate("codex", "failed", time.Now().Add(-10*time.Second))
	assert.True(t, triggered)
	assert.True(t, time.Now().Before(s.backoff["codex"]))
	require.Len(t, captured, 1)
	assert.Equal(t, "codex", captured[0].Pool)
	assert.Equal(t, "30m0s", captured[0].Duration)
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
	setTestWorkspace(t, root)

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
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "ws-test")
	fs.EnsureDir(wsDir)
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(WorkspaceStatus{Repo: "go-io", Agent: "codex"}))

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

// --- agentCompletionMonitor ---

func TestDispatch_AgentCompletionMonitor_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-monitor")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	fs.EnsureDir(repoDir)
	fs.EnsureDir(metaDir)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	proc := &fakeCompletionProcess{
		done:   make(chan struct{}),
		info:   process.Info{ExitCode: 0, Status: process.Status("completed")},
		output: "monitor output",
	}
	close(proc.done)

	s := newPrepWithProcess()
	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        "codex",
		workspaceDir: wsDir,
		outputFile:   core.JoinPath(metaDir, "agent-codex.log"),
		process:      proc,
	}

	r := monitor.run(context.Background(), core.NewOptions())
	assert.True(t, r.OK)

	updated := mustReadStatus(t, wsDir)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, 0, updated.PID)

	output := fs.Read(core.JoinPath(metaDir, "agent-codex.log"))
	require.True(t, output.OK)
	assert.Equal(t, "monitor output", output.Value.(string))
}

func TestDispatch_AgentCompletionMonitor_Bad(t *testing.T) {
	s := newPrepWithProcess()
	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        "codex",
		workspaceDir: t.TempDir(),
	}

	r := monitor.run(context.Background(), core.NewOptions())
	assert.False(t, r.OK)
	require.Error(t, r.Value.(error))
	assert.Contains(t, r.Value.(error).Error(), "process is required")
}

func TestDispatch_AgentCompletionMonitor_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-blocked")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	fs.EnsureDir(repoDir)
	fs.EnsureDir(metaDir)

	fs.Write(core.JoinPath(repoDir, "BLOCKED.md"), "Need credentials")
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	proc := &fakeCompletionProcess{
		done:   make(chan struct{}),
		info:   process.Info{ExitCode: 1, Status: process.Status("failed")},
		output: "",
	}
	close(proc.done)

	s := newPrepWithProcess()
	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        "codex",
		workspaceDir: wsDir,
		outputFile:   core.JoinPath(metaDir, "agent-codex.log"),
		process:      proc,
	}

	r := monitor.run(context.Background(), core.NewOptions())
	assert.True(t, r.OK)

	updated := mustReadStatus(t, wsDir)
	assert.Equal(t, "blocked", updated.Status)
	assert.Equal(t, "Need credentials", updated.Question)
	assert.False(t, fs.Exists(core.JoinPath(metaDir, "agent-codex.log")))
}

// --- onAgentComplete ---

func TestDispatch_OnAgentComplete_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-test")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	fs.EnsureDir(repoDir)
	fs.EnsureDir(metaDir)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	s := newPrepWithProcess()
	outputFile := core.JoinPath(metaDir, "agent-codex.log")
	s.onAgentComplete("codex", wsDir, outputFile, 0, "completed", "test output")

	updated := mustReadStatus(t, wsDir)
	assert.Equal(t, "completed", updated.Status)
	assert.Equal(t, 0, updated.PID)

	r := fs.Read(outputFile)
	assert.True(t, r.OK)
	assert.Equal(t, "test output", r.Value.(string))
}

func TestDispatch_OnAgentComplete_Bad(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-fail")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	fs.EnsureDir(repoDir)
	fs.EnsureDir(metaDir)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	s := newPrepWithProcess()
	s.onAgentComplete("codex", wsDir, core.JoinPath(metaDir, "agent-codex.log"), 1, "failed", "error")

	updated := mustReadStatus(t, wsDir)
	assert.Equal(t, "failed", updated.Status)
	assert.Contains(t, updated.Question, "code 1")
}

func TestDispatch_OnAgentComplete_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-blocked")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	fs.EnsureDir(repoDir)
	fs.EnsureDir(metaDir)

	fs.Write(core.JoinPath(repoDir, "BLOCKED.md"), "Need credentials")
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	s := newPrepWithProcess()
	s.onAgentComplete("codex", wsDir, core.JoinPath(metaDir, "agent-codex.log"), 0, "completed", "")

	updated := mustReadStatus(t, wsDir)
	assert.Equal(t, "blocked", updated.Status)
	assert.Equal(t, "Need credentials", updated.Question)

	// Empty output should NOT create log file
	assert.False(t, fs.Exists(core.JoinPath(metaDir, "agent-codex.log")))
}

func TestDispatch_Run_Bad_Timeout(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-timeout")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	require.True(t, fs.EnsureDir(repoDir).OK)
	require.True(t, fs.EnsureDir(metaDir).OK)

	st := &WorkspaceStatus{
		Status:    "running",
		Agent:     "codex",
		Repo:      "go-io",
		StartedAt: time.Now(),
	}
	require.NoError(t, writeStatus(wsDir, st))

	processResult := testCore.Service("process")
	require.True(t, processResult.OK)
	procSvc, ok := processResult.Value.(*process.Service)
	require.True(t, ok)

	timeout := 100 * time.Millisecond
	opts := dispatchRunOptions("sleep", []string{"60"}, repoDir, timeout)
	assert.Equal(t, timeout, opts.Timeout)
	assert.Equal(t, dispatchGracePeriod, opts.GracePeriod)
	assert.True(t, opts.KillGroup)
	assert.True(t, opts.Detach)

	proc, err := procSvc.StartWithOptions(context.Background(), opts)
	require.NoError(t, err)
	proc.CloseStdin()

	s := newPrepWithProcess()
	s.workspaces = core.NewRegistry[*WorkspaceStatus]()
	startDispatchTimeoutWatch(wsDir, timeout, proc)

	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        "codex",
		workspaceDir: wsDir,
		outputFile:   core.JoinPath(metaDir, "agent-codex.log"),
		process:      proc,
	}

	r := monitor.run(context.Background(), core.NewOptions())
	assert.True(t, r.OK)

	info := proc.Info()
	assert.Equal(t, process.StatusKilled, info.Status)

	updated := mustReadStatus(t, wsDir)
	assert.Equal(t, "failed", updated.Status)
	assert.Equal(t, dispatchTimeoutReason(timeout), updated.Question)
	assert.Equal(t, 0, updated.PID)

	registryResult := s.workspaces.Get(WorkspaceName(wsDir))
	require.True(t, registryResult.OK)
	registryStatus, ok := registryResult.Value.(*WorkspaceStatus)
	require.True(t, ok)
	assert.Equal(t, "failed", registryStatus.Status)
	assert.Equal(t, dispatchTimeoutReason(timeout), registryStatus.Question)

	assert.False(t, fs.Exists(workspaceTimeoutPath(wsDir)))
}

// --- runQA ---

func TestDispatch_RunQA_Good(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main() {}\n")

	s := newPrepWithProcess()
	assert.True(t, s.runQA(wsDir))
}

func TestDispatch_RunQA_Bad(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)

	// Broken Go code
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main( {\n}\n")

	s := newPrepWithProcess()
	assert.False(t, s.runQA(wsDir))

	// PHP project — composer not available
	wsDir2 := t.TempDir()
	repoDir2 := core.JoinPath(wsDir2, "repo")
	fs.EnsureDir(repoDir2)
	fs.Write(core.JoinPath(repoDir2, "composer.json"), `{"name":"test"}`)

	assert.False(t, s.runQA(wsDir2))
}

func TestDispatch_RunQA_Ugly(t *testing.T) {
	// Unknown language — passes QA (no checks)
	wsDir := t.TempDir()
	fs.EnsureDir(core.JoinPath(wsDir, "repo"))

	s := newPrepWithProcess()
	assert.True(t, s.runQA(wsDir))

	// Go vet failure (compiles but bad printf)
	wsDir2 := t.TempDir()
	repoDir2 := core.JoinPath(wsDir2, "repo")
	fs.EnsureDir(repoDir2)
	fs.Write(core.JoinPath(repoDir2, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir2, "main.go"), "package main\nimport \"fmt\"\nfunc main() { fmt.Printf(\"%d\", \"x\") }\n")
	assert.False(t, s.runQA(wsDir2))

	// Node project — npm install likely fails
	wsDir3 := t.TempDir()
	repoDir3 := core.JoinPath(wsDir3, "repo")
	fs.EnsureDir(repoDir3)
	fs.Write(core.JoinPath(repoDir3, "package.json"), `{"name":"test","scripts":{"test":"echo ok"}}`)
	_ = s.runQA(wsDir3) // exercises the node path
}

// --- dispatch ---

func TestDispatch_Dispatch_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	forgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{"title": "Issue", "body": "Fix"})))
	}))
	t.Cleanup(forgeSrv.Close)

	srcRepo := core.JoinPath(t.TempDir(), "core", "go-io")
	testCore.Process().Run(context.Background(), "git", "init", "-b", "main", srcRepo)
	testCore.Process().RunIn(context.Background(), srcRepo, "git", "config", "user.name", "T")
	testCore.Process().RunIn(context.Background(), srcRepo, "git", "config", "user.email", "t@t.com")
	fs.Write(core.JoinPath(srcRepo, "go.mod"), "module test\ngo 1.22\n")
	testCore.Process().RunIn(context.Background(), srcRepo, "git", "add", ".")
	testCore.Process().RunIn(context.Background(), srcRepo, "git", "commit", "-m", "init")

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
	setTestWorkspace(t, root)

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
	setTestWorkspace(t, root)

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
	setTestWorkspace(t, root)

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
	cmd, args := containerCommand("", []string{}, "/ws", "/ws/.meta")
	assert.Equal(t, "docker", cmd)
	assert.Contains(t, args, "run")
	// The image should still be present in args
	assert.Contains(t, args, defaultDockerImage)
	assert.Contains(t, args, "/ws:/workspace")
	assert.Contains(t, args, "/workspace/repo")
}

// --- canDispatchAgent ---
// Good: tested in queue_test.go
// Bad: tested in queue_test.go
// Ugly: see queue_extra_test.go

// --- agentCommand AX-10 ---

func TestDispatch_agentCommand_Good(t *testing.T) {
	command, args, err := agentCommand("codex:gpt-5.4-mini", "Implement AX-10 unit tests for Mantis #169")
	require.NoError(t, err)
	assert.Equal(t, "codex", command)
	assert.Equal(t, []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"-o", "../.meta/agent-codex.log",
		"--model", "gpt-5.4-mini",
		"Implement AX-10 unit tests for Mantis #169",
	}, args)
}

func TestDispatch_agentCommand_Bad(t *testing.T) {
	command, args, err := agentCommand("mantis", "Investigate a failing dispatch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent: mantis")
	assert.Empty(t, command)
	assert.Nil(t, args)
}

func TestDispatch_agentCommand_Ugly(t *testing.T) {
	command, args, err := agentCommand("", "Investigate a failing dispatch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
	assert.Empty(t, command)
	assert.Nil(t, args)
}
