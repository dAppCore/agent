// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestBrainActions_Handlers_Guards — remember/forget/send reject empty options
// (missing key / recipient) rather than panicking.
func TestBrainActions_Handlers_Guards(t *testing.T) {
	sub := &DirectSubsystem{ServiceRuntime: core.NewServiceRuntime(core.New(), DirectOptions{})}
	ctx := context.Background()
	core.AssertFalse(t, sub.handleRemember(ctx, core.NewOptions()).OK)
	core.AssertFalse(t, sub.handleForget(ctx, core.NewOptions()).OK)
	core.AssertFalse(t, sub.handleSend(ctx, core.NewOptions()).OK)
}
