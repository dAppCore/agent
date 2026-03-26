// SPDX-License-Identifier: EUPL-1.2

package messages

import (
	"fmt"
)

func ExampleAgentCompleted() {
	ev := AgentCompleted{
		Agent:     "codex",
		Repo:      "go-io",
		Workspace: "core/go-io/task-5",
		Status:    "completed",
	}
	fmt.Println(ev.Agent, ev.Status)
	// Output: codex completed
}

func ExampleQAResult() {
	ev := QAResult{
		Workspace: "core/go-io/task-5",
		Repo:      "go-io",
		Passed:    true,
	}
	fmt.Println(ev.Repo, ev.Passed)
	// Output: go-io true
}

func ExampleQueueDrained() {
	ev := QueueDrained{Completed: 3}
	fmt.Println(ev.Completed)
	// Output: 3
}
