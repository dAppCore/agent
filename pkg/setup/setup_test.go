// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	core "dappco.re/go"
)

func newSetupService() *Service {
	c := core.New()
	return &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
}

func TestWritesCoreConfigs_Service_Run_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	result := newSetupService().Run(Options{Path: dir})
	core.RequireTrue(t, result.OK)

	build := fs.Read(core.JoinPath(dir, ".core", "build.yaml"))
	core.RequireTrue(t, build.OK)
	core.AssertContains(t, build.Value.(string), "type: go")

	test := fs.Read(core.JoinPath(dir, ".core", "test.yaml"))
	core.RequireTrue(t, test.OK)
	core.AssertContains(t, test.Value.(string), "go test ./...")
}

func TestTemplateAlias_Service_Run_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	result := newSetupService().Run(Options{Path: dir, Template: "agent"})
	core.RequireTrue(t, result.OK)

	prompt := fs.Read(core.JoinPath(dir, "PROMPT.md"))
	core.RequireTrue(t, prompt.OK)
	core.AssertContains(t, prompt.Value.(string), "This workspace was scaffolded by pkg/setup.")
}

func TestMissingTemplate_Service_Run_Bad(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	result := newSetupService().Run(Options{Path: dir, Template: "missing-template"})
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertFalse(t, fs.Exists(core.JoinPath(dir, ".core")))
}

func TestDryRun_Service_Run_Ugly(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module example.com/test\n", 0644).OK)

	result := newSetupService().Run(Options{Path: dir, Template: "agent", DryRun: true})
	core.RequireTrue(t, result.OK)
	core.AssertFalse(t, fs.Exists(core.JoinPath(dir, ".core")))
	core.AssertFalse(t, fs.Exists(core.JoinPath(dir, "PROMPT.md")))
}

func TestSetup_ResolveTemplateName_Good_Auto(t *testing.T) {
	name := resolveTemplateName("auto", TypeGo)
	core.RequireTrue(t, name.OK)
	core.AssertEqual(t, "default", name.Value.(string))
}

func TestSetup_ResolveTemplateName_Bad_Empty(t *testing.T) {
	result := resolveTemplateName("", TypeGo)
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestSetup_ResolveTemplateName_Ugly_ConventionsAlias(t *testing.T) {
	name := resolveTemplateName("conventions", TypeGo)
	core.RequireTrue(t, name.OK)
	core.AssertEqual(t, "review", name.Value.(string))
}

func TestSetup_TemplateExists_Good_Default(t *testing.T) {
	ok := templateExists("default")
	core.AssertTrue(t, ok)
	core.AssertFalse(t, templateExists(""))
}

func TestSetup_TemplateExists_Bad_Missing(t *testing.T) {
	ok := templateExists("missing-template")
	core.AssertFalse(t, ok)
	core.AssertFalse(t, templateExists("missing-template/child"))
}

func TestSetup_TemplateExists_Ugly_Review(t *testing.T) {
	ok := templateExists("review")
	core.AssertTrue(t, ok)
	name := resolveTemplateName("conventions", TypeGo)
	core.RequireTrue(t, name.OK)
	core.AssertTrue(t, templateExists(name.Value.(string)))
}

func TestSetup_DefaultBuildCommand_Good_KnownTypes(t *testing.T) {
	core.AssertEqual(t, "go build ./...", defaultBuildCommand(TypeGo))
	core.AssertEqual(t, "composer test", defaultBuildCommand(TypePHP))
	core.AssertEqual(t, "npm run build", defaultBuildCommand(TypeNode))
}

func TestSetup_DefaultBuildCommand_Bad_Unknown(t *testing.T) {
	got := defaultBuildCommand(TypeUnknown)
	core.AssertEqual(t, "make build", got)
	core.AssertContains(t, got, "make")
}

func TestSetup_DefaultBuildCommand_Ugly_WailsMatchesGo(t *testing.T) {
	goCmd := defaultBuildCommand(TypeGo)
	wailsCmd := defaultBuildCommand(TypeWails)
	core.AssertEqual(t, goCmd, wailsCmd)
	core.AssertContains(t, wailsCmd, "go build")
}

func TestSetup_DefaultTestCommand_Good_KnownTypes(t *testing.T) {
	core.AssertEqual(t, "go test ./...", defaultTestCommand(TypeGo))
	core.AssertEqual(t, "composer test", defaultTestCommand(TypePHP))
	core.AssertEqual(t, "npm test", defaultTestCommand(TypeNode))
}

func TestSetup_DefaultTestCommand_Bad_Unknown(t *testing.T) {
	got := defaultTestCommand(TypeUnknown)
	core.AssertEqual(t, "make test", got)
	core.AssertContains(t, got, "make")
}

func TestSetup_DefaultTestCommand_Ugly_WailsMatchesGo(t *testing.T) {
	goCmd := defaultTestCommand(TypeGo)
	wailsCmd := defaultTestCommand(TypeWails)
	core.AssertEqual(t, goCmd, wailsCmd)
	core.AssertContains(t, wailsCmd, "go test")
}

func TestSetup_FormatFlow_Good(t *testing.T) {
	goFlow := formatFlow(TypeGo)
	core.AssertContains(t, goFlow, "go build ./...")
	core.AssertContains(t, goFlow, "go test ./...")

	phpFlow := formatFlow(TypePHP)
	core.AssertContains(t, phpFlow, "composer test")

	nodeFlow := formatFlow(TypeNode)
	core.AssertContains(t, nodeFlow, "npm run build")
	core.AssertContains(t, nodeFlow, "npm test")
}

func TestSetup_FormatFlow_Bad_Unknown(t *testing.T) {
	flow := formatFlow(TypeUnknown)
	core.AssertContains(t, flow, "make build")
	core.AssertContains(t, flow, "make test")
}

func TestSetup_FormatFlow_Ugly_Wails(t *testing.T) {
	goFlow := formatFlow(TypeGo)
	wailsFlow := formatFlow(TypeWails)
	core.AssertEqual(t, goFlow, wailsFlow)
	core.AssertContains(t, wailsFlow, "go build ./...")
}
