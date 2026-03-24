// SPDX-License-Identifier: EUPL-1.2

package messages

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// TestMessageTypes_Good_AllSatisfyMessage verifies every message type can be
// used as a core.Message (which is `any`). This is a compile-time + runtime check.
func TestMessageTypes_Good_AllSatisfyMessage(t *testing.T) {
	msgs := []core.Message{
		AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5"},
		AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed"},
		QAResult{Workspace: "core/go-io/task-5", Repo: "go-io", Passed: true},
		PRCreated{Repo: "go-io", Branch: "agent/fix", PRURL: "https://forge.lthn.ai/core/go-io/pulls/1", PRNum: 1},
		PRMerged{Repo: "go-io", PRURL: "https://forge.lthn.ai/core/go-io/pulls/1", PRNum: 1},
		PRNeedsReview{Repo: "go-io", PRNum: 1, Reason: "merge conflict"},
		QueueDrained{Completed: 3},
		PokeQueue{},
		RateLimitDetected{Pool: "codex", Duration: "30m"},
		HarvestComplete{Repo: "go-io", Branch: "agent/fix", Files: 5},
		HarvestRejected{Repo: "go-io", Branch: "agent/fix", Reason: "binary detected"},
		InboxMessage{New: 2, Total: 15},
	}

	assert.Len(t, msgs, 12, "expected 12 message types")
	for _, msg := range msgs {
		assert.NotNil(t, msg)
	}
}

// TestAgentCompleted_Good_TypeSwitch verifies the IPC dispatch pattern works.
func TestAgentCompleted_Good_TypeSwitch(t *testing.T) {
	var msg core.Message = AgentCompleted{
		Agent:     "codex",
		Repo:      "go-io",
		Workspace: "core/go-io/task-5",
		Status:    "completed",
	}

	handled := false
	switch ev := msg.(type) {
	case AgentCompleted:
		assert.Equal(t, "codex", ev.Agent)
		assert.Equal(t, "go-io", ev.Repo)
		assert.Equal(t, "completed", ev.Status)
		handled = true
	}
	assert.True(t, handled)
}

// TestPokeQueue_Good_EmptyMessage verifies zero-field messages work as signals.
func TestPokeQueue_Good_EmptyMessage(t *testing.T) {
	var msg core.Message = PokeQueue{}
	_, ok := msg.(PokeQueue)
	assert.True(t, ok)
}
