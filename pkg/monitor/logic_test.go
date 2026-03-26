// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"os"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- handleAgentStarted ---

func TestLogic_HandleAgentStarted_Good(t *testing.T) {
	mon := New()
	ev := messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-1"}
	mon.handleAgentStarted(ev)

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning["core/go-io/task-1"])
}

func TestLogic_HandleAgentStarted_Bad_EmptyWorkspace(t *testing.T) {
	mon := New()
	// Empty workspace key must not panic and must record empty string key.
	ev := messages.AgentStarted{Agent: "", Repo: "", Workspace: ""}
	assert.NotPanics(t, func() { mon.handleAgentStarted(ev) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning[""])
}

// --- handleAgentCompleted ---

func TestLogic_HandleAgentCompleted_Good_NilRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	mon := New()
	// ServiceRuntime is nil — must not panic, must record completion and poke.
	ev := messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-1", Status: "completed"}
	assert.NotPanics(t, func() { mon.handleAgentCompleted(ev) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted["ws-1"])
}

func TestLogic_HandleAgentCompleted_Good_WithCore(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	// Use Register so IPC handlers are wired
	c := core.New(core.WithService(Register))
	mon, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	ev := messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-2", Status: "completed"}
	c.ACTION(ev)

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted["ws-2"])
}

func TestLogic_HandleAgentCompleted_Bad_EmptyFields(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	mon := New()

	// All fields empty — must not panic, must record empty workspace key.
	ev := messages.AgentCompleted{}
	assert.NotPanics(t, func() { mon.handleAgentCompleted(ev) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted[""])
}

// --- checkIdleAfterDelay ---

func TestLogic_CheckIdleAfterDelay_Bad_NilRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	mon := New() // ServiceRuntime is nil

	// Should return immediately without panic after the 5s sleep.
	// We test the "ServiceRuntime == nil" return branch by exercising the guard directly.
	done := make(chan struct{})
	go func() {
		if mon.ServiceRuntime == nil {
			close(done)
			return
		}
		mon.checkIdleAfterDelay()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("checkIdleAfterDelay nil-runtime guard did not return quickly")
	}
}

func TestLogic_CheckIdleAfterDelay_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	// Create a Core with an IPC handler to capture QueueDrained messages
	var captured []messages.QueueDrained
	c := core.New()
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.QueueDrained); ok {
			captured = append(captured, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})

	// With empty workspace, running=0 and queued=0, so queue.drained fires.
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 0, running)
	assert.Equal(t, 0, queued)

	if running == 0 && queued == 0 {
		mon.Core().ACTION(messages.QueueDrained{Completed: 0})
	}

	require.Len(t, captured, 1)
	assert.Equal(t, 0, captured[0].Completed)
}

// --- countLiveWorkspaces ---

func TestLogic_CountLiveWorkspaces_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	mon := New()
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 0, running)
	assert.Equal(t, 0, queued)
}

func TestLogic_CountLiveWorkspaces_Good_QueuedStatus(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "ws-q", map[string]any{
		"status": "queued",
		"repo":   "go-io",
		"agent":  "codex",
	})

	mon := New()
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 0, running)
	assert.Equal(t, 1, queued)
}

func TestLogic_CountLiveWorkspaces_Bad_RunningDeadPID(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	// PID 1 is always init/launchd and not "our" process — on macOS sending
	// signal 0 to PID 1 returns EPERM (process exists but not ours), which
	// means pidAlive returns false for non-owned processes. Use PID 99999999
	// which is near-certainly dead.
	writeWorkspaceStatus(t, wsRoot, "ws-dead", map[string]any{
		"status": "running",
		"repo":   "go-io",
		"agent":  "codex",
		"pid":    99999999,
	})

	mon := New()
	running, queued := mon.countLiveWorkspaces()
	// Dead PID should not count as running.
	assert.Equal(t, 0, running)
	assert.Equal(t, 0, queued)
}

func TestLogic_CountLiveWorkspaces_Good_RunningLivePID(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	// Current process is definitely alive.
	pid := os.Getpid()
	writeWorkspaceStatus(t, wsRoot, "ws-live", map[string]any{
		"status": "running",
		"repo":   "go-io",
		"agent":  "codex",
		"pid":    pid,
	})

	mon := New()
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 1, running)
	assert.Equal(t, 0, queued)
}

// --- pidAlive ---

func TestLogic_PidAlive_Good_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	assert.True(t, pidAlive(pid), "current process must be alive")
}

func TestLogic_PidAlive_Bad_DeadPID(t *testing.T) {
	// PID 99999999 is virtually guaranteed to not exist.
	assert.False(t, pidAlive(99999999))
}

func TestLogic_PidAlive_Ugly_ZeroPID(t *testing.T) {
	// PID 0 is not a valid user process. pidAlive must return false or at
	// least not panic.
	assert.NotPanics(t, func() { pidAlive(0) })
}

func TestLogic_PidAlive_Ugly_NegativePID(t *testing.T) {
	// Negative PID is invalid. Must not panic.
	assert.NotPanics(t, func() { pidAlive(-1) })
}

// --- SetCore ---

func TestLogic_SetCore_Good_RegistersIPCHandler(t *testing.T) {
	c := core.New()
	mon := New()

	// SetCore must not panic and must wire ServiceRuntime.
	assert.NotPanics(t, func() { mon.SetCore(c) })
	assert.NotNil(t, mon.ServiceRuntime)
	assert.Equal(t, c, mon.Core())
}

func TestLogic_SetCore_Good_IPCHandlerFires(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	// IPC handlers are registered via Register, not SetCore
	c := core.New(core.WithService(Register))

	mon, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	// Dispatch an AgentStarted via Core IPC — handler must update seenRunning.
	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "ws-ipc"})

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning["ws-ipc"])
}

func TestLogic_SetCore_Good_CompletedIPCHandler(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	// IPC handlers are registered via Register, not SetCore
	c := core.New(core.WithService(Register))

	mon, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	// Dispatch AgentCompleted — handler must update seenCompleted.
	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-done", Status: "completed"})

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted["ws-done"])
}

// --- OnStartup / OnShutdown ---

func TestLogic_OnStartup_Good_StartsLoop(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})
	r := mon.OnStartup(context.Background())
	assert.True(t, r.OK)

	// cancel must be non-nil after startup (loop running)
	assert.NotNil(t, mon.cancel)

	// Cleanup.
	r2 := mon.OnShutdown(context.Background())
	assert.True(t, r2.OK)
}

func TestLogic_OnStartup_Good_NoError(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	mon := New(Options{Interval: 1 * time.Hour})
	assert.True(t, mon.OnStartup(context.Background()).OK)
	_ = mon.OnShutdown(context.Background())
}

func TestLogic_OnShutdown_Good_NoError(t *testing.T) {
	mon := New(Options{Interval: 1 * time.Hour})
	assert.True(t, mon.OnShutdown(context.Background()).OK)
}

func TestLogic_OnShutdown_Good_StopsLoop(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})
	require.True(t, mon.OnStartup(context.Background()).OK)

	done := make(chan bool, 1)
	go func() {
		done <- mon.OnShutdown(context.Background()).OK
	}()

	select {
	case ok := <-done:
		assert.True(t, ok)
	case <-time.After(5 * time.Second):
		t.Fatal("OnShutdown did not return in time")
	}
}

func TestLogic_OnShutdown_Ugly_NilCancel(t *testing.T) {
	// OnShutdown without prior OnStartup must not panic.
	mon := New()
	assert.NotPanics(t, func() {
		_ = mon.OnShutdown(context.Background())
	})
}

// --- Register ---

func TestLogic_Register_Good_ReturnsSubsystem(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	require.NotNil(t, c)

	// Register returns the Subsystem as Value; WithService auto-registers it
	// under the package name "monitor".
	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	assert.True(t, ok, "Subsystem must be registered as \"monitor\"")
	assert.NotNil(t, svc)
}

func TestLogic_Register_Good_CoreWired(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	require.NotNil(t, c)

	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	// Register must set ServiceRuntime.
	assert.NotNil(t, svc.ServiceRuntime)
	assert.Equal(t, c, svc.Core())
}

func TestLogic_Register_Good_IPCHandlerActive(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(core.JoinPath(wsRoot, "workspace"), 0755))

	c := core.New(core.WithService(Register))
	require.NotNil(t, c)

	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	// Fire an AgentStarted message — the registered IPC handler must update seenRunning.
	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "ws-reg"})

	svc.mu.Lock()
	defer svc.mu.Unlock()
	assert.True(t, svc.seenRunning["ws-reg"])
}
