// SPDX-License-Identifier: EUPL-1.2

package messages

import (
	"testing"

	core "dappco.re/go"
)

// TestMessages_AllSatisfyMessage_Good verifies every message type can be
// used as a core.Message (which is `any`). Compile-time + runtime check.
func TestMessages_AllSatisfyMessage_Good(t *testing.T) {
	msgs := []core.Message{
		AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5"},
		AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed"},
		QAResult{Workspace: "core/go-io/task-5", Repo: "go-io", Passed: true},
		PRCreated{Repo: "go-io", Branch: "agent/fix", PRURL: "https://forge.lthn.ai/core/go-io/pulls/1", PRNum: 1},
		PRMerged{Repo: "go-io", PRURL: "https://forge.lthn.ai/core/go-io/pulls/1", PRNum: 1},
		WorkspacePushed{Repo: "go-io", Branch: "agent/fix", Org: "core"},
		PRNeedsReview{Repo: "go-io", PRNum: 1, Reason: "merge conflict"},
		QueueDrained{Completed: 3},
		PokeQueue{},
		RateLimitDetected{Pool: "codex", Duration: "30m"},
		HarvestComplete{Repo: "go-io", Branch: "agent/fix", Files: 5},
		HarvestRejected{Repo: "go-io", Branch: "agent/fix", Reason: "binary detected"},
		InboxMessage{New: 1, Total: 3},
	}

	core.AssertLen(t, msgs, 13, "expected 13 message types")
	for _, msg := range msgs {
		core.AssertNotNil(t, msg)
	}
}

// TestMessages_TypeSwitch_Good verifies the IPC dispatch pattern works.
func TestMessages_TypeSwitch_Good(t *testing.T) {
	var msg core.Message = AgentCompleted{
		Agent:     "codex",
		Repo:      "go-io",
		Workspace: "core/go-io/task-5",
		Status:    "completed",
	}

	handled := false
	switch ev := msg.(type) {
	case AgentCompleted:
		core.AssertEqual(t, "codex", ev.Agent)
		core.AssertEqual(t, "go-io", ev.Repo)
		core.AssertEqual(t, "completed", ev.Status)
		handled = true
	}
	core.AssertTrue(t, handled)
}

// TestMessages_EmptySignal_Good verifies zero-field messages work as signals.
func TestMessages_EmptySignal_Good(t *testing.T) {
	var msg core.Message = PokeQueue{}
	_, ok := msg.(PokeQueue)
	core.AssertTrue(t, ok)
}
