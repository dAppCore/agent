// SPDX-License-Identifier: EUPL-1.2

package messages

import core "dappco.re/go"

func ExampleAgentCompleted() {
	ev := AgentCompleted{
		Agent:     "codex",
		Repo:      "go-io",
		Workspace: "core/go-io/task-5",
		Status:    "completed",
	}
	core.Println(ev.Agent, ev.Status)
	// Output: codex completed
}

func ExampleQAResult() {
	ev := QAResult{
		Workspace: "core/go-io/task-5",
		Repo:      "go-io",
		Passed:    true,
	}
	core.Println(ev.Repo, ev.Passed)
	// Output: go-io true
}

func ExampleQueueDrained() {
	ev := QueueDrained{Completed: 3}
	core.Println(ev.Completed)
	// Output: 3
}

func ExampleWorkspacePushed() {
	ev := WorkspacePushed{Repo: "go-io", Branch: "agent/fix-tests", Org: "core"}
	core.Println(ev.Repo, ev.Org)
	// Output: go-io core
}
