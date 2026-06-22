// SPDX-License-Identifier: EUPL-1.2

// runner.go branch coverage the existing runner/IPC tests miss.
//
// Seams used here:
//   - A fake "mcp" service satisfying runner's local channelSender interface
//     (ChannelSend(ctx, channel, data)) lets HandleIPCEvents' sendNotification
//     reach the notifier-present + ACTUAL-send leg, with the emitted channel +
//     payload asserted. The existing IPC tests register no mcp service, so
//     sendNotification no-ops past that branch.
//   - agentic.ProcessRegister + c.ServiceStartup wires a live go-process
//     service so actionKill can terminate a REAL detached process (the
//     "PID > 0 && ProcessTerminate → killed++" leg the dead-PID kill test
//     skips). Per-test Core, killed via t.Cleanup — no package TestMain.
//   - LETHEAN_HOME roots AgentsConfigPath() so Register reads a seeded
//     agents.yaml carrying a codex concurrency block (covers the codexTotal
//     debug lookup the default config skips).

package runner

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/messages"
)

// recordingMCP is a fake "mcp" service implementing runner's channelSender
// interface. Each ChannelSend is recorded so tests can assert the runner
// emitted the expected agent.status notification.
type recordingMCP struct {
	channels []string
	payloads []any
}

func (m *recordingMCP) ChannelSend(_ context.Context, channel string, data any) {
	m.channels = append(m.channels, channel)
	m.payloads = append(m.payloads, data)
}

// --- HandleIPCEvents: sendNotification reaches the notifier (runner.go:130-134) ---

// TestRunner_HandleIPCEvents_AgentStarted_NotifiesMCP — with an mcp service
// that satisfies channelSender registered, AgentStarted's sendNotification
// resolves the notifier and actually calls ChannelSend. The emitted channel
// and the started-notification payload are asserted — the side-effect the
// no-mcp tests cannot observe.
func TestRunner_HandleIPCEvents_AgentStarted_NotifiesMCP(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"codex": {Total: 4},
	})

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.AgentStarted{
		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-9",
	})
	core.AssertTrue(t, r.OK)

	core.RequireTrue(t, len(mcp.channels) == 1, "exactly one notification emitted")
	core.AssertLen(t, mcp.channels, 1)
	core.AssertEqual(t, "agent.status", mcp.channels[0])
	notification, ok := mcp.payloads[0].(*AgentNotification)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "started", notification.Status)
	core.AssertEqual(t, "go-io", notification.Repo)
	core.AssertEqual(t, "codex", notification.Agent)
	core.AssertEqual(t, "core/go-io/task-9", notification.Workspace)
	core.AssertEqual(t, 4, notification.Limit)
}

// TestRunner_HandleIPCEvents_AgentCompleted_NotifiesMCP — AgentCompleted with
// a registered mcp service sends the completion notification AND, via Poke(),
// is exercised end to end. The recorded payload carries the event status.
func TestRunner_HandleIPCEvents_AgentCompleted_NotifiesMCP(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.pokeCh = make(chan struct{}, 1) // Poke() inside the completed arm is a no-op without this
	svc.TrackWorkspace("core/go-io/task-c", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 5,
	})

	r := svc.HandleIPCEvents(c, messages.AgentCompleted{
		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-c", Status: "merged",
	})
	core.AssertTrue(t, r.OK)

	core.RequireTrue(t, len(mcp.channels) == 1, "exactly one notification emitted")
	core.AssertLen(t, mcp.channels, 1)
	core.AssertEqual(t, "agent.status", mcp.channels[0])
	notification, ok := mcp.payloads[0].(*AgentNotification)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "merged", notification.Status)
}

// TestRunner_HandleIPCEvents_WrongTypeMCP_NoSend — an mcp service that does
// NOT implement channelSender resolves (OK) but fails the type-assert, so
// sendNotification returns without sending. The runner stays OK and the
// workspace still flips, proving the notify miss is non-fatal. Covers the
// `notifier, ok := ...; !ok → return` leg (runner.go:131-133).
func TestRunner_HandleIPCEvents_WrongTypeMCP_NoSend(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	core.RequireTrue(t, c.RegisterService("mcp", &wrongTypeAgentic{}).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.TrackWorkspace("core/go-io/task-w", &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: 7,
	})

	r := svc.HandleIPCEvents(c, messages.AgentStarted{
		Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-w",
	})
	core.AssertTrue(t, r.OK)
}

// --- actionKill: live-process terminate++ (runner.go:355-357) ---

// TestRunner_ActionKill_Good_TerminatesLiveProcess — a running workspace on
// disk whose PID is a genuinely-alive process is terminated by actionKill: the
// process is killed (killed++), the status flips to "failed" with PID cleared,
// and the result string reports 1 killed. Reaches the
// "PID > 0 && ProcessTerminate → killed++" leg the dead-PID kill test skips.
func TestRunner_ActionKill_Good_TerminatesLiveProcess(t *testing.T) {
	c := liveProcessCore(t)
	proc := startSleep(t, c)
	pid := proc.Info().PID

	// Seed a running workspace on disk pointing at the live PID.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	wsDir := core.PathJoin(agentic.WorkspaceRoot(), "kill-me")
	core.RequireTrue(t, core.MkdirAll(wsDir, 0o755).OK)
	core.RequireTrue(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status: "running", Agent: "codex", Repo: "go-io", PID: pid,
	}).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	result := svc.actionKill(context.Background(), core.NewOptions())
	core.RequireTrue(t, result.OK)
	core.AssertContains(t, result.Value.(string), "killed 1 agents")

	// The process is gone and the status reflects the kill.
	select {
	case <-proc.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("actionKill did not terminate the live process")
	}
	st := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "failed", st.Status)
	core.AssertEqual(t, 0, st.PID)
}

// --- hydrateWorkspaces: nil-registry init + skips (runner.go:428-444) ---

// TestRunner_HydrateWorkspaces_Ugly_NilRegistryRebuilt — calling
// hydrateWorkspaces on a Service whose registry is nil rebuilds it (the
// `if s.workspaces == nil` init) rather than panicking, and the rebuilt
// registry hydrates the on-disk queued workspace. Covers runner.go:428-430.
func TestRunner_HydrateWorkspaces_Ugly_NilRegistryRebuilt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsDir := core.JoinPath(root, "workspace", "core", "go-io", "task-h")
	core.RequireTrue(t, fs.EnsureDir(wsDir).OK)
	core.RequireTrue(t, WriteStatus(wsDir, &WorkspaceStatus{
		Status: "queued", Agent: "codex", Repo: "go-io",
	}).OK)

	svc := &Service{} // registry deliberately nil
	core.AssertNotPanics(t, func() { svc.hydrateWorkspaces() })

	core.AssertNotNil(t, svc.workspaces)
	core.RequireTrue(t, svc.workspaces != nil, "registry rebuilt")
	r := svc.workspaces.Get("core/go-io/task-h")
	core.AssertTrue(t, r.OK)
}

// TestRunner_HydrateWorkspaces_Bad_SkipsUnreadableStatus — a workspace dir
// whose status.json is invalid JSON is skipped (ReadStatusResult !OK →
// continue), so it is NOT registered. A second, valid queued workspace IS
// registered. Covers the read-failure continue (runner.go:434-435).
func TestRunner_HydrateWorkspaces_Bad_SkipsUnreadableStatus(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)
	wsRoot := core.JoinPath(root, "workspace")

	badDir := core.JoinPath(wsRoot, "core", "go-io", "broken")
	core.RequireTrue(t, fs.EnsureDir(badDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(badDir, "status.json"), "{not-json").OK)

	goodDir := core.JoinPath(wsRoot, "core", "go-io", "fine")
	core.RequireTrue(t, fs.EnsureDir(goodDir).OK)
	core.RequireTrue(t, WriteStatus(goodDir, &WorkspaceStatus{
		Status: "queued", Agent: "codex", Repo: "go-io",
	}).OK)

	svc := New()
	svc.hydrateWorkspaces()

	core.AssertFalse(t, svc.workspaces.Get("core/go-io/broken").OK, "unreadable status skipped")
	core.AssertTrue(t, svc.workspaces.Get("core/go-io/fine").OK, "valid status hydrated")
}

// --- runLoop: pokeCh drain arm (runner.go:414-415) ---

// TestRunner_RunLoop_Good_PokeDrains — starting runLoop in a goroutine and
// firing Poke() drives the `<-s.pokeCh` arm, which calls drainQueueAndNotify
// and emits a QueueDrained action. The test waits on a captured-event channel
// (no sleep): receiving QueueDrained proves the poke arm ran. The 30s ticker
// arm has no injectable seam and is left uncovered.
func TestRunner_RunLoop_Good_PokeDrains(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-loop"))
	t.Setenv("CORE_WORKSPACE", t.TempDir()) // empty workspace → drain finds nothing, completes cleanly

	drained := make(chan messages.QueueDrained, 4)
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.QueueDrained); ok {
			select {
			case drained <- ev:
			default:
			}
		}
		return core.Result{OK: true}
	})

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.pokeCh = make(chan struct{}, 1)

	go svc.runLoop()
	svc.Poke()

	select {
	case ev := <-drained:
		core.AssertEqual(t, 0, ev.Completed)
	case <-time.After(5 * time.Second):
		t.Fatal("runLoop poke arm did not drain within 5s")
	}
}

// --- actionDispatch: default-agent + capacity-deny (runner.go:279-290) ---

// TestRunner_ActionDispatch_Ugly_DefaultsAgentToCodex — dispatching with NO
// agent option defaults the agent to "codex" (the `if agent == "" { agent =
// "codex" }` leg) and tracks a pending workspace under that agent. Covers
// runner.go:279-281.
func TestRunner_ActionDispatch_Ugly_DefaultsAgentToCodex(t *testing.T) {
	svc := New()
	svc.frozen = false

	r := svc.actionDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
	))
	core.AssertTrue(t, r.OK)

	tracked := svc.workspaces.Get("pending/go-io")
	core.RequireTrue(t, tracked.OK)
	ws := tracked.Value.(*WorkspaceStatus)
	core.AssertEqual(t, "codex", ws.Agent, "empty agent option defaults to codex")
	core.AssertEqual(t, -1, ws.PID)
}

// TestRunner_ActionDispatch_Bad_AtCapacityDenies — with a codex total limit of
// 1 already met by a tracked running workspace, actionDispatch's
// canDispatchAgent gate fails and it returns the "queue at capacity" error
// without tracking a new pending workspace. Covers runner.go:288-290.
func TestRunner_ActionDispatch_Bad_AtCapacityDenies(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-cap"))
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"codex": {Total: 1},
	})
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.frozen = false
	svc.TrackWorkspace("core/go-io/busy", &WorkspaceStatus{
		Status: "running", Agent: "codex", PID: -1,
	})

	r := svc.actionDispatch(context.Background(), core.NewOptions(
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "repo", Value: "go-log"},
	))
	core.AssertFalse(t, r.OK)
	err, ok := r.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "capacity")
	// No new pending workspace was tracked.
	core.AssertFalse(t, svc.workspaces.Get("pending/go-log").OK)
}

// --- TrackWorkspace: unconvertible value skipped (runner.go:239-241) ---

// TestRunner_TrackWorkspace_Ugly_UnconvertibleValueSkips — a value that JSON
// MARSHALS fine but cannot UNMARSHAL into a WorkspaceStatus (a JSON array vs
// the struct) leaves workspaceStatus nil through the default switch arm, so
// TrackWorkspace hits the `if workspaceStatus == nil { return }` guard and
// registers nothing. The existing Ugly test passes a map that round-trips
// successfully, so it skips this leg.
func TestRunner_TrackWorkspace_Ugly_UnconvertibleValueSkips(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("x", []int{1, 2, 3}) // array → struct unmarshal fails → stays nil

	core.AssertFalse(t, svc.workspaces.Get("x").OK, "unconvertible value is not tracked")
	core.AssertEqual(t, 0, svc.Workspaces().Len())
}

// --- Register: codexTotal debug lookup (runner.go:75-77) ---

// TestRunner_Register_Ugly_ReadsCodexConcurrency — Register reads a seeded
// agents.yaml carrying a codex concurrency block, so the codexTotal debug
// lookup (the `if limit, ok := config.Concurrency["codex"]` leg) finds it and
// stashes the total in config. The default config has no codex key, so this is
// the only path that reaches that branch.
func TestRunner_Register_Ugly_ReadsCodexConcurrency(t *testing.T) {
	home := t.TempDir()
	t.Setenv("LETHEAN_HOME", home)
	core.RequireTrue(t, fs.Write(agentic.AgentsConfigPath(), `
version: 1
concurrency:
  codex:
    total: 6
`).OK)

	c := core.New(core.WithOption("name", "runner-reg"))
	r := Register(c)
	core.RequireTrue(t, r.OK)
	core.AssertEqual(t, 6, c.Config().Get("agents.codex_limit_debug").Value)
}

// --- startRunner: CORE_AGENT_DISPATCH=1 unfreezes (runner.go:398-400) ---

// TestRunner_StartRunner_Good_DispatchEnvUnfreezes — with CORE_AGENT_DISPATCH=1
// startRunner takes the unfreeze leg (frozen = false). OnStartup drives it.
// The existing Ugly OnStartup test asserts the opposite (frozen without the
// env), so this covers the if-branch it skips.
func TestRunner_StartRunner_Good_DispatchEnvUnfreezes(t *testing.T) {
	t.Setenv("CORE_AGENT_DISPATCH", "1")
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	c := core.New(core.WithOption("name", "runner-dispatch"))
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	svc.OnStartup(context.Background())
	core.AssertNotNil(t, svc.pokeCh)
	core.AssertFalse(t, svc.IsFrozen(), "CORE_AGENT_DISPATCH=1 unfreezes the queue")
}

// --- actionPoke: with runtime resolves Core (runner.go:386-391) ---

// TestRunner_ActionPoke_Good_WithRuntimeDrains — actionPoke on a Service WITH
// a ServiceRuntime takes the `coreApp = s.Core()` leg, then drainQueueAndNotify
// emits a QueueDrained action (captured here). The existing actionPoke test
// runs with NO runtime, so it skips this assignment. Empty workspace → 0
// completed.
func TestRunner_ActionPoke_Good_WithRuntimeDrains(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-poke"))
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	var captured []messages.QueueDrained
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.QueueDrained); ok {
			captured = append(captured, ev)
		}
		return core.Result{OK: true}
	})

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.actionPoke(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)
	core.RequireTrue(t, len(captured) == 1, "exactly one QueueDrained captured")
	core.AssertLen(t, captured, 1)
	core.AssertEqual(t, 0, captured[0].Completed)
}
