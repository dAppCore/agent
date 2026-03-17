// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	coreio "forge.lthn.ai/core/go-io"
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
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	eventsFile := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace", "events.jsonl")

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
	existing, _ := coreio.Local.Read(eventsFile)
	coreio.Local.Write(eventsFile, existing+string(data)+"\n")
}
