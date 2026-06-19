// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestAgenticHandlers_IdentifierGuards — sprint + content handlers that require
// an identifier reject empty input before any platform call (the mock platform
// guarantees no real network is touched).
func TestAgenticHandlers_IdentifierGuards(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	s := testPrepWithPlatformServer(t, srv, "token")
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertFalse(t, s.sprintGet(ctx, SprintGetInput{}).OK)
		core.AssertFalse(t, s.sprintStart(ctx, SprintTransitionInput{}).OK)
		core.AssertFalse(t, s.sprintComplete(ctx, SprintTransitionInput{}).OK)
		core.AssertFalse(t, s.sprintArchive(ctx, SprintArchiveInput{}).OK)
		core.AssertFalse(t, s.contentBriefGet(ctx, ContentBriefGetInput{}).OK)
	})
}
