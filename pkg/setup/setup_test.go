// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSetupService() *Service {
	c := core.New()
	return &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}
}

func TestSetup_Run_Good_WritesCoreConfigs(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	err := newSetupService().Run(Options{Path: dir})
	require.NoError(t, err)

	build := fs.Read(core.JoinPath(dir, ".core", "build.yaml"))
	require.True(t, build.OK)
	assert.Contains(t, build.Value.(string), "type: go")

	test := fs.Read(core.JoinPath(dir, ".core", "test.yaml"))
	require.True(t, test.OK)
	assert.Contains(t, test.Value.(string), "go test ./...")
}

func TestSetup_Run_Good_TemplateAlias(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	err := newSetupService().Run(Options{Path: dir, Template: "agent"})
	require.NoError(t, err)

	prompt := fs.Read(core.JoinPath(dir, "PROMPT.md"))
	require.True(t, prompt.OK)
	assert.Contains(t, prompt.Value.(string), "This workspace was scaffolded by pkg/setup.")
}

func TestSetup_ResolveTemplateName_Good_Auto(t *testing.T) {
	name, err := resolveTemplateName("auto", TypeGo)
	require.NoError(t, err)
	assert.Equal(t, "default", name)
}

func TestSetup_ResolveTemplateName_Bad_Empty(t *testing.T) {
	_, err := resolveTemplateName("", TypeGo)
	require.Error(t, err)
}

func TestSetup_TemplateExists_Good_Default(t *testing.T) {
	assert.True(t, templateExists("default"))
}

func TestSetup_TemplateExists_Bad_Missing(t *testing.T) {
	assert.False(t, templateExists("missing-template"))
}

func TestSetup_DefaultBuildCommand_Good(t *testing.T) {
	assert.Equal(t, "go build ./...", defaultBuildCommand(TypeGo))
	assert.Equal(t, "go build ./...", defaultBuildCommand(TypeWails))
	assert.Equal(t, "composer test", defaultBuildCommand(TypePHP))
	assert.Equal(t, "npm run build", defaultBuildCommand(TypeNode))
	assert.Equal(t, "make build", defaultBuildCommand(TypeUnknown))
}

func TestSetup_DefaultTestCommand_Good(t *testing.T) {
	assert.Equal(t, "go test ./...", defaultTestCommand(TypeGo))
	assert.Equal(t, "go test ./...", defaultTestCommand(TypeWails))
	assert.Equal(t, "composer test", defaultTestCommand(TypePHP))
	assert.Equal(t, "npm test", defaultTestCommand(TypeNode))
	assert.Equal(t, "make test", defaultTestCommand(TypeUnknown))
}

func TestSetup_FormatFlow_Good(t *testing.T) {
	goFlow := formatFlow(TypeGo)
	assert.Contains(t, goFlow, "go build ./...")
	assert.Contains(t, goFlow, "go test ./...")

	phpFlow := formatFlow(TypePHP)
	assert.Contains(t, phpFlow, "composer test")

	nodeFlow := formatFlow(TypeNode)
	assert.Contains(t, nodeFlow, "npm run build")
	assert.Contains(t, nodeFlow, "npm test")
}
