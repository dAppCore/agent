// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	core "dappco.re/go"
)

// TestChat_defaultUserChatsPath_Good — the per-user archive path follows the
// ~/Lethean/data/users/<user>/chats.duckdb convention chathistory expects.
func TestChat_defaultUserChatsPath_Good(t *testing.T) {
	core.AssertContains(t, defaultUserChatsPath("owlet"),
		core.JoinPath("Lethean", "data", "users", "owlet", "chats.duckdb"))
}

// TestChat_chat_Bad_RequiresUser — chat with no --user prints guidance and
// returns a non-OK result without touching the (nil) core.
func TestChat_chat_Bad_RequiresUser(t *testing.T) {
	cmds := applicationCommandSet{}
	var r core.Result
	out := captureStdout(t, func() { r = cmds.chat(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "--user")
}
