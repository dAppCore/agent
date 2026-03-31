// SPDX-License-Identifier: EUPL-1.2

package agentic

import "fmt"

func Example_parseContentResult() {
	result := parseContentResult(map[string]any{
		"batch_id":      "batch_123",
		"provider":      "claude",
		"model":         "claude-3.7-sonnet",
		"content":       "Draft ready",
		"output_tokens": 64,
	})

	fmt.Println(result.BatchID, result.Provider, result.Model, result.OutputTokens)
	// Output: batch_123 claude claude-3.7-sonnet 64
}
