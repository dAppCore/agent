// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GenerateBuildConfig_Good_Go(t *testing.T) {
	config := GenerateBuildConfig("/tmp/myapp", TypeGo)
	require.True(t, config.OK)
	text := config.Value.(string)
	assert.Contains(t, text, "# myapp build configuration")
	assert.Contains(t, text, "type: go")
	assert.Contains(t, text, "name: myapp")
	assert.Contains(t, text, "main: ./cmd/myapp")
	assert.Contains(t, text, "cgo: false")
}

func TestConfig_GenerateBuildConfig_Bad_Unknown(t *testing.T) {
	config := GenerateBuildConfig("/tmp/myapp", TypeUnknown)
	require.True(t, config.OK)
	assert.NotEmpty(t, config.Value.(string))
}

func TestConfig_GenerateBuildConfig_Ugly_WailsNestedPath(t *testing.T) {
	config := GenerateBuildConfig("/tmp/workspaces/team-console", TypeWails)
	require.True(t, config.OK)
	text := config.Value.(string)
	assert.Contains(t, text, "name: team-console")
	assert.Contains(t, text, "type: wails")
	assert.Contains(t, text, "main: ./cmd/team-console")
}

func TestConfig_GenerateTestConfig_Good_Go(t *testing.T) {
	config := GenerateTestConfig(TypeGo)
	require.True(t, config.OK)
	assert.Contains(t, config.Value.(string), "go test")
}

func TestConfig_GenerateTestConfig_Bad_Unknown(t *testing.T) {
	config := GenerateTestConfig(TypeUnknown)
	require.True(t, config.OK)
	text := config.Value.(string)
	assert.Contains(t, text, "# Test configuration")
	assert.NotContains(t, text, "commands:")
}

func TestConfig_GenerateTestConfig_Ugly_WailsUsesGoSuite(t *testing.T) {
	config := GenerateTestConfig(TypeWails)
	require.True(t, config.OK)
	text := config.Value.(string)
	assert.Contains(t, text, "go test ./...")
	assert.Contains(t, text, "go test -race ./...")
}

func TestConfig_ParseGitRemote_Good_CommonFormats(t *testing.T) {
	tests := map[string]string{
		"https://github.com/dAppCore/go-io.git":       "dAppCore/go-io",
		"git@github.com:dAppCore/go-io.git":           "dAppCore/go-io",
		"ssh://git@forge.lthn.ai:2223/core/agent.git": "core/agent",
		"ssh://git@forge.lthn.ai:2223/core/agent":     "core/agent",
		"git@forge.lthn.ai:core/agent.git":            "core/agent",
		"/srv/git/core/agent.git":                     "srv/git/core/agent",
	}

	for remote, want := range tests {
		assert.Equal(t, want, parseGitRemote(remote), remote)
	}
}

func TestConfig_ParseGitRemote_Bad_Empty(t *testing.T) {
	assert.Equal(t, "", parseGitRemote(""))
	assert.Equal(t, "", parseGitRemote("origin"))
}

func TestConfig_ParseGitRemote_Ugly_WhitespaceAndTrailingSlash(t *testing.T) {
	remote := "  https://github.com/dAppCore/go-io.git/  "
	assert.Equal(t, "dAppCore/go-io", parseGitRemote(remote))
}

func TestConfig_TrimRemotePath_Good(t *testing.T) {
	assert.Equal(t, "core/go-io", trimRemotePath("/core/go-io.git"))
}

func TestConfig_TrimRemotePath_Bad_Empty(t *testing.T) {
	assert.Equal(t, "", trimRemotePath(""))
}

func TestConfig_TrimRemotePath_Ugly_RepeatedSlashes(t *testing.T) {
	assert.Equal(t, "core/go-io", trimRemotePath("///core/go-io.git///"))
}

func TestConfig_RenderConfig_Good_SingleSection(t *testing.T) {
	sections := []configSection{
		{Key: "project", Values: []configValue{{Key: "name", Value: "test"}}},
	}
	result := renderConfig("Test", sections)
	require.True(t, result.OK)
	assert.Contains(t, result.Value.(string), "name: test")
}

func TestConfig_RenderConfig_Bad_UnsupportedValue(t *testing.T) {
	sections := []configSection{
		{Key: "project", Values: []configValue{{Key: "name", Value: func() {}}}},
	}
	result := renderConfig("Test", sections)
	assert.False(t, result.OK)
	assert.Error(t, result.Value.(error))
}

func TestConfig_RenderConfig_Ugly_EmptySections(t *testing.T) {
	result := renderConfig("", nil)
	require.True(t, result.OK)
	assert.Equal(t, "", result.Value.(string))
}
