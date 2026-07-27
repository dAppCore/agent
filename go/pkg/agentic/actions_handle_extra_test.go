// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestContent_HandleWrappers_Guards — the content batch-generate + brief-get
// action wrappers reject empty options (missing batch/brief id).
func TestContent_HandleWrappers_Guards(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	ctx := context.Background()
	captureStdout(t, func() {
		core.AssertFalse(t, s.handleContentBatchGenerate(ctx, core.NewOptions()).OK)
		core.AssertFalse(t, s.handleContentBriefGet(ctx, core.NewOptions()).OK)
	})
}
