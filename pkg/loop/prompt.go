package loop

import (
	"encoding/json"
	"fmt"
	"strings"
)

// BuildSystemPrompt constructs the system prompt that instructs the model how
// to use the available tools. When no tools are registered it returns a plain
// assistant preamble without tool-calling instructions.
func BuildSystemPrompt(tools []Tool) string {
	if len(tools) == 0 {
		return "You are a helpful assistant."
	}

	var b strings.Builder
	b.WriteString("You are a helpful assistant with access to the following tools:\n\n")

	for _, tool := range tools {
		b.WriteString(fmt.Sprintf("### %s\n", tool.Name))
		b.WriteString(fmt.Sprintf("%s\n", tool.Description))
		if tool.Parameters != nil {
			schema, _ := json.MarshalIndent(tool.Parameters, "", "  ")
			b.WriteString(fmt.Sprintf("Parameters: %s\n", schema))
		}
		b.WriteString("\n")
	}

	b.WriteString("To use a tool, output a fenced block:\n")
	b.WriteString("```tool\n")
	b.WriteString("{\"name\": \"tool_name\", \"args\": {\"key\": \"value\"}}\n")
	b.WriteString("```\n\n")
	b.WriteString("You may call multiple tools in one response. After tool results are provided, continue reasoning. When you have a final answer, respond normally without tool blocks.\n")

	return b.String()
}

// BuildFullPrompt assembles the complete prompt string from the system prompt,
// conversation history, and current user message. Each message is tagged with
// its role so the model can distinguish turns. Tool results are annotated with
// the tool name for traceability.
func BuildFullPrompt(system string, history []Message, userMessage string) string {
	var b strings.Builder

	if system != "" {
		b.WriteString(system)
		b.WriteString("\n\n")
	}

	for _, msg := range history {
		switch msg.Role {
		case RoleUser:
			b.WriteString(fmt.Sprintf("[user]\n%s\n\n", msg.Content))
		case RoleAssistant:
			b.WriteString(fmt.Sprintf("[assistant]\n%s\n\n", msg.Content))
		case RoleToolResult:
			toolName := "unknown"
			if len(msg.ToolUses) > 0 {
				toolName = msg.ToolUses[0].Name
			}
			b.WriteString(fmt.Sprintf("[tool_result: %s]\n%s\n\n", toolName, msg.Content))
		}
	}

	if userMessage != "" {
		b.WriteString(fmt.Sprintf("[user]\n%s\n\n", userMessage))
	}

	return b.String()
}
