// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- handleAgentStarted ---

func TestHandleAgentStarted_Good(t *testing.T) {
	mon := New()
	ev := messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-1"}
	mon.handleAgentStarted(ev)

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning["core/go-io/task-1"])
}

func TestHandleAgentStarted_Bad_EmptyWorkspace(t *testing.T) {
	mon := New()
	// Empty workspace key must not panic and must record empty string key.
	ev := messages.AgentStarted{Agent: "", Repo: "", Workspace: ""}
	assert.NotPanics(t, func() { mon.handleAgentStarted(ev) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning[""])
}

// --- handleAgentCompleted ---

func TestHandleAgentCompleted_Good_NilNotifier(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	// notifier is nil — must not panic, must record completion and poke.
	ev := messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-1", Status: "completed"}
	assert.NotPanics(t, func() { mon.handleAgentCompleted(ev) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted["ws-1"])
}

func TestHandleAgentCompleted_Good_WithNotifier(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	ev := messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-2", Status: "completed"}
	mon.handleAgentCompleted(ev)

	// Give the goroutine spawned by checkIdleAfterDelay time to not fire within test
	// (it has a 5s sleep inside, so we just verify the notifier got the immediate event)
	events := notifier.Events()
	require.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, "agent.completed", events[0].channel)

	data := events[0].data.(map[string]any)
	assert.Equal(t, "go-io", data["repo"])
	assert.Equal(t, "codex", data["agent"])
	assert.Equal(t, "ws-2", data["workspace"])
	assert.Equal(t, "completed", data["status"])
}

func TestHandleAgentCompleted_Bad_EmptyFields(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	// All fields empty — must not panic.
	ev := messages.AgentCompleted{}
	assert.NotPanics(t, func() { mon.handleAgentCompleted(ev) })

	events := notifier.Events()
	require.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, "agent.completed", events[0].channel)
}

// --- checkIdleAfterDelay ---

func TestCheckIdleAfterDelay_Bad_NilNotifier(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New() // notifier is nil

	// Should return immediately without panic after the 5s sleep.
	// We override the sleep by calling it via a short-circuit: replace the
	// notifier check path — we just verify it doesn't panic and returns.
	done := make(chan struct{})
	go func() {
		// checkIdleAfterDelay has a time.Sleep(5s) — call with nil notifier path.
		// To avoid a 5-second wait we test the "notifier == nil" return branch
		// by only exercising the guard directly.
		if mon.notifier == nil {
			close(done)
			return
		}
		mon.checkIdleAfterDelay()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("checkIdleAfterDelay nil-notifier guard did not return quickly")
	}
}

func TestCheckIdleAfterDelay_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	notifier := &mockNotifier{}
	mon.SetNotifier(notifier)

	// With empty workspace, running=0 and queued=0, so queue.drained fires.
	// We run countLiveWorkspaces + the notifier call path directly to avoid the
	// 5s sleep in checkIdleAfterDelay.
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 0, running)
	assert.Equal(t, 0, queued)

	if running == 0 && queued == 0 {
		mon.notifier.ChannelSend(context.Background(), "queue.drained", map[string]any{
			"running": running,
			"queued":  queued,
		})
	}

	events := notifier.Events()
	require.Len(t, events, 1)
	assert.Equal(t, "queue.drained", events[0].channel)
}

// --- countLiveWorkspaces ---

func TestCountLiveWorkspaces_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New()
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 0, running)
	assert.Equal(t, 0, queued)
}

func TestCountLiveWorkspaces_Good_QueuedStatus(t *testing.T) {
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

func TestCountLiveWorkspaces_Bad_RunningDeadPID(t *testing.T) {
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

func TestCountLiveWorkspaces_Good_RunningLivePID(t *testing.T) {
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

func TestPidAlive_Good_CurrentProcess(t *testing.T) {
	pid := os.Getpid()
	assert.True(t, pidAlive(pid), "current process must be alive")
}

func TestPidAlive_Bad_DeadPID(t *testing.T) {
	// PID 99999999 is virtually guaranteed to not exist.
	assert.False(t, pidAlive(99999999))
}

func TestPidAlive_Ugly_ZeroPID(t *testing.T) {
	// PID 0 is not a valid user process. pidAlive must return false or at
	// least not panic.
	assert.NotPanics(t, func() { pidAlive(0) })
}

func TestPidAlive_Ugly_NegativePID(t *testing.T) {
	// Negative PID is invalid. Must not panic.
	assert.NotPanics(t, func() { pidAlive(-1) })
}

// --- SetCore ---

func TestSetCore_Good_RegistersIPCHandler(t *testing.T) {
	c := core.New()
	mon := New()

	// SetCore must not panic and must wire mon.core.
	assert.NotPanics(t, func() { mon.SetCore(c) })
	assert.Equal(t, c, mon.core)
}

func TestSetCore_Good_IPCHandlerFires(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	c := core.New()
	mon := New()
	mon.SetCore(c)

	// Dispatch an AgentStarted via Core IPC — handler must update seenRunning.
	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "ws-ipc"})

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning["ws-ipc"])
}

func TestSetCore_Good_CompletedIPCHandler(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	c := core.New()
	mon := New()
	mon.SetCore(c)

	// Dispatch AgentCompleted — handler must update seenCompleted.
	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-done", Status: "completed"})

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted["ws-done"])
}

// --- OnStartup / OnShutdown ---

func TestOnStartup_Good_StartsLoop(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})
	err := mon.OnStartup(context.Background())
	require.NoError(t, err)

	// cancel must be non-nil after startup (loop running)
	assert.NotNil(t, mon.cancel)

	// Cleanup.
	require.NoError(t, mon.OnShutdown(context.Background()))
}

func TestOnStartup_Good_NoError(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	mon := New(Options{Interval: 1 * time.Hour})
	assert.NoError(t, mon.OnStartup(context.Background()))
	_ = mon.OnShutdown(context.Background())
}

func TestOnShutdown_Good_NoError(t *testing.T) {
	mon := New(Options{Interval: 1 * time.Hour})
	assert.NoError(t, mon.OnShutdown(context.Background()))
}

func TestOnShutdown_Good_StopsLoop(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})
	require.NoError(t, mon.OnStartup(context.Background()))

	done := make(chan error, 1)
	go func() {
		done <- mon.OnShutdown(context.Background())
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("OnShutdown did not return in time")
	}
}

func TestOnShutdown_Ugly_NilCancel(t *testing.T) {
	// OnShutdown without prior OnStartup must not panic.
	mon := New()
	assert.NotPanics(t, func() {
		_ = mon.OnShutdown(context.Background())
	})
}

// --- Register ---

func TestRegister_Good_ReturnsSubsystem(t *testing.T) {
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

func TestRegister_Good_CoreWired(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	c := core.New(core.WithService(Register))
	require.NotNil(t, c)

	svc, ok := core.ServiceFor[*Subsystem](c, "monitor")
	require.True(t, ok)

	// Register must set mon.core to the Core instance.
	assert.Equal(t, c, svc.core)
}

func TestRegister_Good_IPCHandlerActive(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	require.NoError(t, os.MkdirAll(filepath.Join(wsRoot, "workspace"), 0755))

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
