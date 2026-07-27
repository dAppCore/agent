// SPDX-License-Identifier: EUPL-1.2

package chathistory

import (
	"testing"

	core "dappco.re/go"
)

// TestChatHistory_PathUserID_Good — the path + user-id getters return the
// archive's identity.
func TestChatHistory_PathUserID_Good(t *testing.T) {
	h := &History{path: "/x/chats.duckdb", userID: "owlet"}
	core.AssertEqual(t, "/x/chats.duckdb", h.Path())
	core.AssertEqual(t, "owlet", h.UserID())
}

// TestChatHistory_LoadTurns_Bad_Closed — a nil or closed history errors instead
// of querying.
func TestChatHistory_LoadTurns_Bad_Closed(t *testing.T) {
	var nilH *History
	_, err := nilH.LoadTurns("conv-1")
	core.AssertTrue(t, err != nil)
	_, err = (&History{}).LoadTurns("conv-1")
	core.AssertTrue(t, err != nil)
}
