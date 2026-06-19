// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestPlatformTools_SyncTools_Good — the sync push/pull/status tools each call
// the platform and return a successful Result for a well-formed response.
func TestPlatformTools_SyncTools_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	core.AssertTrue(t, s.syncPushTool(ctx, SyncPushInput{}).OK)
	core.AssertTrue(t, s.syncPullTool(ctx, SyncPullInput{}).OK)
	core.AssertTrue(t, s.syncStatusTool(ctx, SyncStatusInput{}).OK)
}

// TestPlatformTools_computeBudgetMapValue_GoodBad — nil/zero budgets map to
// nil; a populated budget yields the corresponding map entries.
func TestPlatformTools_computeBudgetMapValue_GoodBad(t *testing.T) {
	core.AssertTrue(t, computeBudgetMapValue(nil) == nil)
	core.AssertTrue(t, computeBudgetMapValue(&ComputeBudget{}) == nil)
	m := computeBudgetMapValue(&ComputeBudget{MaxDailyHours: 8, QuietStart: "22:00"})
	core.AssertTrue(t, m != nil)
	core.AssertEqual(t, "22:00", m["quiet_start"])
}

// TestPlatformTools_FleetRegisterTool_Good — fleet register calls the platform
// and returns a successful FleetNode Result for a well-formed response.
func TestPlatformTools_FleetRegisterTool_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"agent_id":"node-1","platform":"darwin"}}`))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	r := s.fleetRegisterTool(context.Background(), FleetNode{AgentID: "node-1", Platform: "darwin", Models: []string{"go"}})
	core.AssertTrue(t, r.OK)
}

// TestPlatformTools_FleetHeartbeatTool_Good — fleet heartbeat calls the platform
// with a valid node and returns a successful Result.
func TestPlatformTools_FleetHeartbeatTool_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"agent_id":"node-1"}}`))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	r := s.fleetHeartbeatTool(context.Background(), FleetNode{AgentID: "node-1", Status: "online"})
	core.AssertTrue(t, r.OK)
}

// TestPlatformTools_FleetDeregisterTool_Good — fleet deregister calls the
// platform with a valid agent id and returns a successful Result.
func TestPlatformTools_FleetDeregisterTool_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	r := s.fleetDeregisterTool(context.Background(), FleetDeregisterInput{AgentID: "node-1"})
	core.AssertTrue(t, r.OK)
}

// TestPlatformTools_AuthProvisionTool_Good — auth provision calls the platform
// with a valid oauth user + name and returns a successful Result.
func TestPlatformTools_AuthProvisionTool_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"agent_id":"a1","local_key":"k1"}}`))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	r := s.authProvisionTool(context.Background(), AuthProvisionInput{OAuthUserID: "u1", Name: "agent"})
	core.AssertTrue(t, r.OK)
}

// TestPlatformTools_AuthRevokeAndLogin_ReachPlatform — auth revoke (by key id)
// and auth login (by pairing code) each build their request and call the
// platform endpoint (verified by the mock recording both hits).
func TestPlatformTools_AuthRevokeAndLogin_ReachPlatform(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	s.authRevokeTool(ctx, AuthRevokeInput{KeyID: "k1"})
	s.authLoginTool(ctx, AuthLoginInput{Code: "123456"})
	core.AssertTrue(t, hits >= 2)
}

// TestPlatformTools_RemainingTools_Exercised — drive the remaining fleet/credits/
// subscription tools through their request-building paths; the list/get tools
// reach the platform (mock records hits), the rest exercise their guards.
func TestPlatformTools_RemainingTools_Exercised(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	s.fleetNodesTool(ctx, FleetNodesInput{})
	s.fleetTaskAssignTool(ctx, FleetTaskAssignInput{})
	s.fleetTaskCompleteTool(ctx, FleetTaskCompleteInput{})
	s.fleetTaskNextTool(ctx, FleetTaskNextInput{})
	s.fleetEventsTool(ctx, FleetEventsInput{})
	s.creditsAwardTool(ctx, CreditsAwardInput{})
	s.creditsBalanceTool(ctx, CreditsBalanceInput{})
	s.creditsHistoryTool(ctx, CreditsHistoryInput{})
	s.subscriptionDetectTool(ctx, SubscriptionDetectInput{})
	s.subscriptionBudgetTool(ctx, SubscriptionBudgetInput{})
	s.subscriptionBudgetUpdateTool(ctx, SubscriptionBudgetUpdateInput{})
	core.AssertTrue(t, hits > 0)
}
