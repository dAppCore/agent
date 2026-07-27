// SPDX-License-Identifier: EUPL-1.2

// Error-path coverage for the cmd* platform command wrappers in
// commands_platform.go. The guard branches (missing identifier) and the
// happy paths are already covered elsewhere; the uncovered leg in nearly
// every wrapper is the "handleX returned !OK" branch — a backend failure
// that the wrapper prints and propagates. One always-500 backend drives
// that branch across the whole cluster, plus a few success-path variants
// (empty-list / status optional lines) that the happy-path tests skipped.

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// platformFailServer answers every route with 500 so each wrapper's
// handleX-error branch fires.
func platformFailServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestCmdPlatform_ErrorPaths_BackendDown — with valid args (so the guard
// passes) but a 500 backend, every platform wrapper returns non-OK via its
// handleX-error branch.
func TestCmdPlatform_ErrorPaths_BackendDown(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformFailServer(t), "secret-token")

	id := func(v string) core.Option { return core.Option{Key: "_arg", Value: v} }

	cases := []struct {
		name string
		call func() core.Result
	}{
		{"auth-provision", func() core.Result { return s.cmdAuthProvision(core.NewOptions(id("user-1"))) }},
		{"auth-revoke", func() core.Result { return s.cmdAuthRevoke(core.NewOptions(id("42"))) }},
		{"auth-login", func() core.Result { return s.cmdAuthLogin(core.NewOptions(id("123456"))) }},
		{"fleet-register", func() core.Result {
			return s.cmdFleetRegister(core.NewOptions(id("charon"), core.Option{Key: "platform", Value: "linux"}))
		}},
		{"fleet-heartbeat", func() core.Result {
			return s.cmdFleetHeartbeat(core.NewOptions(id("charon"), core.Option{Key: "status", Value: "online"}))
		}},
		{"fleet-deregister", func() core.Result { return s.cmdFleetDeregister(core.NewOptions(id("charon"))) }},
		{"fleet-nodes", func() core.Result { return s.cmdFleetNodes(core.NewOptions()) }},
		{"fleet-task-assign", func() core.Result {
			return s.cmdFleetTaskAssign(core.NewOptions(id("charon"),
				core.Option{Key: "repo", Value: "core/go-io"}, core.Option{Key: "task", Value: "do it"}))
		}},
		{"fleet-task-complete", func() core.Result {
			return s.cmdFleetTaskComplete(core.NewOptions(
				core.Option{Key: "agent_id", Value: "charon"}, core.Option{Key: "task_id", Value: 7}))
		}},
		{"fleet-task-next", func() core.Result {
			return s.cmdFleetTaskNext(core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
		}},
		{"fleet-stats", func() core.Result { return s.cmdFleetStats(core.NewOptions()) }},
		{"credits-balance", func() core.Result { return s.cmdCreditsBalance(core.NewOptions(id("charon"))) }},
		{"credits-history", func() core.Result { return s.cmdCreditsHistory(core.NewOptions(id("charon"))) }},
		{"credits-award", func() core.Result {
			return s.cmdCreditsAward(core.NewOptions(id("charon"),
				core.Option{Key: "task_type", Value: "fleet-task"}, core.Option{Key: "amount", Value: 2}))
		}},
		{"subscription-budget", func() core.Result { return s.cmdSubscriptionBudget(core.NewOptions(id("charon"))) }},
		{"subscription-update-budget", func() core.Result {
			return s.cmdSubscriptionUpdateBudget(core.NewOptions(id("charon"),
				core.Option{Key: "limits", Value: map[string]any{"max_daily_hours": 2}}))
		}},
		{"subscription-detect", func() core.Result { return s.cmdSubscriptionDetect(core.NewOptions(id("charon"))) }},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var r core.Result
			captureStdout(t, func() { r = tc.call() })
			core.AssertFalse(t, r.OK)
		})
	}
}

// TestCmdPlatform_FleetNodes_Good_Empty — an empty node list prints the
// "no fleet nodes" line and returns OK.
func TestCmdPlatform_FleetNodes_Good_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"nodes":[],"total":0}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	out := captureStdout(t, func() { r = s.cmdFleetNodes(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "no fleet nodes")
}

// TestCmdPlatform_FleetNodes_Good_Populated — a populated node list renders
// each node row and the total.
func TestCmdPlatform_FleetNodes_Good_Populated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"nodes":[{"agent_id":"charon","platform":"linux","status":"online","models":["codex"]}],"total":1}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	out := captureStdout(t, func() { r = s.cmdFleetNodes(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "charon")
	core.AssertContains(t, out, "total: 1")
}

// TestCmdPlatform_SyncStatus_Good_RemoteEnvelope — a remote status envelope
// (nested under "status") populates the agent / status / last-push / last-pull
// lines from the response.
func TestCmdPlatform_SyncStatus_Good_RemoteEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"status":{"agent_id":"charon","status":"synced","queued":1,"context_count":3,"last_push_at":"2026-01-01","last_pull_at":"2026-01-02"}}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	out := captureStdout(t, func() { r = s.cmdSyncStatus(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "status:        synced")
	core.AssertContains(t, out, "last push:     2026-01-01")
	core.AssertContains(t, out, "last pull:     2026-01-02")
}

// TestCmdPlatform_SyncStatus_Good_RemoteError — when the remote status probe
// fails, the local status carries a remote-error line and the command still
// returns OK (sync status is local-first).
func TestCmdPlatform_SyncStatus_Good_RemoteError(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformFailServer(t), "secret-token")

	var r core.Result
	out := captureStdout(t, func() { r = s.cmdSyncStatus(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "remote error:")
}

// TestCmdPlatform_SyncPushPull_Good — push + pull render their count lines.
func TestCmdPlatform_SyncPushPull_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"count":4,"items":[],"synced":4}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	out := captureStdout(t, func() {
		core.AssertTrue(t, s.cmdSyncPush(core.NewOptions()).OK)
		core.AssertTrue(t, s.cmdSyncPull(core.NewOptions()).OK)
	})
	core.AssertContains(t, out, "synced:")
	core.AssertContains(t, out, "context items:")
}

// --- registerPlatformCommands: first conflict ------------------------

// TestCommandsPlatform_RegisterPlatformCommands_Bad_Conflict — a pre-registered
// sync/push command makes the first registration inside
// registerPlatformCommands fail, exercising the error-return branch.
func TestCommandsPlatform_RegisterPlatformCommands_Bad_Conflict(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{})}
	core.AssertTrue(t, c.Command("sync/push", core.Command{
		Description: "conflict",
		Action:      func(_ core.Options) core.Result { return core.Result{OK: true} },
	}).OK)

	r := s.registerPlatformCommands()
	core.AssertFalse(t, r.OK)
}
