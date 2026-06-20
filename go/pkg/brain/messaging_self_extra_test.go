// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"testing"

	core "dappco.re/go"
)

// TestMessaging_SendMessage_Self_NoMCP — a "self"-targeted send routes to
// notifySelf, which short-circuits on the "mcp service not found" guard (a
// bare Core has no mcp service registered) and reports success without any
// remote call. Covers the To=="self" branch + notifySelf's mcp-lookup guard.
//
// A real Core is required (not localDirect) because notifySelf calls
// s.Core() unconditionally — localDirect has a nil ServiceRuntime and would
// panic before the guard, so it only suits the pre-self input-validation
// paths.
func TestMessaging_SendMessage_Self_NoMCP(t *testing.T) {
	sub := &DirectSubsystem{ServiceRuntime: core.NewServiceRuntime(core.New(), DirectOptions{})}
	_, out, err := sub.sendMessage(context.Background(), nil, SendInput{
		To:      "self",
		Content: "note to self",
		Subject: "reminder",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "self", out.To)
}
