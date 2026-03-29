// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GenerateBuildConfig_Good_Go(t *testing.T) {
	config, err := GenerateBuildConfig("/tmp/myapp", TypeGo)
	require.NoError(t, err)
	assert.Contains(t, config, "# myapp build configuration")
	assert.Contains(t, config, "type: go")
	assert.Contains(t, config, "name: myapp")
	assert.Contains(t, config, "main: ./cmd/myapp")
	assert.Contains(t, config, "cgo: false")
}

func TestConfig_GenerateBuildConfig_Bad_Unknown(t *testing.T) {
	config, err := GenerateBuildConfig("/tmp/myapp", TypeUnknown)
	require.NoError(t, err)
	assert.NotEmpty(t, config)
}

func TestConfig_GenerateBuildConfig_Ugly_WailsNestedPath(t *testing.T) {
	config, err := GenerateBuildConfig("/tmp/workspaces/team-console", TypeWails)
	require.NoError(t, err)
	assert.Contains(t, config, "name: team-console")
	assert.Contains(t, config, "type: wails")
	assert.Contains(t, config, "main: ./cmd/team-console")
}

func TestConfig_GenerateTestConfig_Good_Go(t *testing.T) {
	config, err := GenerateTestConfig(TypeGo)
	require.NoError(t, err)
	assert.Contains(t, config, "go test")
}

func TestConfig_GenerateTestConfig_Bad_Unknown(t *testing.T) {
	config, err := GenerateTestConfig(TypeUnknown)
	require.NoError(t, err)
	assert.Contains(t, config, "# Test configuration")
	assert.NotContains(t, config, "commands:")
}

func TestConfig_GenerateTestConfig_Ugly_WailsUsesGoSuite(t *testing.T) {
	config, err := GenerateTestConfig(TypeWails)
	require.NoError(t, err)
	assert.Contains(t, config, "go test ./...")
	assert.Contains(t, config, "go test -race ./...")
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
	result, err := renderConfig("Test", sections)
	assert.NoError(t, err)
	assert.Contains(t, result, "name: test")
}

func TestConfig_RenderConfig_Bad_UnsupportedValue(t *testing.T) {
	sections := []configSection{
		{Key: "project", Values: []configValue{{Key: "name", Value: func() {}}}},
	}
	result, err := renderConfig("Test", sections)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestConfig_RenderConfig_Ugly_EmptySections(t *testing.T) {
	result, err := renderConfig("", nil)
	assert.NoError(t, err)
	assert.Equal(t, "", result)
}
