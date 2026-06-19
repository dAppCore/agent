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
