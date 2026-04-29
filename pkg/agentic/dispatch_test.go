// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
	"dappco.re/go/forge"
	"dappco.re/go/process"
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

func TestDispatch_AgentOutputFile_Good_Case(t *testing.T) {
	core.AssertContains(t, agentOutputFile("/ws", "codex"), ".meta/agent-codex.log")
	core.AssertContains(t, agentOutputFile("/ws", "claude:opus"), ".meta/agent-claude.log")
	core.AssertContains(t, agentOutputFile("/ws", "gemini:flash"), ".meta/agent-gemini.log")
}

func TestDispatch_AgentOutputFile_Bad_Case(t *testing.T) {
	// Empty agent — still produces a path (no crash)
	result := agentOutputFile("/ws", "")
	core.AssertContains(
		t,
		result,
		".meta/agent-.log",
	)
}

func TestDispatch_AgentOutputFile_Ugly_Case(t *testing.T) {
	// Agent with multiple colons — only splits on first
	result := agentOutputFile("/ws", "claude:opus:latest")
	core.AssertContains(
		t,
		result,
		"agent-claude.log",
	)
}

// --- detectFinalStatus ---

func TestDispatch_DetectFinalStatus_Good_Case(t *testing.T) {
	dir := t.TempDir()

	// Clean exit = completed
	status, question := detectFinalStatus(dir, 0, "completed")
	core.AssertEqual(t, "completed", status)
	core.AssertEmpty(t, question)
}

func TestDispatch_DetectFinalStatus_Bad_Case(t *testing.T) {
	dir := t.TempDir()

	// Non-zero exit code
	status, question := detectFinalStatus(dir, 1, "completed")
	core.AssertEqual(t, "failed", status)
	core.AssertContains(t, question, "code 1")

	// Process killed
	status2, _ := detectFinalStatus(dir, 0, "killed")
	core.AssertEqual(t, "failed", status2)

	// Process status "failed"
	status3, _ := detectFinalStatus(dir, 0, "failed")
	core.AssertEqual(t, "failed", status3)
}

func TestDispatch_DetectFinalStatus_Ugly_Case(t *testing.T) {
	dir := t.TempDir()

	// BLOCKED.md exists but is whitespace only — NOT blocked
	fs.Write(core.JoinPath(dir, "BLOCKED.md"), "  \n  ")
	status, _ := detectFinalStatus(dir, 0, "completed")
	core.AssertEqual(t, "completed", status)

	// BLOCKED.md takes precedence over non-zero exit
	fs.Write(core.JoinPath(dir, "BLOCKED.md"), "Need credentials")
	status2, question2 := detectFinalStatus(dir, 1, "failed")
	core.AssertEqual(t, "blocked", status2)
	core.AssertEqual(t, "Need credentials", question2)
}

// --- trackFailureRate ---

func TestDispatch_TrackFailureRate_Good_Case(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), backoff: make(map[string]time.Time), failCount: map[string]int{"codex": 2}}

	// Success resets count
	triggered := s.trackFailureRate("codex", "completed", time.Now().Add(-10*time.Second))
	core.AssertFalse(t, triggered)
	core.AssertEqual(t, 0, s.failCount["codex"])
}

func TestDispatch_TrackFailureRate_Bad_Case(t *testing.T) {
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
	core.AssertTrue(t, triggered)
	core.AssertTrue(t, time.Now().Before(s.backoff["codex"]))
	core.AssertLen(t, captured, 1)
	core.AssertEqual(t, "codex", captured[0].Pool)
	core.AssertEqual(t, "30m0s", captured[0].Duration)
}

func TestDispatch_TrackFailureRate_Ugly_Case(t *testing.T) {
	s := newPrepWithProcess()

	// Slow failure (>60s) resets count instead of incrementing
	s.failCount["codex"] = 2
	s.trackFailureRate("codex", "failed", time.Now().Add(-5*time.Minute))
	core.AssertEqual(t, 0, s.failCount["codex"])

	// Model variant tracks by base pool
	s.trackFailureRate("codex:gpt-5.4", "failed", time.Now().Add(-10*time.Second))
	core.AssertEqual(t, 1, s.failCount["codex"])
}

// --- startIssueTracking ---

func TestDispatch_StartIssueTracking_Good_Case(t *testing.T) {
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

func TestDispatch_StartIssueTracking_Bad_Case(t *testing.T) {
	// No forge — returns early
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.startIssueTracking(t.TempDir())

	// No status file
	s2 := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s2.startIssueTracking(t.TempDir())
}

func TestDispatch_StartIssueTracking_Ugly_Case(t *testing.T) {
	// Status has no issue — early return
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "running", Repo: "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: forge.NewForge("http://invalid", "tok"), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.startIssueTracking(dir) // no issue → skips API call
}

// --- stopIssueTracking ---

func TestDispatch_StopIssueTracking_Good_Case(t *testing.T) {
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

func TestDispatch_StopIssueTracking_Bad_Case(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.stopIssueTracking(t.TempDir())
	core.AssertNil(t, s.forge)
}

func TestDispatch_StopIssueTracking_Ugly_Case(t *testing.T) {
	// Status has no issue
	dir := t.TempDir()
	st := &WorkspaceStatus{Status: "completed", Repo: "test"}
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(st))

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), forge: forge.NewForge("http://invalid", "tok"), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.stopIssueTracking(dir)
}

// --- broadcastStart ---

func TestDispatch_BroadcastStart_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "ws-test")
	fs.EnsureDir(wsDir)
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(WorkspaceStatus{Repo: "go-io", Agent: "codex"}))

	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastStart("codex", wsDir)
}

func TestDispatch_BroadcastStart_Bad_Case(t *testing.T) {
	// No Core — should not panic
	s := &PrepSubsystem{ServiceRuntime: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastStart("codex", t.TempDir())
	core.AssertNil(t, s.ServiceRuntime)
}

func TestDispatch_BroadcastStart_Ugly_Case(t *testing.T) {
	// No status file — broadcasts with empty repo
	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastStart("codex", t.TempDir())
}

// --- broadcastComplete ---

func TestDispatch_BroadcastComplete_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "workspace", "ws-test")
	fs.EnsureDir(wsDir)
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(WorkspaceStatus{Repo: "go-io", Agent: "codex"}))

	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastComplete("codex", wsDir, "completed")
}

func TestDispatch_BroadcastComplete_Bad_Case(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: nil, backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastComplete("codex", t.TempDir(), "failed")
	core.AssertNil(t, s.ServiceRuntime)
}

func TestDispatch_BroadcastComplete_Ugly_Case(t *testing.T) {
	// No status file
	c := core.New()
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	s.broadcastComplete("codex", t.TempDir(), "completed")
}

// --- agentCompletionMonitor ---

func TestDispatch_AgentCompletionMonitor_Good_Case(t *testing.T) {
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
	core.AssertTrue(t, r.OK)

	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "completed", updated.Status)
	core.AssertEqual(t, 0, updated.PID)

	output := fs.Read(core.JoinPath(metaDir, "agent-codex.log"))
	core.RequireTrue(t, output.OK)
	core.AssertEqual(t, "monitor output", output.Value.(string))
}

func TestDispatch_AgentCompletionMonitor_Bad_Case(t *testing.T) {
	s := newPrepWithProcess()
	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        "codex",
		workspaceDir: t.TempDir(),
	}

	r := monitor.run(context.Background(), core.NewOptions())
	core.AssertFalse(t, r.OK)
	core.AssertError(t, r.Value.(error))
	core.AssertContains(t, r.Value.(error).Error(), "process is required")
}

func TestDispatch_AgentCompletionMonitor_Ugly_Case(t *testing.T) {
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
	core.AssertTrue(t, r.OK)

	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "blocked", updated.Status)
	core.AssertEqual(t, "Need credentials", updated.Question)
	core.AssertFalse(t, fs.Exists(core.JoinPath(metaDir, "agent-codex.log")))
}

// --- onAgentComplete ---

func TestDispatch_OnAgentComplete_Good_Case(t *testing.T) {
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
	core.AssertEqual(t, "completed", updated.Status)
	core.AssertEqual(t, 0, updated.PID)

	r := fs.Read(outputFile)
	core.AssertTrue(t, r.OK)
	core.AssertEqual(t, "test output", r.Value.(string))
}

func TestDispatch_OnAgentComplete_Bad_Case(t *testing.T) {
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
	core.AssertEqual(t, "failed", updated.Status)
	core.AssertContains(t, updated.Question, "code 1")
}

func TestDispatch_OnAgentComplete_Ugly_Case(t *testing.T) {
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
	core.AssertEqual(t, "blocked", updated.Status)
	core.AssertEqual(t, "Need credentials", updated.Question)

	// Empty output should NOT create log file
	core.AssertFalse(t, fs.Exists(core.JoinPath(metaDir, "agent-codex.log")))
}

func TestDispatch_Run_Bad_Timeout(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-timeout")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	core.RequireTrue(t, fs.EnsureDir(repoDir).OK)
	core.RequireTrue(t, fs.EnsureDir(metaDir).OK)

	st := &WorkspaceStatus{
		Status:    "running",
		Agent:     "codex",
		Repo:      "go-io",
		StartedAt: time.Now(),
	}
	core.RequireNoError(t, writeStatus(wsDir, st))

	processResult := testCore.Service("process")
	core.RequireTrue(t, processResult.OK)
	procSvc, ok := processResult.Value.(*process.Service)
	core.RequireTrue(t, ok)

	timeout := 100 * time.Millisecond
	opts := dispatchRunOptions("sleep", []string{"60"}, repoDir, timeout)
	core.AssertEqual(t, timeout, opts.Timeout)
	core.AssertEqual(t, dispatchGracePeriod, opts.GracePeriod)
	core.AssertTrue(t, opts.KillGroup)
	core.AssertTrue(t, opts.Detach)

	proc, err := procSvc.StartWithOptions(context.Background(), opts)
	core.RequireNoError(t, err)
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
	core.AssertTrue(t, r.OK)

	info := proc.Info()
	core.AssertEqual(t, process.StatusKilled, info.Status)

	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "failed", updated.Status)
	core.AssertEqual(t, dispatchTimeoutReason(timeout), updated.Question)
	core.AssertEqual(t, 0, updated.PID)

	registryResult := s.workspaces.Get(WorkspaceName(wsDir))
	core.RequireTrue(t, registryResult.OK)
	registryStatus, ok := registryResult.Value.(*WorkspaceStatus)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "failed", registryStatus.Status)
	core.AssertEqual(t, dispatchTimeoutReason(timeout), registryStatus.Question)

	core.AssertFalse(t, fs.Exists(workspaceTimeoutPath(wsDir)))
}

// --- runQA ---

func TestDispatch_RunQA_Good_Case(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main() {}\n")

	s := newPrepWithProcess()
	core.AssertTrue(t, s.runQA(wsDir))
}

func TestDispatch_RunQA_Bad_Case(t *testing.T) {
	wsDir := t.TempDir()
	repoDir := core.JoinPath(wsDir, "repo")
	fs.EnsureDir(repoDir)

	// Broken Go code
	fs.Write(core.JoinPath(repoDir, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir, "main.go"), "package main\nfunc main( {\n}\n")

	s := newPrepWithProcess()
	core.AssertFalse(t, s.runQA(wsDir))

	// PHP project — composer not available
	wsDir2 := t.TempDir()
	repoDir2 := core.JoinPath(wsDir2, "repo")
	fs.EnsureDir(repoDir2)
	fs.Write(core.JoinPath(repoDir2, "composer.json"), `{"name":"test"}`)

	core.AssertFalse(t, s.runQA(wsDir2))
}

func TestDispatch_RunQA_Ugly_Case(t *testing.T) {
	// Unknown language — passes QA (no checks)
	wsDir := t.TempDir()
	fs.EnsureDir(core.JoinPath(wsDir, "repo"))

	s := newPrepWithProcess()
	core.AssertTrue(t, s.runQA(wsDir))

	// Go vet failure (compiles but bad printf)
	wsDir2 := t.TempDir()
	repoDir2 := core.JoinPath(wsDir2, "repo")
	fs.EnsureDir(repoDir2)
	fs.Write(core.JoinPath(repoDir2, "go.mod"), "module testmod\n\ngo 1.22\n")
	fs.Write(core.JoinPath(repoDir2, "main.go"), "package main\nimport \"fmt\"\nfunc main() { fmt.Printf(\"%d\", \"x\") }\n")
	core.AssertFalse(t, s.runQA(wsDir2))

	// Node project — npm install likely fails
	wsDir3 := t.TempDir()
	repoDir3 := core.JoinPath(wsDir3, "repo")
	fs.EnsureDir(repoDir3)
	fs.Write(core.JoinPath(repoDir3, "package.json"), `{"name":"test","scripts":{"test":"echo ok"}}`)
	_ = s.runQA(wsDir3) // exercises the node path
}

// --- dispatch ---

func TestDispatch_Dispatch_Good_Case(t *testing.T) {
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
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "codex", out.Agent)
	core.AssertNotEmpty(t, out.Prompt)
}

func TestDispatch_Dispatch_Bad_Case(t *testing.T) {
	s := newPrepWithProcess()

	// No repo
	_, _, err := s.dispatch(context.Background(), nil, DispatchInput{Task: "do"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "repo is required")

	// No task
	_, _, err = s.dispatch(context.Background(), nil, DispatchInput{Repo: "go-io"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "task is required")
}

func TestDispatch_Dispatch_Ugly_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// Prep fails (no local clone)
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), codePath: t.TempDir(), backoff: make(map[string]time.Time), failCount: make(map[string]int)}
	_, _, err := s.dispatch(context.Background(), nil, DispatchInput{
		Repo: "nonexistent", Task: "do", Issue: 1,
	})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "prep workspace failed")
}

// --- workspaceDir ---

func TestDispatch_WorkspaceDir_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	dir, err := workspaceDir("core", "go-io", PrepInput{Issue: 42})
	core.RequireNoError(t, err)
	core.AssertContains(t, dir, "task-42")

	dir2, _ := workspaceDir("core", "go-io", PrepInput{PR: 7})
	core.AssertContains(t, dir2, "pr-7")

	dir3, _ := workspaceDir("core", "go-io", PrepInput{Branch: "feat/new"})
	core.AssertContains(t, dir3, "feat/new")

	dir4, _ := workspaceDir("core", "go-io", PrepInput{Tag: "v1.0.0"})
	core.AssertContains(t, dir4, "v1.0.0")
}

func TestDispatch_WorkspaceDir_Bad_Case(t *testing.T) {
	_, err := workspaceDir("core", "go-io", PrepInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "one of issue, pr, branch, or tag")
}

func TestDispatch_WorkspaceDir_Ugly_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	// PR takes precedence when multiple set (first match)
	dir, err := workspaceDir("core", "go-io", PrepInput{PR: 3, Issue: 5})
	core.RequireNoError(t, err)
	core.AssertContains(t, dir, "pr-3")
}

// --- containerCommand ---

func TestDispatch_ContainerCommand_Bad_Case(t *testing.T) {
	t.Setenv("AGENT_DOCKER_IMAGE", "")
	t.Setenv("DIR_HOME", "/home/dev")

	// Empty command string — docker still runs, just with no command after image
	cmd, args := containerCommand("", []string{}, "/ws", "/ws/.meta")
	core.AssertEqual(t, "docker", cmd)
	core.AssertContains(t, args, "run")
	// The image should still be present in args
	core.AssertContains(t, args, defaultDockerImage)
	core.AssertContains(t, args, "/ws:/workspace")
	core.AssertContains(t, args, "/workspace/repo")
}

// --- canDispatchAgent ---
// Good: tested in queue_test.go
// Bad: tested in queue_test.go
// Ugly: see queue_extra_test.go

// --- agentCommand AX-10 ---

func TestDispatch_agentCommand_Good_Case(t *testing.T) {
	command, args, err := agentCommand("codex:gpt-5.4-mini", "Implement AX-10 unit tests for Mantis #169")
	core.RequireNoError(t, err)
	core.AssertEqual(t, "codex", command)
	core.AssertEqual(t, []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"-o", "../.meta/agent-codex.log",
		"--model", "gpt-5.4-mini",
		"Implement AX-10 unit tests for Mantis #169",
	}, args)
}

func TestDispatch_agentCommand_Bad_Case(t *testing.T) {
	command, args, err := agentCommand("mantis", "Investigate a failing dispatch")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unknown agent: mantis")
	core.AssertEmpty(t, command)
	core.AssertNil(t, args)
}

func TestDispatch_agentCommand_Ugly_Case(t *testing.T) {
	command, args, err := agentCommand("", "Investigate a failing dispatch")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unknown agent")
	core.AssertEmpty(t, command)
	core.AssertNil(t, args)
}
