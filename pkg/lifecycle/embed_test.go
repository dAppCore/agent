package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrompt_Good_CommitExists(t *testing.T) {
	content := Prompt("commit")
	assert.NotEmpty(t, content, "commit prompt should exist")
	assert.Contains(t, content, "Commit")
}

func TestPrompt_Bad_NonexistentReturnsEmpty(t *testing.T) {
	content := Prompt("nonexistent-prompt-that-does-not-exist")
	assert.Empty(t, content, "nonexistent prompt should return empty string")
}

func TestPrompt_Good_ContentIsTrimmed(t *testing.T) {
	content := Prompt("commit")
	// Should not start or end with whitespace.
	assert.Equal(t, content[0:1] != " " && content[0:1] != "\n", true, "should not start with whitespace")
	lastChar := content[len(content)-1:]
	assert.Equal(t, lastChar != " " && lastChar != "\n", true, "should not end with whitespace")
}
