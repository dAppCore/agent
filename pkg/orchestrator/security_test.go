package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizePath_Good(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple name", input: "myfile.txt", expected: "myfile.txt"},
		{name: "with hyphen", input: "my-file", expected: "my-file"},
		{name: "with underscore", input: "my_file", expected: "my_file"},
		{name: "with dots", input: "file.tar.gz", expected: "file.tar.gz"},
		{name: "strips directory", input: "/path/to/file.txt", expected: "file.txt"},
		{name: "alphanumeric", input: "abc123", expected: "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SanitizePath(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizePath_Good_StripsDirTraversal(t *testing.T) {
	// filepath.Base("../secret") returns "secret" which is safe.
	result, err := SanitizePath("../secret")
	require.NoError(t, err)
	assert.Equal(t, "secret", result, "directory traversal component stripped by filepath.Base")
}

func TestSanitizePath_Bad(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "spaces", input: "my file"},
		{name: "special chars", input: "file;rm -rf"},
		{name: "pipe", input: "file|cmd"},
		{name: "backtick", input: "file`cmd`"},
		{name: "dollar", input: "file$var"},
		{name: "single dot", input: "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SanitizePath(tt.input)
			assert.Error(t, err)
		})
	}
}

func TestEscapeShellArg_Good(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "simple string", input: "hello", expected: "'hello'"},
		{name: "with spaces", input: "hello world", expected: "'hello world'"},
		{name: "empty string", input: "", expected: "''"},
		{name: "with single quote", input: "it's", expected: "'it'\\''s'"},
		{name: "multiple single quotes", input: "a'b'c", expected: "'a'\\''b'\\''c'"},
		{name: "with special chars", input: "$(rm -rf /)", expected: "'$(rm -rf /)'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeShellArg(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecureSSHCommand_Good(t *testing.T) {
	cmd := SecureSSHCommand("claude@10.0.0.1", "ls -la /tmp")

	assert.Equal(t, "ssh", cmd.Path[len(cmd.Path)-3:])
	args := cmd.Args
	assert.Contains(t, args, "-o")
	assert.Contains(t, args, "StrictHostKeyChecking=yes")
	assert.Contains(t, args, "BatchMode=yes")
	assert.Contains(t, args, "ConnectTimeout=10")
	assert.Contains(t, args, "claude@10.0.0.1")
	assert.Contains(t, args, "ls -la /tmp")
}

func TestMaskToken_Good(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{name: "normal token", token: "abcdefghijkl", expected: "abcd****ijkl"},
		{name: "exactly 8 chars", token: "12345678", expected: "1234****5678"},
		{name: "short token", token: "abc", expected: "*****"},
		{name: "empty token", token: "", expected: "*****"},
		{name: "7 chars", token: "1234567", expected: "*****"},
		{name: "long token", token: "ghp_1234567890abcdef", expected: "ghp_****cdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}
