// SPDX-License-Identifier: EUPL-1.2

// Success-path coverage for the cmd* platform command wrappers in
// commands_platform.go. The guard branches are already covered (see
// commands_platform_test.go / commands_more_platform_extra_test.go / the
// Example tests); the wrappers' happy paths — the handleX-success leg, the
// result.Value.(T) type assert, and the success-print block — were not.
//
// Each test points brainURL at a local httptest mux (testPrepWithPlatformServer)
// so the real api.lthn.sh is never contacted — the same pattern the
// handleX happy-path tests in platform_test.go use.

package agentic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "dappco.re/go"
)

// platformCmdMux answers every platform endpoint the cmd* wrappers reach
// with a minimal valid envelope. Routing is by path suffix so one server
// serves the whole cluster (fleet task complete fans out to credits/award).
func platformCmdMux(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasSuffix(p, "/v1/fleet/register"):
			_, _ = w.Write([]byte(`{"data":{"node":{"id":1,"agent_id":"charon","platform":"linux","status":"online"}}}`))
		case strings.HasSuffix(p, "/v1/fleet/heartbeat"):
			_, _ = w.Write([]byte(`{"data":{"node":{"id":1,"agent_id":"charon","status":"online"}}}`))
		case strings.HasSuffix(p, "/v1/fleet/deregister"):
			_, _ = w.Write([]byte(`{"data":{"agent_id":"charon"}}`))
		case strings.HasSuffix(p, "/v1/fleet/task/assign"):
			_, _ = w.Write([]byte(`{"data":{"task":{"id":7,"repo":"core/go-io","status":"assigned"}}}`))
		case strings.HasSuffix(p, "/v1/fleet/task/complete"):
			_, _ = w.Write([]byte(`{"data":{"task":{"id":7,"repo":"core/go-io","status":"completed"}}}`))
		case strings.HasSuffix(p, "/v1/credits/award"):
			_, _ = w.Write([]byte(`{"data":{"entry":{"id":3,"task_type":"fleet-task","amount":2,"balance_after":12}}}`))
		case strings.Contains(p, "/v1/credits/balance/"):
			_, _ = w.Write([]byte(`{"data":{"agent_id":"charon","balance":12,"entries":4}}`))
		case strings.Contains(p, "/v1/credits/history/"):
			_, _ = w.Write([]byte(`{"data":{"entries":[{"id":1,"task_type":"fleet-task","amount":2,"balance_after":2}],"total":1}}`))
		case strings.HasSuffix(p, "/v1/fleet/stats"):
			_, _ = w.Write([]byte(`{"data":{"nodes_online":2,"tasks_today":5,"tasks_week":20,"repos_touched":3,"findings_total":7,"compute_hours":4}}`))
		case strings.HasSuffix(p, "/v1/fleet/task/next"):
			_, _ = w.Write([]byte(`{"data":{"task":{"id":9,"repo":"core/go-io","status":"assigned"}}}`))
		case strings.Contains(p, "/v1/subscription/budget/"):
			_, _ = w.Write([]byte(`{"data":{"max_daily_hours":2}}`))
		default:
			_, _ = w.Write([]byte(`{"data":{}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCmdPlatform_FleetRegister_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdFleetRegister(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
			core.Option{Key: "platform", Value: "linux"},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "registered:")
}

func TestCmdPlatform_FleetHeartbeat_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdFleetHeartbeat(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
			core.Option{Key: "status", Value: "online"},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "heartbeat:")
}

func TestCmdPlatform_FleetDeregister_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdFleetDeregister(core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "deregistered:")
}

func TestCmdPlatform_FleetTaskAssign_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdFleetTaskAssign(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
			core.Option{Key: "repo", Value: "core/go-io"},
			core.Option{Key: "task", Value: "fix tests"},
		))
	})
	core.AssertTrue(t, r.OK)
}

func TestCmdPlatform_FleetTaskComplete_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdFleetTaskComplete(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
			core.Option{Key: "task_id", Value: 7},
		))
	})
	core.AssertTrue(t, r.OK)
}

func TestCmdPlatform_FleetTaskNext_Good_HasTask(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdFleetTaskNext(core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
}

// emptyTaskMux returns an empty data envelope so handleFleetNextTask yields
// a nil *FleetTask → the cmd wrapper's "no task available" branch.
func emptyTaskMux(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCmdPlatform_FleetTaskNext_Good_NoTask(t *testing.T) {
	s := testPrepWithPlatformServer(t, emptyTaskMux(t), "secret-token")
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdFleetTaskNext(core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "no task available")
}

func TestCmdPlatform_FleetStats_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	out := captureStdout(t, func() { r = s.cmdFleetStats(core.NewOptions()) })
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "nodes online:")
}

func TestCmdPlatform_CreditsAward_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdCreditsAward(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
			core.Option{Key: "task_type", Value: "fleet-task"},
			core.Option{Key: "amount", Value: 2},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "balance after:")
}

func TestCmdPlatform_CreditsBalance_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	out := captureStdout(t, func() {
		r = s.cmdCreditsBalance(core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "balance:")
}

func TestCmdPlatform_CreditsHistory_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdCreditsHistory(core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
}

func TestCmdPlatform_SubscriptionBudget_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdSubscriptionBudget(core.NewOptions(core.Option{Key: "agent_id", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
}

func TestCmdPlatform_SubscriptionUpdateBudget_Good(t *testing.T) {
	s := testPrepWithPlatformServer(t, platformCmdMux(t), "secret-token")
	var r core.Result
	captureStdout(t, func() {
		r = s.cmdSubscriptionUpdateBudget(core.NewOptions(
			core.Option{Key: "agent_id", Value: "charon"},
			core.Option{Key: "limits", Value: `{"max_daily_hours":2}`},
		))
	})
	core.AssertTrue(t, r.OK)
}
