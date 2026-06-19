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
