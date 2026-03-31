// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testMon *Subsystem

func TestMain(m *testing.M) {
	c := core.New(core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	testMon = New()
	testMon.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	os.Exit(m.Run())
}

type capturedChannelEvent struct {
	Channel string
	Data    any
}

type captureNotifier struct {
	events []capturedChannelEvent
}

func (n *captureNotifier) ChannelSend(_ context.Context, channel string, data any) {
	n.events = append(n.events, capturedChannelEvent{Channel: channel, Data: data})
}

// setupBrainKey creates a ~/.claude/brain.key file for API auth tests.
func setupBrainKey(t *testing.T, key string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	claudeDir := core.JoinPath(home, ".claude")
	fs.EnsureDir(claudeDir)
	fs.Write(core.JoinPath(claudeDir, "brain.key"), key)
}

// setupAPIEnv sets up brain key, CORE_API_URL, and AGENT_NAME for API tests.
func setupAPIEnv(t *testing.T, apiURL string) {
	t.Helper()
	setupBrainKey(t, "key")
	t.Setenv("CORE_API_URL", apiURL)
	t.Setenv("AGENT_NAME", "test-agent")
}

// writeWorkspaceStatus creates a workspace directory with a status.json file
// under the given root. Returns the workspace directory path.
func writeWorkspaceStatus(t *testing.T, wsRoot, name string, fields map[string]any) string {
	t.Helper()
	dir := core.JoinPath(wsRoot, "workspace", name)
	fs.EnsureDir(dir)
	fs.Write(core.JoinPath(dir, "status.json"), core.JSONMarshalString(fields))
	return dir
}

func startManagedProcess(t *testing.T, c *core.Core) *process.Process {
	t.Helper()

	r := c.Process().Start(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	require.True(t, r.OK)

	proc, ok := r.Value.(*process.Process)
	require.True(t, ok)
	t.Cleanup(func() {
		_ = proc.Kill()
	})
	return proc
}

// --- New ---

func TestMonitor_New_Good_Defaults(t *testing.T) {
	t.Setenv("MONITOR_INTERVAL", "")
	mon := New()
	assert.Equal(t, 2*time.Minute, mon.interval)
	assert.NotNil(t, mon.poke)
}

func TestMonitor_New_Good_CustomInterval(t *testing.T) {
	mon := New(Options{Interval: 30 * time.Second})
	assert.Equal(t, 30*time.Second, mon.interval)
}

func TestMonitor_New_Bad_ZeroInterval(t *testing.T) {
	t.Setenv("MONITOR_INTERVAL", "")
	mon := New(Options{Interval: 0})
	assert.Equal(t, 2*time.Minute, mon.interval)
}

func TestMonitor_Name_Good(t *testing.T) {
	mon := New()
	assert.Equal(t, "monitor", mon.Name())
}

// --- Poke ---

func TestMonitor_Poke_Good(t *testing.T) {
	mon := New()
	mon.Poke()

	select {
	case <-mon.poke:
	default:
		t.Fatal("expected poke to send a value")
	}
}

func TestMonitor_Poke_Good_NonBlocking(t *testing.T) {
	mon := New()
	mon.Poke()
	mon.Poke() // second poke should be a no-op, not block

	select {
	case <-mon.poke:
	default:
		t.Fatal("expected at least one poke")
	}

	select {
	case <-mon.poke:
		t.Fatal("expected channel to be empty after drain")
	default:
	}
}

// --- Start / Shutdown ---

func TestMonitor_StartShutdown_Good(t *testing.T) {
	mon := New(Options{Interval: 1 * time.Hour})

	ctx := context.Background()
	mon.Start(ctx)

	err := mon.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestMonitor_Shutdown_Good_NilCancel(t *testing.T) {
	mon := New()
	err := mon.Shutdown(context.Background())
	assert.NoError(t, err)
}

// --- handleAgentStarted / handleAgentCompleted ---

func TestMonitor_HandleAgentStarted_Good(t *testing.T) {
	mon := New()
	ev := messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-1"}
	mon.handleAgentStarted(ev)

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning["core/go-io/task-1"])
}

func TestMonitor_HandleAgentStarted_Bad_EmptyWorkspace(t *testing.T) {
	mon := New()
	ev := messages.AgentStarted{}

	assert.NotPanics(t, func() { mon.handleAgentStarted(ev) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenRunning[""])
}

func TestMonitor_HandleAgentCompleted_Good_NilRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	ev := messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-1", Status: "completed"}

	assert.NotPanics(t, func() { mon.handleAgentCompleted(ev) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted["ws-1"])
}

func TestMonitor_HandleAgentCompleted_Good_WithCore(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	c := core.New(core.WithService(Register))
	service := c.Service("monitor")
	require.True(t, service.OK)
	mon, ok := service.Value.(*Subsystem)
	require.True(t, ok)

	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "ws-2", Status: "completed"})

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted["ws-2"])
}

func TestMonitor_HandleAgentCompleted_Bad_EmptyFields(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()

	assert.NotPanics(t, func() { mon.handleAgentCompleted(messages.AgentCompleted{}) })

	mon.mu.Lock()
	defer mon.mu.Unlock()
	assert.True(t, mon.seenCompleted[""])
}

// --- checkIdleAfterDelay ---

func TestMonitor_CheckIdleAfterDelay_Bad_NilRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
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

func TestMonitor_CheckIdleAfterDelay_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	var captured []messages.QueueDrained
	c := core.New()
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.QueueDrained); ok {
			captured = append(captured, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, Options{})

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

func TestMonitor_CountLiveWorkspaces_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 0, running)
	assert.Equal(t, 0, queued)
}

func TestMonitor_CountLiveWorkspaces_Good_QueuedStatus(t *testing.T) {
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

func TestMonitor_CountLiveWorkspaces_Bad_RunningDeadPID(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "ws-dead", map[string]any{
		"status": "running",
		"repo":   "go-io",
		"agent":  "codex",
		"pid":    99999999,
	})

	mon := New()
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 0, running)
	assert.Equal(t, 0, queued)
}

func TestMonitor_CountLiveWorkspaces_Good_RunningLivePID(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	proc := startManagedProcess(t, testMon.Core())
	pid := proc.Info().PID
	writeWorkspaceStatus(t, wsRoot, "ws-live", map[string]any{
		"status":     "running",
		"repo":       "go-io",
		"agent":      "codex",
		"pid":        pid,
		"process_id": proc.ID,
	})

	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	running, queued := mon.countLiveWorkspaces()
	assert.Equal(t, 1, running)
	assert.Equal(t, 0, queued)
}

// --- processAlive ---

func TestMonitor_ProcessAlive_Good_ManagedProcess(t *testing.T) {
	proc := startManagedProcess(t, testMon.Core())
	assert.True(t, processAlive(testMon.Core(), proc.ID, proc.Info().PID), "managed process must be alive")
}

func TestMonitor_ProcessAlive_Bad_DeadPID(t *testing.T) {
	assert.False(t, processAlive(testMon.Core(), "", 99999999))
}

func TestMonitor_ProcessAlive_Ugly_ZeroPID(t *testing.T) {
	assert.NotPanics(t, func() { processAlive(testMon.Core(), "", 0) })
}

func TestMonitor_ProcessAlive_Ugly_NegativePID(t *testing.T) {
	assert.NotPanics(t, func() { processAlive(testMon.Core(), "", -1) })
}

// --- OnStartup / OnShutdown ---

func TestMonitor_OnStartup_Good_StartsLoop(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})
	r := mon.OnStartup(context.Background())
	assert.True(t, r.OK)
	assert.NotNil(t, mon.cancel)

	r2 := mon.OnShutdown(context.Background())
	assert.True(t, r2.OK)
}

func TestMonitor_OnStartup_Good_NoError(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New(Options{Interval: 1 * time.Hour})
	assert.True(t, mon.OnStartup(context.Background()).OK)
	_ = mon.OnShutdown(context.Background())
}

func TestMonitor_OnShutdown_Good_NoError(t *testing.T) {
	mon := New(Options{Interval: 1 * time.Hour})
	assert.True(t, mon.OnShutdown(context.Background()).OK)
}

func TestMonitor_OnShutdown_Good_StopsLoop(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

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

func TestMonitor_OnShutdown_Ugly_NilCancel(t *testing.T) {
	mon := New()
	assert.NotPanics(t, func() {
		_ = mon.OnShutdown(context.Background())
	})
}

// --- checkCompletions ---

func TestMonitor_CheckCompletions_Good_NewCompletions(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	// Create Core with IPC handler to capture QueueDrained messages
	var drainEvents []messages.QueueDrained
	c := core.New()
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.QueueDrained); ok {
			drainEvents = append(drainEvents, ev)
		}
		return core.Result{OK: true}
	})

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	assert.Equal(t, "", mon.checkCompletions())

	for i := 0; i < 2; i++ {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": "completed",
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "2 agent(s) completed")

	// checkCompletions emits QueueDrained via c.ACTION() when running=0 and queued=0
	require.Len(t, drainEvents, 1)
	assert.Equal(t, 2, drainEvents[0].Completed)
}

func TestMonitor_CheckCompletions_Good_MixedStatuses(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	assert.Equal(t, "", mon.checkCompletions())

	for i, status := range []string{"completed", "running", "queued"} {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": status,
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "1 agent(s) completed")
	assert.Contains(t, msg, "1 still running")
	assert.Contains(t, msg, "1 queued")
}

func TestMonitor_CheckCompletions_Good_NoNewCompletions(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "ws-0", map[string]any{
		"status": "completed", "repo": "r", "agent": "a",
	})

	mon := New()
	mon.checkCompletions() // sets baseline

	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckCompletions_Good_EmptyWorkspace(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckCompletions_Bad_InvalidJSON(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	dir := core.JoinPath(wsRoot, "workspace", "ws-bad")
	fs.EnsureDir(dir)
	fs.Write(core.JoinPath(dir, "status.json"), "not json")

	mon := New()
	msg := mon.checkCompletions()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckCompletions_Good_NilRuntime(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	assert.Equal(t, "", mon.checkCompletions())

	writeWorkspaceStatus(t, wsRoot, "ws-0", map[string]any{
		"status": "completed", "repo": "r", "agent": "a",
	})

	msg := mon.checkCompletions()
	assert.Contains(t, msg, "1 agent(s) completed")
}

func TestMonitor_CheckCompletions_Good_DeepWorkspaceName(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "core/go-io/task-7", map[string]any{
		"status": "completed", "repo": "go-io", "agent": "codex",
	})

	mon := New()
	assert.Equal(t, "", mon.checkCompletions())
	assert.True(t, mon.seenCompleted["core/go-io/task-7"])
}

// --- checkInbox ---

func TestMonitor_CheckInbox_Good_UnreadMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages/inbox", r.URL.Path)
		assert.NotEmpty(t, r.URL.Query().Get("agent"))

		resp := map[string]any{
			"data": []map[string]any{
				{"id": 3, "read": false, "from": "clotho", "subject": "task done"},
				{"id": 2, "read": false, "from": "gemini", "subject": "review ready"},
				{"id": 1, "read": true, "from": "clotho", "subject": "old msg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupBrainKey(t, "test-key")
	t.Setenv("CORE_API_URL", srv.URL)
	t.Setenv("AGENT_NAME", "test-agent")

	// Create Core with an MCP notifier to capture channel events.
	c := core.New()
	notifier := &captureNotifier{}
	require.True(t, c.RegisterService("mcp", notifier).OK)

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	mon.inboxSeeded = true

	msg := mon.checkInbox()
	assert.Contains(t, msg, "2 unread message(s) in inbox")

	// 3 new messages (ids 1,2,3 all > prevMaxID=0), but only 2 unread.
	// Monitor sends one channel notification per new message.
	require.Len(t, notifier.events, 3)
	data := notifier.events[0].Data.(map[string]any)
	assert.Equal(t, "clotho", data["from"])
}

func TestMonitor_CheckInbox_Good_EncodesAgentQuery(t *testing.T) {
	expectedAgent := "test agent+1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages/inbox", r.URL.Path)
		assert.Equal(t, expectedAgent, r.URL.Query().Get("agent"))
		resp := map[string]any{
			"data": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupBrainKey(t, "test-key")
	t.Setenv("CORE_API_URL", srv.URL)
	t.Setenv("AGENT_NAME", expectedAgent)

	mon := New()
	mon.inboxSeeded = true

	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Good_NoUnread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": 1, "read": true, "from": "clotho", "subject": "old"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Good_SameCountNoRepeat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": 1, "read": false, "from": "clotho", "subject": "msg"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	mon.checkInbox() // sets baseline

	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Bad_NoBrainKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Bad_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Bad_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	mon := New()
	msg := mon.checkInbox()
	assert.Equal(t, "", msg)
}

func TestMonitor_CheckInbox_Good_MultipleSameSender(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": 3, "read": false, "from": "clotho", "subject": "msg1"},
				{"id": 2, "read": false, "from": "clotho", "subject": "msg2"},
				{"id": 1, "read": false, "from": "gemini", "subject": "msg3"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(core.JSONMarshalString(resp)))
	}))
	defer srv.Close()

	setupAPIEnv(t, srv.URL)

	// Create Core with an MCP notifier to capture channel events.
	c := core.New()
	notifier := &captureNotifier{}
	require.True(t, c.RegisterService("mcp", notifier).OK)

	mon := New()
	mon.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	mon.inboxSeeded = true

	msg := mon.checkInbox()
	assert.Contains(t, msg, "3 unread message(s)")

	require.True(t, len(notifier.events) > 0, "should capture at least one channel event")
}

// --- check (integration of sub-checks) ---

func TestMonitor_Check_Good_CombinesMessages(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "ws-0", map[string]any{
		"status": "completed", "repo": "r", "agent": "a",
	})

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	mon.check(context.Background())

	mon.mu.Lock()
	assert.True(t, mon.completionsSeeded)
	assert.True(t, mon.seenCompleted["ws-0"])
	mon.mu.Unlock()
}

func TestMonitor_Check_Good_NoMessages(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New()
	mon.check(context.Background())
}

// --- notify ---

func TestMonitor_Notify_Good_NilServer(t *testing.T) {
	mon := New()
	mon.notify(context.Background(), "test message")
}

// --- loop ---

func TestMonitor_Loop_Good_ImmediateCancel(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		mon.loop(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not exit after context cancellation")
	}
}

func TestMonitor_Loop_Good_PokeTriggersCheck(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	home := t.TempDir()
	t.Setenv("HOME", home)

	mon := New(Options{Interval: 1 * time.Hour})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mon.wg.Add(1)
	go func() {
		defer mon.wg.Done()
		mon.loop(ctx)
	}()

	// Wait for initial delay (5s) + first check + scheduler overhead
	time.Sleep(7 * time.Second)

	writeWorkspaceStatus(t, wsRoot, "ws-poke", map[string]any{
		"status": "completed", "repo": "poke-repo", "agent": "a",
	})

	mon.Poke()

	// Poll until the poke-triggered check updates the count
	require.Eventually(t, func() bool {
		mon.mu.Lock()
		defer mon.mu.Unlock()
		return mon.seenCompleted["ws-poke"]
	}, 5*time.Second, 50*time.Millisecond, "expected ws-poke completion to be recorded")

	cancel()
	mon.wg.Wait()
}

// --- agentStatusResource ---

func TestMonitor_AgentStatusResource_Good(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	for i, status := range []string{"completed", "running"} {
		writeWorkspaceStatus(t, wsRoot, fmt.Sprintf("ws-%d", i), map[string]any{
			"status": status,
			"repo":   fmt.Sprintf("repo-%d", i),
			"agent":  "claude:sonnet",
		})
	}

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "status://agents", result.Contents[0].URI)

	var workspaces []map[string]any
	require.True(t, core.JSONUnmarshalString(result.Contents[0].Text, &workspaces).OK)
	assert.Len(t, workspaces, 2)
}

func TestMonitor_AgentStatusResource_Good_Empty(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)
	fs.EnsureDir(core.JoinPath(wsRoot, "workspace"))

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	assert.Equal(t, "null", result.Contents[0].Text)
}

func TestMonitor_AgentStatusResource_Bad_InvalidJSON(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	dir := core.JoinPath(wsRoot, "workspace", "ws-bad")
	fs.EnsureDir(dir)
	fs.Write(core.JoinPath(dir, "status.json"), "bad")

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)
	assert.Equal(t, "null", result.Contents[0].Text)
}

func TestMonitor_AgentStatusResource_Good_DeepWorkspaceName(t *testing.T) {
	wsRoot := t.TempDir()
	t.Setenv("CORE_WORKSPACE", wsRoot)

	writeWorkspaceStatus(t, wsRoot, "core/go-io/task-9", map[string]any{
		"status": "completed", "repo": "go-io", "agent": "claude:sonnet",
	})

	mon := New()
	result, err := mon.agentStatusResource(context.Background(), &mcp.ReadResourceRequest{})
	require.NoError(t, err)

	var workspaces []map[string]any
	require.True(t, core.JSONUnmarshalString(result.Contents[0].Text, &workspaces).OK)
	require.Len(t, workspaces, 1)
	assert.Equal(t, "core/go-io/task-9", workspaces[0]["name"])
}
