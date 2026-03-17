// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CompletionEvent is emitted when a dispatched agent finishes.
// Written to ~/.core/workspace/events.jsonl as append-only log.
type CompletionEvent struct {
	Type      string `json:"type"`
	Agent     string `json:"agent"`
	Workspace string `json:"workspace"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// emitCompletionEvent appends a completion event to the events log.
// The plugin's hook watches this file to notify the orchestrating agent.
func emitCompletionEvent(agent, workspace string) {
	eventsFile := filepath.Join(WorkspaceRoot(), "events.jsonl")

	event := CompletionEvent{
		Type:      "agent_completed",
		Agent:     agent,
		Workspace: workspace,
		Status:    "completed",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	// Append to events log
	f, err := os.OpenFile(eventsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}
