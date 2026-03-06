package loop

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTool_Good_SingleCall(t *testing.T) {
	input := "Let me read that file.\n```tool\n{\"name\": \"file_read\", \"args\": {\"path\": \"/tmp/test.txt\"}}\n```\n"
	calls, text := ParseToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "file_read", calls[0].Name)
	assert.Equal(t, "/tmp/test.txt", calls[0].Args["path"])
	assert.Contains(t, text, "Let me read that file.")
	assert.NotContains(t, text, "```tool")
}

func TestParseTool_Good_MultipleCalls(t *testing.T) {
	input := "I'll check both.\n```tool\n{\"name\": \"file_read\", \"args\": {\"path\": \"a.txt\"}}\n```\nAnd also:\n```tool\n{\"name\": \"file_read\", \"args\": {\"path\": \"b.txt\"}}\n```\n"
	calls, _ := ParseToolCalls(input)
	require.Len(t, calls, 2)
	assert.Equal(t, "a.txt", calls[0].Args["path"])
	assert.Equal(t, "b.txt", calls[1].Args["path"])
}

func TestParseTool_Good_NoToolCalls(t *testing.T) {
	input := "Here is a normal response with no tool calls."
	calls, text := ParseToolCalls(input)
	assert.Empty(t, calls)
	assert.Equal(t, input, text)
}

func TestParseTool_Bad_MalformedJSON(t *testing.T) {
	input := "```tool\n{not valid json}\n```\n"
	calls, _ := ParseToolCalls(input)
	assert.Empty(t, calls)
}

func TestParseTool_Good_WithSurroundingText(t *testing.T) {
	input := "Before text.\n```tool\n{\"name\": \"test\", \"args\": {}}\n```\nAfter text."
	calls, text := ParseToolCalls(input)
	require.Len(t, calls, 1)
	assert.Contains(t, text, "Before text.")
	assert.Contains(t, text, "After text.")
}

func TestParseTool_Ugly_NestedBackticks(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```\n```tool\n{\"name\": \"test\", \"args\": {}}\n```\n"
	calls, text := ParseToolCalls(input)
	require.Len(t, calls, 1)
	assert.Equal(t, "test", calls[0].Name)
	assert.Contains(t, text, "```go")
}

func TestParseTool_Bad_EmptyToolBlock(t *testing.T) {
	input := "```tool\n\n```\n"
	calls, _ := ParseToolCalls(input)
	assert.Empty(t, calls)
}

func TestParseTool_Good_ArgsWithNestedObject(t *testing.T) {
	input := "```tool\n{\"name\": \"complex\", \"args\": {\"config\": {\"key\": \"value\", \"num\": 42}}}\n```\n"
	calls, _ := ParseToolCalls(input)
	require.Len(t, calls, 1)
	config, ok := calls[0].Args["config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "value", config["key"])
}
