// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(filepath.Join(dir, "go.mod"), "module example.com/test\n", 0644).OK)

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

func TestRun_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(filepath.Join(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	err := Run(Options{Path: dir})
	require.NoError(t, err)

	build := fs.Read(filepath.Join(dir, ".core", "build.yaml"))
	require.True(t, build.OK)
	assert.Contains(t, build.Value.(string), "type: go")

	test := fs.Read(filepath.Join(dir, ".core", "test.yaml"))
	require.True(t, test.OK)
	assert.Contains(t, test.Value.(string), "go test ./...")
}

func TestRun_TemplateAlias_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(filepath.Join(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	err := Run(Options{Path: dir, Template: "agent"})
	require.NoError(t, err)

	prompt := fs.Read(filepath.Join(dir, "PROMPT.md"))
	require.True(t, prompt.OK)
	assert.Contains(t, prompt.Value.(string), "This workspace was scaffolded by pkg/setup.")
}
