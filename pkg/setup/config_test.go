// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	core "dappco.re/go"
)

func TestGo_GenerateBuildConfig_Good(t *testing.T) {
	config := GenerateBuildConfig("/tmp/myapp", TypeGo)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "# myapp build configuration")
	core.AssertContains(t, text, "type: go")
	core.AssertContains(t, text, "name: myapp")
	core.AssertContains(t, text, "main: ./cmd/myapp")
	core.AssertContains(t, text, "cgo: false")
}

func TestUnknown_GenerateBuildConfig_Bad(t *testing.T) {
	config := GenerateBuildConfig("/tmp/myapp", TypeUnknown)
	core.RequireTrue(t, config.OK)
	core.AssertNotEmpty(t, config.Value.(string))
}

func TestWailsNestedPath_GenerateBuildConfig_Ugly(t *testing.T) {
	config := GenerateBuildConfig("/tmp/workspaces/team-console", TypeWails)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "name: team-console")
	core.AssertContains(t, text, "type: wails")
	core.AssertContains(t, text, "main: ./cmd/team-console")
}

func TestGo_GenerateTestConfig_Good(t *testing.T) {
	config := GenerateTestConfig(TypeGo)
	core.RequireTrue(t, config.OK)
	core.AssertContains(t, config.Value.(string), "go test")
}

func TestUnknown_GenerateTestConfig_Bad(t *testing.T) {
	config := GenerateTestConfig(TypeUnknown)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "# Test configuration")
	core.AssertNotContains(t, text, "commands:")
}

func TestWailsSuite_GenerateTestConfig_Ugly(t *testing.T) {
	config := GenerateTestConfig(TypeWails)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "go test ./...")
	core.AssertContains(t, text, "go test -race ./...")
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
		core.AssertEqual(t, want, parseGitRemote(remote), remote)
	}
}

func TestConfig_ParseGitRemote_Bad_Empty(t *testing.T) {
	core.AssertEqual(t, "", parseGitRemote(""))
	core.AssertEqual(t, "", parseGitRemote("origin"))
	core.AssertEqual(t, "", parseGitRemote("   "))
}

func TestConfig_ParseGitRemote_Ugly_WhitespaceAndTrailingSlash(t *testing.T) {
	remote := "  https://github.com/dAppCore/go-io.git/  "
	got := parseGitRemote(remote)
	core.AssertEqual(t, "dAppCore/go-io", got)
	core.AssertNotContains(t, got, ".git")
}

func TestConfig_TrimRemotePath_Good_Case(t *testing.T) {
	got := trimRemotePath("/core/go-io.git")
	core.AssertEqual(t, "core/go-io", got)
	core.AssertNotContains(t, got, ".git")
}

func TestConfig_TrimRemotePath_Bad_Empty(t *testing.T) {
	got := trimRemotePath("")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestConfig_TrimRemotePath_Ugly_RepeatedSlashes(t *testing.T) {
	got := trimRemotePath("///core/go-io.git///")
	core.AssertEqual(t, "core/go-io", got)
	core.AssertNotContains(t, got, "//")
}

func TestConfig_RenderConfig_Good_SingleSection(t *testing.T) {
	sections := []configSection{
		{Key: "project", Values: []configValue{{Key: "name", Value: "test"}}},
	}
	result := renderConfig("Test", sections)
	core.RequireTrue(t, result.OK)
	core.AssertContains(t, result.Value.(string), "name: test")
}

func TestConfig_RenderConfig_Bad_UnsupportedValue(t *testing.T) {
	sections := []configSection{
		{Key: "project", Values: []configValue{{Key: "name", Value: func() {}}}},
	}
	result := renderConfig("Test", sections)
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestConfig_RenderConfig_Ugly_EmptySections(t *testing.T) {
	result := renderConfig("", nil)
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, "", result.Value.(string))
}

func TestConfig_GenerateBuildConfig_Good(t *testing.T) {
	config := GenerateBuildConfig("/tmp/myapp", TypeGo)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "# myapp build configuration")
	core.AssertContains(t, text, "type: go")
	core.AssertContains(t, text, "name: myapp")
	core.AssertContains(t, text, "main: ./cmd/myapp")
	core.AssertContains(t, text, "cgo: false")
}

func TestConfig_GenerateBuildConfig_Bad(t *testing.T) {
	config := GenerateBuildConfig("/tmp/myapp", TypeUnknown)
	core.RequireTrue(t, config.OK)
	core.AssertNotEmpty(t, config.Value.(string))
}

func TestConfig_GenerateBuildConfig_Ugly(t *testing.T) {
	config := GenerateBuildConfig("/tmp/workspaces/team-console", TypeWails)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "name: team-console")
	core.AssertContains(t, text, "type: wails")
	core.AssertContains(t, text, "main: ./cmd/team-console")
}

func TestConfig_GenerateTestConfig_Good(t *testing.T) {
	config := GenerateTestConfig(TypeGo)
	core.RequireTrue(t, config.OK)
	core.AssertContains(t, config.Value.(string), "go test")
}

func TestConfig_GenerateTestConfig_Bad(t *testing.T) {
	config := GenerateTestConfig(TypeUnknown)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "# Test configuration")
	core.AssertNotContains(t, text, "commands:")
}

func TestConfig_GenerateTestConfig_Ugly(t *testing.T) {
	config := GenerateTestConfig(TypeWails)
	core.RequireTrue(t, config.OK)
	text := config.Value.(string)
	core.AssertContains(t, text, "go test ./...")
	core.AssertContains(t, text, "go test -race ./...")
}
