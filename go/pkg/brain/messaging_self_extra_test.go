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
// remote call. Covers the To=="self" branch + notifySelf's mcp-lookup guard
// on a wired ServiceRuntime.
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

// TestMessaging_SendMessage_Self_NilRuntime_NoPanic — regression for the
// localDirect() nil-ServiceRuntime path. localDirect builds a DirectSubsystem
// with a nil embedded *core.ServiceRuntime; a "self" send routes to notifySelf,
// whose first statement calls s.Core(). Core() dereferences its receiver
// (returns r.core), so without the s.ServiceRuntime==nil short-circuit guard
// this panicked with a nil-pointer deref BEFORE the original "if s.Core()==nil"
// check could fire — a real production nil-deref. The guard now returns cleanly
// and sendMessage still reports success for the self target.
func TestMessaging_SendMessage_Self_NilRuntime_NoPanic(t *testing.T) {
	// localDirect() == &DirectSubsystem{...} with ServiceRuntime nil.
	sub := localDirect()
	core.AssertTrue(t, sub.ServiceRuntime == nil)

	_, out, err := sub.sendMessage(context.Background(), nil, SendInput{
		To:      "self",
		Content: "note to self",
		Subject: "reminder",
	})
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertEqual(t, "self", out.To)
}
