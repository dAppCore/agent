// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"time"

	core "dappco.re/go/core"
)

// event := agentic.CompletionEvent{Type: "agent_completed", Agent: "codex", Workspace: "go-io-123", Status: "completed"}
type CompletionEvent struct {
	Type      string `json:"type"`
	Agent     string `json:"agent"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func emitEvent(eventType, agent, workspace, status string) {
	eventsFile := core.JoinPath(WorkspaceRoot(), "events.jsonl")

	event := CompletionEvent{
		Type:      eventType,
		Agent:     agent,
		Workspace: workspace,
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	line := core.Concat(core.JSONMarshalString(event), "\n")

	r := fs.Append(eventsFile)
	if !r.OK {
		return
	}
	core.WriteAll(r.Value, line)
}

func emitStartEvent(agent, workspace string) {
	emitEvent("agent_started", agent, workspace, "running")
}

func emitCompletionEvent(agent, workspace, status string) {
	emitEvent("agent_completed", agent, workspace, status)
}
