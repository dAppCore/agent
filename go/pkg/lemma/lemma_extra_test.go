// SPDX-License-Identifier: EUPL-1.2

package lemma

import (
	"testing"

	core "dappco.re/go"
)

// TestLemma_ResumeAndConversationID_Good — Resume builds a session bound to the
// conversation id; ConversationID reads it back (nil-safe).
func TestLemma_ResumeAndConversationID_Good(t *testing.T) {
	sess := (&Service{}).Resume("owlet", "conv-1")
	core.AssertEqual(t, "conv-1", sess.ConversationID())

	var nilSess *Session
	core.AssertEqual(t, "", nilSess.ConversationID())
}
