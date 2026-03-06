package loop

import "context"

const (
	RoleUser       = "user"
	RoleAssistant  = "assistant"
	RoleToolResult = "tool_result"
	RoleSystem     = "system"
)

// Message represents one turn in the conversation.
type Message struct {
	Role     string    `json:"role"`
	Content  string    `json:"content"`
	ToolUses []ToolUse `json:"tool_uses,omitempty"`
}

// ToolUse represents a parsed tool invocation from model output.
type ToolUse struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// Tool describes an available tool the model can invoke.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Handler     func(ctx context.Context, args map[string]any) (string, error)
}

// Result is the final output after the loop completes.
type Result struct {
	Response string    // final text from the model (tool blocks stripped)
	Messages []Message // full conversation history
	Turns    int       // number of LLM calls made
}
