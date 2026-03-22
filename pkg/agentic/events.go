// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"io"
	"time"

	core "dappco.re/go/core"
)

// CompletionEvent is emitted when a dispatched agent finishes.
// Written to ~/.core/workspace/events.jsonl as append-only log.
//
//	event := agentic.CompletionEvent{Type: "agent_completed", Agent: "codex", Workspace: "go-io-123", Status: "completed"}
type CompletionEvent struct {
	Type      string `json:"type"`
	Agent     string `json:"agent"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// emitCompletionEvent appends a completion event to the events log.
// The plugin's hook watches this file to notify the orchestrating agent.
// Status should be the actual terminal state: completed, failed, or blocked.
func emitCompletionEvent(agent, workspace, status string) {
	eventsFile := core.JoinPath(WorkspaceRoot(), "events.jsonl")

	event := CompletionEvent{
		Type:      "agent_completed",
		Agent:     agent,
		Workspace: workspace,
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	// Append to events log
	r := fs.Append(eventsFile)
	if !r.OK {
		return
	}
	wc := r.Value.(io.WriteCloser)
	defer wc.Close()
	wc.Write(append(data, '\n'))
}
