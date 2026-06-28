// SPDX-License-Identifier: EUPL-1.2

// HandleIPCEvents coverage for the three events that were emitted but
// unhandled before (H1/H4/H5 in PLAN-cli-square-up.md): QueueDrained,
// HarvestRejected, and InboxMessage. Each now reaches sendNotification with a
// dedicated channel + typed payload, so a previously floor-dropped broadcast
// becomes an observable notification. Reuses the recordingMCP channelSender
// seam from runner_coverage_extra_test.go.

package runner

import (
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// TestRunner_HandleIPCEvents_QueueDrained_NotifiesMCP — QueueDrained surfaces
// on the queue.status channel carrying the completed count (H1).
func TestRunner_HandleIPCEvents_QueueDrained_NotifiesMCP(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.QueueDrained{Completed: 3})
	core.AssertTrue(t, r.OK)

	core.RequireTrue(t, len(mcp.channels) == 1, "exactly one notification emitted")
	core.AssertEqual(t, "queue.status", mcp.channels[0])
	notification, ok := mcp.payloads[0].(*QueueNotification)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 3, notification.Completed)
}

// TestRunner_HandleIPCEvents_HarvestRejected_NotifiesMCP — HarvestRejected
// surfaces on harvest.status with status "rejected" + the rejection reason, so
// a rejected harvest is visible rather than silently dropped (H4).
func TestRunner_HandleIPCEvents_HarvestRejected_NotifiesMCP(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.HarvestRejected{
		Repo: "go-io", Branch: "agent/fix-tests", Reason: "binary detected",
	})
	core.AssertTrue(t, r.OK)

	core.RequireTrue(t, len(mcp.channels) == 1, "exactly one notification emitted")
	core.AssertEqual(t, "harvest.status", mcp.channels[0])
	notification, ok := mcp.payloads[0].(*HarvestNotification)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "rejected", notification.Status)
	core.AssertEqual(t, "go-io", notification.Repo)
	core.AssertEqual(t, "agent/fix-tests", notification.Branch)
	core.AssertEqual(t, "binary detected", notification.Reason)
}

// TestRunner_HandleIPCEvents_InboxMessage_NotifiesMCP — InboxMessage surfaces
// on inbox.status with the new + total counts so OpenBrain inbox arrivals are
// observable (H5).
func TestRunner_HandleIPCEvents_InboxMessage_NotifiesMCP(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.InboxMessage{New: 2, Total: 5})
	core.AssertTrue(t, r.OK)

	core.RequireTrue(t, len(mcp.channels) == 1, "exactly one notification emitted")
	core.AssertEqual(t, "inbox.status", mcp.channels[0])
	notification, ok := mcp.payloads[0].(*InboxNotification)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 2, notification.New)
	core.AssertEqual(t, 5, notification.Total)
}

// TestRunner_HandleIPCEvents_RateLimitDetected_BacksOffPool — RateLimitDetected
// records a future backoff for the named pool (so drainOne skips it) and surfaces
// it on ratelimit.status (H2). The backoff map was read at queue.go:219 but had
// no writer until this handler.
func TestRunner_HandleIPCEvents_RateLimitDetected_BacksOffPool(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.RateLimitDetected{Pool: "codex", Duration: "30m"})
	core.AssertTrue(t, r.OK)

	until, ok := svc.backoff["codex"]
	core.RequireTrue(t, ok, "codex pool backed off")
	core.AssertTrue(t, until.After(time.Now()))

	core.RequireTrue(t, len(mcp.channels) == 1, "exactly one notification emitted")
	core.AssertEqual(t, "ratelimit.status", mcp.channels[0])
	notification, ok := mcp.payloads[0].(*RateLimitNotification)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "codex", notification.Pool)
	core.AssertEqual(t, "30m", notification.Duration)
}

// TestRunner_HandleIPCEvents_RateLimitDetected_BadDuration_NotifiesNoBackoff —
// an unparseable Duration still notifies but records no backoff (the pool is not
// silently frozen forever on a malformed event).
func TestRunner_HandleIPCEvents_RateLimitDetected_BadDuration_NotifiesNoBackoff(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.RateLimitDetected{Pool: "codex", Duration: "not-a-duration"})
	core.AssertTrue(t, r.OK)

	_, ok := svc.backoff["codex"]
	core.AssertFalse(t, ok)
	core.RequireTrue(t, len(mcp.channels) == 1, "still surfaced")
	core.AssertEqual(t, "ratelimit.status", mcp.channels[0])
}

// TestRunner_HandleIPCEvents_HarvestComplete_NotifiesMCP — HarvestComplete
// surfaces on harvest.status with status "complete" + the file count (H3, notify
// side). The auto-pr re-dispatch lives in the agentic handler on the same event.
func TestRunner_HandleIPCEvents_HarvestComplete_NotifiesMCP(t *testing.T) {
	c := core.New(core.WithOption("name", "runner-mcp"))
	mcp := &recordingMCP{}
	core.RequireTrue(t, c.RegisterService("mcp", mcp).OK)

	svc := New()
	svc.ServiceRuntime = core.NewServiceRuntime(c, Options{})

	r := svc.HandleIPCEvents(c, messages.HarvestComplete{Repo: "go-io", Branch: "agent/fix-tests", Files: 5})
	core.AssertTrue(t, r.OK)

	core.RequireTrue(t, len(mcp.channels) == 1, "exactly one notification emitted")
	core.AssertEqual(t, "harvest.status", mcp.channels[0])
	notification, ok := mcp.payloads[0].(*HarvestNotification)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "complete", notification.Status)
	core.AssertEqual(t, "go-io", notification.Repo)
	core.AssertEqual(t, "agent/fix-tests", notification.Branch)
	core.AssertEqual(t, 5, notification.Files)
}
