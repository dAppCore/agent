// SPDX-License-Identifier: EUPL-1.2

package agentic

import "fmt"

func Example_parseFleetNode() {
	node := parseFleetNode(map[string]any{
		"agent_id":     "charon",
		"platform":     "linux",
		"models":       []any{"codex", "gpt-5.4"},
		"capabilities": map[string]any{"go": true, "review": true},
		"status":       "online",
	})

	fmt.Println(node.AgentID, node.Platform, len(node.Models), len(node.Capabilities))
	// Output: charon linux 2 2
}
