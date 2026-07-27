// SPDX-License-Identifier: EUPL-1.2

// queue.go branch coverage the existing queue tests miss.
//
// Seams used here:
//   - agentic.ProcessRegister + c.ServiceStartup wires a live go-process
//     service, so c.Process().Start("sleep", "30") spawns a REAL detached
//     process whose PID reads back via proc.Info().PID. Tracking a workspace
//     with that PID makes countRunningByAgent/countRunningByModel reach the
//     "PID > 0 && ProcessAlive → count++" leg the dead-PID tests skip. The
//     process is killed via t.Cleanup. No package-level TestMain — a per-test
//     Core keeps the 155 process-free tests unperturbed (they rely on
//     dead-PID → count 0).
//   - LETHEAN_HOME roots agentic.AgentsConfigPath() ("<home>/conf/agents.yaml"),
//     so seeding that exact file drives loadAgentsConfig's read+parse path.
//     (The existing invalid-YAML test sets CORE_WORKSPACE, which never reaches
//     the config file — it lives under conf/, not workspace/.)

package runner

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/process"
)

// liveProcessCore builds a Core with a started go-process service so
// c.Process().Start spawns real processes. Returns the Core.
func liveProcessCore(t *testing.T) *core.Core {
	t.Helper()
	c := core.New(core.WithOption("name", "runner-live"), core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	return c
}

// startSleep spawns a real detached `sleep 30` via the Core's process service
// and returns it (killed on cleanup). Its PID reads back through the same
// service ProcessAlive consults.
func startSleep(t *testing.T, c *core.Core) *process.Process {
	t.Helper()
	r := c.Process().Start(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	core.RequireTrue(t, r.OK)
	proc, ok := r.Value.(*process.Process)
	core.RequireTrue(t, ok)
	t.Cleanup(func() { _ = proc.Kill() })
	return proc
}

// --- countRunningByAgent: live-process count++ leg (queue.go:159) ---

// TestQueue_CountRunningByAgent_Good_LiveProcessCounts — a running workspace
// whose PID belongs to a genuinely-alive process is counted. This reaches the
// "PID > 0 && ProcessAlive(...) → count++" branch the dead-PID tests cannot.
func TestQueue_CountRunningByAgent_Good_LiveProcessCounts(t *testing.T) {
	c := liveProcessCore(t)
	proc := startSleep(t, c)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.TrackWorkspace("core/go-io/live", &WorkspaceStatus{
		Status: "running", Agent: "codex", PID: proc.Info().PID,
	})
	// A dead-PID running workspace for the same agent must NOT add to the count.
	svc.TrackWorkspace("core/go-io/dead", &WorkspaceStatus{
		Status: "running", Agent: "codex", PID: 999999999,
	})

	core.AssertEqual(t, 1, svc.countRunningByAgent("codex"))
}

// --- countRunningByModel: live-process count++ leg (queue.go:180) ---

// TestQueue_CountRunningByModel_Good_LiveProcessCounts — same live-process
// leg, but for the exact-model variant counter. Only the matching model with a
// live PID counts.
func TestQueue_CountRunningByModel_Good_LiveProcessCounts(t *testing.T) {
	c := liveProcessCore(t)
	proc := startSleep(t, c)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.TrackWorkspace("core/go-io/live", &WorkspaceStatus{
		Status: "running", Agent: "codex:gpt-5.4", PID: proc.Info().PID,
	})
	// Different model variant, live or not, must not be counted.
	svc.TrackWorkspace("core/go-io/other", &WorkspaceStatus{
		Status: "running", Agent: "codex:gpt-5.3", PID: proc.Info().PID,
	})

	core.AssertEqual(t, 1, svc.countRunningByModel("codex:gpt-5.4"))
}

// --- canDispatchAgent: live total-limit deny (queue.go:126) ---

// TestQueue_CanDispatchAgent_Bad_LiveProcessAtTotalLimit — with a total limit
// of 1 and one genuinely-alive codex process tracked, canDispatchAgent denies
// with the "total 1/1" reason. Exercises the limit-reached return through a
// real process rather than a PID<0 sentinel.
func TestQueue_CanDispatchAgent_Bad_LiveProcessAtTotalLimit(t *testing.T) {
	c := liveProcessCore(t)
	proc := startSleep(t, c)
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"codex": {Total: 1},
	})

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	svc.TrackWorkspace("core/go-io/live", &WorkspaceStatus{
		Status: "running", Agent: "codex", PID: proc.Info().PID,
	})

	can, reason := svc.canDispatchAgent("codex")
	core.AssertFalse(t, can)
	core.AssertEqual(t, "total 1/1", reason)
}

// --- loadAgentsConfig: read+parse the real config path (queue.go:81-89) ---

// TestQueue_LoadAgentsConfig_Good_ReadsRealFile — a valid agents.yaml at the
// resolved AgentsConfigPath() is read and unmarshalled (the readResult.OK +
// successful-parse return), so the values come from the file, not the
// hard-coded defaults.
func TestQueue_LoadAgentsConfig_Good_ReadsRealFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("LETHEAN_HOME", home)
	core.RequireTrue(t, fs.Write(agentic.AgentsConfigPath(), `
version: 7
dispatch:
  default_agent: clotho
concurrency:
  codex:
    total: 9
`).OK)

	svc := New()
	cfg := svc.loadAgentsConfig()
	core.AssertEqual(t, 7, cfg.Version)
	core.AssertEqual(t, "clotho", cfg.Dispatch.DefaultAgent)
	core.AssertEqual(t, 9, cfg.Concurrency["codex"].Total)
}

// TestQueue_LoadAgentsConfig_Ugly_InvalidRealFile — an unparseable agents.yaml
// at the resolved path makes yaml.Unmarshal fail, so the loop `continue`s past
// the bad file and the hard-coded defaults are returned (claude/gemini block).
// This reaches the unmarshal-error continue at queue.go:86-87.
func TestQueue_LoadAgentsConfig_Ugly_InvalidRealFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("LETHEAN_HOME", home)
	core.RequireTrue(t, fs.Write(agentic.AgentsConfigPath(), "::: not yaml :::\n\t- broken").OK)

	svc := New()
	cfg := svc.loadAgentsConfig()
	// Defaults: the parse failed so we fell through to the built-in config.
	core.AssertEqual(t, "claude", cfg.Dispatch.DefaultAgent)
	core.AssertEqual(t, 1, cfg.Concurrency["claude"].Total)
	core.AssertEqual(t, 3, cfg.Concurrency["gemini"].Total)
}

// --- delayForAgent: full ResetUTC parse + window branches (queue.go:296-319) ---

// TestQueue_DelayForAgent_Good_SustainedOutsideBurstWindow — a valid "HH:MM"
// ResetUTC parses cleanly; with BurstWindow 0 the burst branch is skipped and
// the sustained delay is returned. Reaches the strconv.Atoi success legs for
// both hour and minute (queue.go:299-304).
func TestQueue_DelayForAgent_Good_SustainedOutsideBurstWindow(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-rate"))
	c.Config().Set("agents.rates", map[string]RateConfig{
		"codex": {ResetUTC: "06:30", SustainedDelay: 11, BurstWindow: 0, BurstDelay: 99},
	})
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	core.AssertEqual(t, 11*time.Second, svc.delayForAgent("codex:gpt-5.4"))
}

// TestQueue_DelayForAgent_Ugly_BurstWindowAlwaysHits — BurstWindow 24h spans
// the whole day, so hoursUntilReset (always < 24) falls inside the window and
// the burst delay is returned instead of the sustained one. Reaches the
// burst-window return (queue.go:315-317) deterministically regardless of clock.
func TestQueue_DelayForAgent_Ugly_BurstWindowAlwaysHits(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-rate"))
	c.Config().Set("agents.rates", map[string]RateConfig{
		"codex": {ResetUTC: "00:00", SustainedDelay: 5, BurstWindow: 24, BurstDelay: 42},
	})
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	core.AssertEqual(t, 42*time.Second, svc.delayForAgent("codex"))
}

// --- drainQueue: success-loop completed++ (queue.go:196-198) ---

// TestQueue_DrainQueue_Good_DispatchesAndCounts — a single queued workspace on
// disk plus a fake agentic spawner makes drainQueue dispatch it (drainOne
// returns true once, then false), so the completed counter increments to 1.
// Exercises the for-loop body the frozen/empty drainQueue tests skip.
func TestQueue_DrainQueue_Good_DispatchesAndCounts(t *testing.T) {
	_ = seedQueuedWorkspace(t, "codex", "drain me")
	spawn := &fakeSpawner{pid: 7777, ok: true}
	svc := coreRunner(t, spawn)
	svc.frozen = false

	completed := svc.drainQueue()
	core.AssertEqual(t, 1, completed)
	core.AssertEqual(t, 1, spawn.calls)
}

// --- drainOne: backoff-skip leg (queue.go:219-220) ---

// TestQueue_drainOne_Bad_BackoffSkips — a queued workspace whose agent pool is
// in backoff (future timestamp) is skipped without spawning; drainOne returns
// false. Reaches the `if until, ok := s.backoff[pool]; before → continue` leg.
func TestQueue_drainOne_Bad_BackoffSkips(t *testing.T) {
	_ = seedQueuedWorkspace(t, "codex", "backed off")
	spawn := &fakeSpawner{pid: 1, ok: true}
	svc := coreRunner(t, spawn)
	svc.backoff["codex"] = time.Now().Add(time.Hour)

	core.AssertFalse(t, svc.drainOne())
	core.AssertEqual(t, 0, spawn.calls)
}

// --- drainOne: pre-spawn delay sleep (queue.go:224-226) ---

// TestQueue_drainOne_Good_DelaySleepThenSpawns — a rate config with a 1s
// sustained delay (and a burst window so the value is deterministic) makes
// drainOne sleep before dispatching, then spawn. Reaches the `if delay > 0 {
// time.Sleep(delay) }` leg. Costs one real ~1s sleep — the minimum, since
// SustainedDelay is multiplied by time.Second and there is no injectable clock
// seam (and adding one would violate the no-product-seam rule).
func TestQueue_drainOne_Good_DelaySleepThenSpawns(t *testing.T) {
	wsDir := seedQueuedWorkspace(t, "codex", "delayed dispatch")
	spawn := &fakeSpawner{pid: 5151, ok: true}
	svc := coreRunner(t, spawn)
	svc.Core().Config().Set("agents.rates", map[string]RateConfig{
		"codex": {ResetUTC: "00:00", SustainedDelay: 1, BurstWindow: 24, BurstDelay: 1},
	})

	start := time.Now()
	core.AssertTrue(t, svc.drainOne())
	core.AssertGreaterOrEqual(t, int(time.Since(start)/time.Second), 1)
	core.AssertEqual(t, 1, spawn.calls)

	st := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "running", st.Status)
	core.AssertEqual(t, 5151, st.PID)
}
