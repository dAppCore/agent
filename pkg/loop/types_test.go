package loop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_Good_UserMessage(t *testing.T) {
	m := Message{Role: RoleUser, Content: "hello"}
	assert.Equal(t, RoleUser, m.Role)
	assert.Equal(t, "hello", m.Content)
	assert.Nil(t, m.ToolUses)
}

func TestMessage_Good_AssistantWithTools(t *testing.T) {
	m := Message{
		Role:    RoleAssistant,
		Content: "I'll read that file.",
		ToolUses: []ToolUse{
			{Name: "file_read", Args: map[string]any{"path": "/tmp/test.txt"}},
		},
	}
	assert.Len(t, m.ToolUses, 1)
	assert.Equal(t, "file_read", m.ToolUses[0].Name)
}

func TestTool_Good_HasHandler(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  map[string]any{"type": "object"},
	}
	assert.Equal(t, "test_tool", tool.Name)
	assert.NotEmpty(t, tool.Description)
}

func TestResult_Good_Fields(t *testing.T) {
	r := Result{
		Response: "done",
		Turns:    3,
	}
	assert.Equal(t, "done", r.Response)
	assert.Equal(t, 3, r.Turns)
}
