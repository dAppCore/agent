package loop

import (
	"encoding/json"
	"regexp"
	"strings"
)

var toolBlockRe = regexp.MustCompile("(?s)```tool\\s*\n(.*?)\\s*```")

// ParseToolCalls extracts tool invocations from fenced ```tool blocks in
// model output. Only blocks tagged "tool" are matched; other fenced blocks
// (```go, ```json, etc.) pass through untouched. Malformed JSON is silently
// skipped. Returns the parsed calls and the cleaned text with tool blocks
// removed.
func ParseToolCalls(output string) ([]ToolUse, string) {
	matches := toolBlockRe.FindAllStringSubmatchIndex(output, -1)
	if len(matches) == 0 {
		return nil, output
	}

	var calls []ToolUse
	cleaned := output

	// Walk matches in reverse so index arithmetic stays valid after each splice.
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		fullStart, fullEnd := m[0], m[1]
		bodyStart, bodyEnd := m[2], m[3]

		body := strings.TrimSpace(output[bodyStart:bodyEnd])
		if body == "" {
			cleaned = cleaned[:fullStart] + cleaned[fullEnd:]
			continue
		}

		var call ToolUse
		if err := json.Unmarshal([]byte(body), &call); err != nil {
			cleaned = cleaned[:fullStart] + cleaned[fullEnd:]
			continue
		}

		calls = append([]ToolUse{call}, calls...)
		cleaned = cleaned[:fullStart] + cleaned[fullEnd:]
	}

	cleaned = strings.TrimSpace(cleaned)
	return calls, cleaned
}
