package loop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_Good_WithTools(t *testing.T) {
	tools := []Tool{
		{
			Name:        "file_read",
			Description: "Read a file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
				"required": []any{"path"},
			},
		},
		{
			Name:        "eaas_score",
			Description: "Score text for AI content",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{"type": "string"},
				},
			},
		},
	}

	prompt := BuildSystemPrompt(tools)
	assert.Contains(t, prompt, "file_read")
	assert.Contains(t, prompt, "Read a file")
	assert.Contains(t, prompt, "eaas_score")
	assert.Contains(t, prompt, "```tool")
}

func TestBuildSystemPrompt_Good_NoTools(t *testing.T) {
	prompt := BuildSystemPrompt(nil)
	assert.NotEmpty(t, prompt)
	assert.NotContains(t, prompt, "```tool")
}

func TestBuildFullPrompt_Good(t *testing.T) {
	history := []Message{
		{Role: RoleUser, Content: "hello"},
		{Role: RoleAssistant, Content: "hi there"},
	}
	prompt := BuildFullPrompt("system prompt", history, "what next?")
	assert.Contains(t, prompt, "system prompt")
	assert.Contains(t, prompt, "hello")
	assert.Contains(t, prompt, "hi there")
	assert.Contains(t, prompt, "what next?")
}

func TestBuildFullPrompt_Good_IncludesToolResults(t *testing.T) {
	history := []Message{
		{Role: RoleUser, Content: "read test.txt"},
		{Role: RoleAssistant, Content: "I'll read it.", ToolUses: []ToolUse{{Name: "file_read", Args: map[string]any{"path": "test.txt"}}}},
		{Role: RoleToolResult, Content: "file contents here", ToolUses: []ToolUse{{Name: "file_read"}}},
	}
	prompt := BuildFullPrompt("", history, "")
	assert.Contains(t, prompt, "[tool_result: file_read]")
	assert.Contains(t, prompt, "file contents here")
}
