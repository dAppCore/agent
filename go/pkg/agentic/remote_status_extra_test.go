// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestRemote_statusRemote_Bad — remote status with empty input + an erroring
// platform fails rather than panicking.
func TestRemote_statusRemote_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	core.AssertFalse(t, s.statusRemote(context.Background(), RemoteStatusInput{}).OK)
}
