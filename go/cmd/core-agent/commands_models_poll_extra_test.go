// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestModels_pollDownload_CancelledContext — an already-cancelled context
// makes pollDownload's first select fire on ctx.Done() before any admin
// call, printing the timeout line and returning a non-OK result. Covers the
// loop-entry + ctx-timeout branch without touching the network (admin is
// never dereferenced on this path).
func TestModels_pollDownload_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the first iteration

	var r core.Result
	out := captureStdout(t, func() {
		r = pollDownload(ctx, nil, "job-123")
	})

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "job-123")
}
