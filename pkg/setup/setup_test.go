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
	return &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
}

func TestSetup_Run_Good_WritesCoreConfigs(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	result := newSetupService().Run(Options{Path: dir})
	require.True(t, result.OK)

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

	result := newSetupService().Run(Options{Path: dir, Template: "agent"})
	require.True(t, result.OK)

	prompt := fs.Read(core.JoinPath(dir, "PROMPT.md"))
	require.True(t, prompt.OK)
	assert.Contains(t, prompt.Value.(string), "This workspace was scaffolded by pkg/setup.")
}

func TestSetup_Run_Bad_MissingTemplateDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	result := newSetupService().Run(Options{Path: dir, Template: "missing-template"})
	require.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.False(t, fs.Exists(core.JoinPath(dir, ".core")))
}

func TestSetup_Run_Ugly_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	result := newSetupService().Run(Options{Path: dir, Template: "agent", DryRun: true})
	require.True(t, result.OK)
	assert.False(t, fs.Exists(core.JoinPath(dir, ".core")))
	assert.False(t, fs.Exists(core.JoinPath(dir, "PROMPT.md")))
}

func TestSetup_ResolveTemplateName_Good_Auto(t *testing.T) {
	name := resolveTemplateName("auto", TypeGo)
	require.True(t, name.OK)
	assert.Equal(t, "default", name.Value.(string))
}

func TestSetup_ResolveTemplateName_Bad_Empty(t *testing.T) {
	result := resolveTemplateName("", TypeGo)
	require.False(t, result.OK)
	require.Error(t, result.Value.(error))
}

func TestSetup_ResolveTemplateName_Ugly_ConventionsAlias(t *testing.T) {
	name := resolveTemplateName("conventions", TypeGo)
	require.True(t, name.OK)
	assert.Equal(t, "review", name.Value.(string))
}

func TestSetup_TemplateExists_Good_Default(t *testing.T) {
	assert.True(t, templateExists("default"))
}

func TestSetup_TemplateExists_Bad_Missing(t *testing.T) {
	assert.False(t, templateExists("missing-template"))
}

func TestSetup_TemplateExists_Ugly_Review(t *testing.T) {
	assert.True(t, templateExists("review"))
}

func TestSetup_DefaultBuildCommand_Good_KnownTypes(t *testing.T) {
	assert.Equal(t, "go build ./...", defaultBuildCommand(TypeGo))
	assert.Equal(t, "composer test", defaultBuildCommand(TypePHP))
	assert.Equal(t, "npm run build", defaultBuildCommand(TypeNode))
}

func TestSetup_DefaultBuildCommand_Bad_Unknown(t *testing.T) {
	assert.Equal(t, "make build", defaultBuildCommand(TypeUnknown))
}

func TestSetup_DefaultBuildCommand_Ugly_WailsMatchesGo(t *testing.T) {
	assert.Equal(t, defaultBuildCommand(TypeGo), defaultBuildCommand(TypeWails))
}

func TestSetup_DefaultTestCommand_Good_KnownTypes(t *testing.T) {
	assert.Equal(t, "go test ./...", defaultTestCommand(TypeGo))
	assert.Equal(t, "composer test", defaultTestCommand(TypePHP))
	assert.Equal(t, "npm test", defaultTestCommand(TypeNode))
}

func TestSetup_DefaultTestCommand_Bad_Unknown(t *testing.T) {
	assert.Equal(t, "make test", defaultTestCommand(TypeUnknown))
}

func TestSetup_DefaultTestCommand_Ugly_WailsMatchesGo(t *testing.T) {
	assert.Equal(t, defaultTestCommand(TypeGo), defaultTestCommand(TypeWails))
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

func TestSetup_FormatFlow_Bad_Unknown(t *testing.T) {
	flow := formatFlow(TypeUnknown)
	assert.Contains(t, flow, "make build")
	assert.Contains(t, flow, "make test")
}

func TestSetup_FormatFlow_Ugly_Wails(t *testing.T) {
	assert.Equal(t, formatFlow(TypeGo), formatFlow(TypeWails))
}
