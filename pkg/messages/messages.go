// SPDX-License-Identifier: EUPL-1.2

// Package messages defines IPC message types for inter-service communication
// within core-agent. Services emit these via c.ACTION() and handle them via
// c.RegisterAction(). No service imports another — they share only these types.
//
//	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Status: "completed"})
package messages

// --- Agent Lifecycle ---

// AgentStarted is broadcast when a subagent process is spawned.
//
//	c.ACTION(messages.AgentStarted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5"})
type AgentStarted struct {
	Agent     string
	Repo      string
	Workspace string
}

// AgentCompleted is broadcast when a subagent process exits.
//
//	c.ACTION(messages.AgentCompleted{Agent: "codex", Repo: "go-io", Workspace: "core/go-io/task-5", Status: "completed"})
type AgentCompleted struct {
	Agent     string
	Repo      string
	Workspace string
	Status    string // completed, failed, blocked
}

// --- QA & PR Pipeline ---

// QAResult is broadcast after QA runs on a completed workspace.
//
//	c.ACTION(messages.QAResult{Workspace: "core/go-io/task-5", Repo: "go-io", Passed: true})
type QAResult struct {
	Workspace string
	Repo      string
	Passed    bool
	Output    string
}

// PRCreated is broadcast after a PR is auto-created on Forge.
//
//	c.ACTION(messages.PRCreated{Repo: "go-io", Branch: "agent/fix-tests", PRURL: "https://...", PRNum: 12})
type PRCreated struct {
	Repo   string
	Branch string
	PRURL  string
	PRNum  int
}

// PRMerged is broadcast after a PR is auto-verified and merged.
//
//	c.ACTION(messages.PRMerged{Repo: "go-io", PRURL: "https://...", PRNum: 12})
type PRMerged struct {
	Repo  string
	PRURL string
	PRNum int
}

// PRNeedsReview is broadcast when auto-merge fails and human attention is needed.
//
//	c.ACTION(messages.PRNeedsReview{Repo: "go-io", PRNum: 12, Reason: "merge conflict"})
type PRNeedsReview struct {
	Repo   string
	PRURL  string
	PRNum  int
	Reason string
}

// --- Queue ---

// QueueDrained is broadcast when running=0 and queued=0 (genuinely empty).
//
//	c.ACTION(messages.QueueDrained{Completed: 3})
type QueueDrained struct {
	Completed int
}

// PokeQueue signals the runner to drain the queue immediately.
//
//	c.ACTION(messages.PokeQueue{})
type PokeQueue struct{}

// SpawnQueued is sent by the runner to request agentic spawn a queued workspace.
// Runner gates (frozen + concurrency), agentic owns the actual process spawn.
//
//	c.ACTION(messages.SpawnQueued{Workspace: "core/go-io/task-5", Agent: "codex", Task: "review"})
type SpawnQueued struct {
	Workspace string
	Agent     string
	Task      string
}

// RateLimitDetected is broadcast when fast failures trigger agent pool backoff.
//
//	c.ACTION(messages.RateLimitDetected{Pool: "codex", Duration: "30m"})
type RateLimitDetected struct {
	Pool     string
	Duration string
}

// --- Monitor Events ---

// HarvestComplete is broadcast when a workspace branch is ready for review.
//
//	c.ACTION(messages.HarvestComplete{Repo: "go-io", Branch: "agent/fix-tests", Files: 5})
type HarvestComplete struct {
	Repo   string
	Branch string
	Files  int
}

// HarvestRejected is broadcast when a workspace fails safety checks (binaries, size).
//
//	c.ACTION(messages.HarvestRejected{Repo: "go-io", Branch: "agent/fix-tests", Reason: "binary detected"})
type HarvestRejected struct {
	Repo   string
	Branch string
	Reason string
}

// InboxMessage is broadcast when a new inter-agent message arrives.
//
//	c.ACTION(messages.InboxMessage{From: "charon", Subject: "status", Content: "all green"})
type InboxMessage struct {
	From    string
	Subject string
	Content string
}
