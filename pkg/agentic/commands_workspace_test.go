// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- extractField ---

func TestExtractField_Good_SimpleJSON(t *testing.T) {
	json := `{"status":"running","repo":"go-io","agent":"codex"}`
	assert.Equal(t, "running", extractField(json, "status"))
	assert.Equal(t, "go-io", extractField(json, "repo"))
	assert.Equal(t, "codex", extractField(json, "agent"))
}

func TestExtractField_Good_PrettyPrinted(t *testing.T) {
	json := `{
  "status": "completed",
  "repo": "go-crypt"
}`
	assert.Equal(t, "completed", extractField(json, "status"))
	assert.Equal(t, "go-crypt", extractField(json, "repo"))
}

func TestExtractField_Good_TabSeparated(t *testing.T) {
	json := `{"status":	"blocked"}`
	assert.Equal(t, "blocked", extractField(json, "status"))
}

func TestExtractField_Bad_MissingField(t *testing.T) {
	json := `{"status":"running"}`
	assert.Empty(t, extractField(json, "nonexistent"))
}

func TestExtractField_Bad_EmptyJSON(t *testing.T) {
	assert.Empty(t, extractField("", "status"))
	assert.Empty(t, extractField("{}", "status"))
}

func TestExtractField_Bad_NoValue(t *testing.T) {
	// Field key exists but no quoted value after colon
	json := `{"status": 42}`
	assert.Empty(t, extractField(json, "status"))
}

func TestExtractField_Bad_TruncatedJSON(t *testing.T) {
	// Field key exists but string is truncated
	json := `{"status":`
	assert.Empty(t, extractField(json, "status"))
}

func TestExtractField_Good_EmptyValue(t *testing.T) {
	json := `{"status":""}`
	assert.Equal(t, "", extractField(json, "status"))
}

func TestExtractField_Good_ValueWithSpaces(t *testing.T) {
	json := `{"task":"fix the failing tests"}`
	assert.Equal(t, "fix the failing tests", extractField(json, "task"))
}
