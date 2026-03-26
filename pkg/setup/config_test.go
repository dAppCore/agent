// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GenerateBuildConfig_Good(t *testing.T) {
	config, err := GenerateBuildConfig("/tmp/myapp", TypeGo)
	assert.NoError(t, err)
	assert.Contains(t, config, "type: go")
	assert.Contains(t, config, "name: myapp")
}

func TestConfig_GenerateBuildConfig_Bad_Unknown(t *testing.T) {
	config, err := GenerateBuildConfig("/tmp/myapp", TypeUnknown)
	assert.NoError(t, err)
	assert.NotEmpty(t, config)
}

func TestConfig_GenerateTestConfig_Good(t *testing.T) {
	config, err := GenerateTestConfig(TypeGo)
	assert.NoError(t, err)
	assert.Contains(t, config, "go test")
}

func TestConfig_ParseGitRemote_Good(t *testing.T) {
	assert.Equal(t, "core/go-io", parseGitRemote("https://forge.lthn.ai/core/go-io.git"))
	assert.Equal(t, "core/go-io", parseGitRemote("git@forge.lthn.ai:core/go-io.git"))
}

func TestConfig_ParseGitRemote_Ugly_Empty(t *testing.T) {
	assert.Equal(t, "", parseGitRemote(""))
}

func TestConfig_TrimRemotePath_Good(t *testing.T) {
	assert.Equal(t, "core/go-io", trimRemotePath("/core/go-io.git"))
}

func TestConfig_RenderConfig_Good(t *testing.T) {
	sections := []configSection{
		{Key: "project", Values: []configValue{{Key: "name", Value: "test"}}},
	}
	result, err := renderConfig("Test", sections)
	assert.NoError(t, err)
	assert.Contains(t, result, "name: test")
}

