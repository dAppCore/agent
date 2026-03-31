// SPDX-License-Identifier: EUPL-1.2

// c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Status: "completed"})
package messages

// c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5"})
type AgentStarted struct {
	Agent     string
	Repo      string
	Workspace string
}

// c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed"})
type AgentCompleted struct {
	Agent     string
	Repo      string
	Workspace string
	Status    string
}

// c.ACTION(messages.QAResult{Workspace: "core/go-io/task-5", Repo: "go-io", Passed: true})
type QAResult struct {
	Workspace string
	Repo      string
	Passed    bool
	Output    string
}

// c.ACTION(messages.PRCreated{Repo: "go-io", Branch: "agent/fix-tests", PRURL: "https://...", PRNum: 12})
type PRCreated struct {
	Repo   string
	Branch string
	PRURL  string
	PRNum  int
}

// c.ACTION(messages.PRMerged{Repo: "go-io", PRURL: "https://...", PRNum: 12})
type PRMerged struct {
	Repo  string
	PRURL string
	PRNum int
}

// c.ACTION(messages.PRNeedsReview{Repo: "go-io", PRNum: 12, Reason: "merge conflict"})
type PRNeedsReview struct {
	Repo   string
	PRURL  string
	PRNum  int
	Reason string
}

// c.ACTION(messages.QueueDrained{Completed: 3})
type QueueDrained struct {
	Completed int
}

// c.ACTION(messages.PokeQueue{})
type PokeQueue struct{}

// c.ACTION(messages.SpawnQueued{Workspace: "core/go-io/task-5", Agent: "codex", Task: "review"})
type SpawnQueued struct {
	Workspace string
	Agent     string
	Task      string
}

// c.ACTION(messages.RateLimitDetected{Pool: "codex", Duration: "30m"})
type RateLimitDetected struct {
	Pool     string
	Duration string
}

// c.ACTION(messages.HarvestComplete{Repo: "go-io", Branch: "agent/fix-tests", Files: 5})
type HarvestComplete struct {
	Repo   string
	Branch string
	Files  int
}

// c.ACTION(messages.HarvestRejected{Repo: "go-io", Branch: "agent/fix-tests", Reason: "binary detected"})
type HarvestRejected struct {
	Repo   string
	Branch string
	Reason string
}

// c.ACTION(messages.InboxMessage{From: "charon", Subject: "status", Content: "all green"})
type InboxMessage struct {
	From    string
	Subject string
	Content string
}
