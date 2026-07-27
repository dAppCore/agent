// SPDX-License-Identifier: EUPL-1.2

// The last cleanly-reachable branch legs in queue.go + runner.go that the
// main coverage files leave open. Each targets a specific fall-through or
// guard the happy-path tests step over. The genuinely-defensive twins (the
// nil-error-wrapping returns in paths.go, the ProcessTerminate/WriteStatus
// failure warns, the 30s runLoop ticker arm) are left uncovered by design —
// they need an injectable seam the product code doesn't expose, and adding one
// for coverage alone is out of scope.

package runner

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
)

// --- canDispatchAgent: model-limit present but UNDER limit (queue.go:142) ---

// TestQueue_CanDispatchAgent_Good_ModelUnderLimitFallsThrough — a configured
// model limit where the running model count is below the limit falls through
// both the total gate and the model gate to the final `return true, ""`. The
// existing model tests all hit the deny; this reaches the allow fall-through.
func TestQueue_CanDispatchAgent_Good_ModelUnderLimitFallsThrough(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-cap"))
	c.Config().Set("agents.concurrency", map[string]ConcurrencyLimit{
		"codex": {Total: 5, Models: map[string]int{"gpt-5.4": 2}},
	})
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})
	// One running codex:gpt-5.4 (PID<0 counts) — total 1/5 and model 1/2, both
	// under limit, so dispatch is allowed.
	svc.TrackWorkspace("core/go-io/one", &WorkspaceStatus{
		Status: "running", Agent: "codex:gpt-5.4", PID: -1,
	})

	can, reason := svc.canDispatchAgent("codex:gpt-5.4")
	core.AssertTrue(t, can, "under both total and model limits → allowed")
	core.AssertEmpty(t, reason)
}

// --- delayForAgent: reset-today rollback leg (queue.go:309) ---

// TestQueue_DelayForAgent_Ugly_ResetRollback — a late ResetUTC ("23:59") means
// "now" is almost always before today's reset instant, so the
// `if now.Before(resetToday) { resetToday = resetToday.AddDate(0,0,-1) }`
// rollback fires. With BurstWindow 0 the sustained delay is returned. This
// drives the rollback arm deterministically except in the final minute of the
// UTC day; the assertion holds for the returned delay regardless of which arm
// computed nextReset.
func TestQueue_DelayForAgent_Ugly_ResetRollback(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-rate"))
	c.Config().Set("agents.rates", map[string]RateConfig{
		"codex": {ResetUTC: "23:59", SustainedDelay: 13, BurstWindow: 0, BurstDelay: 1},
	})
	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	core.AssertEqual(t, 13*core.Second, svc.delayForAgent("codex"))
}

// --- drainOne: nil ServiceRuntime bails before spawn (queue.go:235) ---

// TestQueue_drainOne_Bad_NilRuntimeBails — a queued workspace on disk passes
// the concurrency + backoff + delay gates (all of which guard on a nil
// ServiceRuntime and fall back to disk config), then hits the
// `if s.ServiceRuntime == nil { continue }` guard and returns false without
// spawning. No status flip.
func TestQueue_drainOne_Bad_NilRuntimeBails(t *testing.T) {
	wsDir := seedQueuedWorkspace(t, "codex", "no runtime")
	svc := New() // ServiceRuntime deliberately nil

	core.AssertFalse(t, svc.drainOne())

	st := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "queued", st.Status, "no runtime → no spawn, status untouched")
}

// --- drainOne: unreadable status skipped (queue.go:206) ---

// TestQueue_drainOne_Bad_SkipsUnreadableStatus — a workspace dir whose
// status.json is invalid JSON makes ReadStatusResult fail, so drainOne
// `continue`s past it and, finding nothing else queued, returns false.
// Reaches the read-failure continue.
func TestQueue_drainOne_Bad_SkipsUnreadableStatus(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	wsDir := core.PathJoin(agentic.WorkspaceRoot(), "broken")
	core.RequireTrue(t, core.MkdirAll(wsDir, 0o755).OK)
	core.RequireTrue(t, fs.Write(agentic.WorkspaceStatusPath(wsDir), "{not-json").OK)
	// Sanity: the path is discovered (so the continue, not an empty walk, is
	// what makes drainOne return false).
	core.RequireTrue(t, len(agentic.WorkspaceStatusPaths()) == 1)

	svc := New()
	core.AssertFalse(t, svc.drainOne())
}

// --- handleWorkspaceQuery: whole-registry fall-through (runner.go:270) ---

// TestRunner_HandleWorkspaceQuery_Ugly_EmptyQueryReturnsRegistry — a query
// with neither Name nor Status set falls through to the final return, handing
// back the whole registry. The existing name/status tests never reach this
// leg.
func TestRunner_HandleWorkspaceQuery_Ugly_EmptyQueryReturnsRegistry(t *testing.T) {
	svc := New()
	svc.TrackWorkspace("ws-1", &WorkspaceStatus{Status: "running"})
	svc.TrackWorkspace("ws-2", &WorkspaceStatus{Status: "queued"})

	result := svc.handleWorkspaceQuery(nil, WorkspaceQuery{})
	core.RequireTrue(t, result.OK)
	registry, ok := result.Value.(*core.Registry[*WorkspaceStatus])
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 2, registry.Len())
}
