// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSvc creates a setup Service for tests.
func testSvc() *Service {
	c := core.New()
	return &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}
}

func TestDetect_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	assert.Equal(t, TypeGo, Detect(dir))
	assert.Equal(t, []ProjectType{TypeGo}, DetectAll(dir))
}

func TestGenerateBuildConfig_Good(t *testing.T) {
	cfg, err := GenerateBuildConfig("/tmp/example", TypeGo)
	require.NoError(t, err)

	assert.Contains(t, cfg, "# example build configuration")
	assert.Contains(t, cfg, "project:")
	assert.Contains(t, cfg, "name: example")
	assert.Contains(t, cfg, "type: go")
	assert.Contains(t, cfg, "main: ./cmd/example")
	assert.Contains(t, cfg, "cgo: false")
}

func TestParseGitRemote_Good(t *testing.T) {
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

func TestParseGitRemote_Bad(t *testing.T) {
	assert.Equal(t, "", parseGitRemote(""))
	assert.Equal(t, "", parseGitRemote("origin"))
}

func TestRun_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	err := testSvc().Run(Options{Path: dir})
	require.NoError(t, err)

	build := fs.Read(core.JoinPath(dir, ".core", "build.yaml"))
	require.True(t, build.OK)
	assert.Contains(t, build.Value.(string), "type: go")

	test := fs.Read(core.JoinPath(dir, ".core", "test.yaml"))
	require.True(t, test.OK)
	assert.Contains(t, test.Value.(string), "go test ./...")
}

func TestRun_TemplateAlias_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	err := testSvc().Run(Options{Path: dir, Template: "agent"})
	require.NoError(t, err)

	prompt := fs.Read(core.JoinPath(dir, "PROMPT.md"))
	require.True(t, prompt.OK)
	assert.Contains(t, prompt.Value.(string), "This workspace was scaffolded by pkg/setup.")
}
