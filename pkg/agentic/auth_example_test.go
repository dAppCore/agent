// SPDX-License-Identifier: EUPL-1.2

package agentic

import "fmt"

func Example_parseAgentAPIKey() {
	key := parseAgentAPIKey(map[string]any{
		"id":          7,
		"name":        "codex local",
		"prefix":      "ak_live",
		"permissions": []any{"plans:read", "plans:write"},
	})

	fmt.Println(key.ID, key.Name, key.Prefix, len(key.Permissions))
	// Output: 7 codex local ak_live 2
}
