// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
)

// Options controls one setup run.
//
//	err := svc.Run(setup.Options{Path: ".", Template: "auto", Force: true})
type Options struct {
	Path     string // Target directory (default: cwd)
	DryRun   bool   // Preview only, don't write
	Force    bool   // Overwrite existing files
	Template string // Workspace template or compatibility alias (default, review, security, agent, go, php, gui, auto)
}

// Run generates `.core/` files and optional workspace scaffolding for a repo.
//
//	err := svc.Run(setup.Options{Path: ".", Template: "auto"})
func (s *Service) Run(opts Options) error {
	if opts.Path == "" {
		opts.Path = core.Env("DIR_CWD")
	}
	opts.Path = absolutePath(opts.Path)

	projType := Detect(opts.Path)
	allTypes := DetectAll(opts.Path)

	core.Print(nil, "Project: %s", core.PathBase(opts.Path))
	core.Print(nil, "Type:    %s", projType)
	if len(allTypes) > 1 {
		core.Print(nil, "Also:    %v (polyglot)", allTypes)
	}

	// Generate .core/ config files
	if err := setupCoreDir(opts, projType); err != nil {
		return err
	}

	// Scaffold from dir template if requested
	if opts.Template != "" {
		return s.scaffoldTemplate(opts, projType)
	}

	return nil
}

// setupCoreDir creates .core/ with build.yaml and test.yaml.
func setupCoreDir(opts Options, projType ProjectType) error {
	coreDir := core.JoinPath(opts.Path, ".core")

	if opts.DryRun {
		core.Print(nil, "")
		core.Print(nil, "Would create %s/", coreDir)
	} else {
		if r := fs.EnsureDir(coreDir); !r.OK {
			err, _ := r.Value.(error)
			return core.E("setup.setupCoreDir", "create .core directory", err)
		}
	}

	// build.yaml
	buildConfig, err := GenerateBuildConfig(opts.Path, projType)
	if err != nil {
		return core.E("setup.setupCoreDir", "generate build config", err)
	}
	if err := writeConfig(core.JoinPath(coreDir, "build.yaml"), buildConfig, opts); err != nil {
		return err
	}

	// test.yaml
	testConfig, err := GenerateTestConfig(projType)
	if err != nil {
		return core.E("setup.setupCoreDir", "generate test config", err)
	}
	if err := writeConfig(core.JoinPath(coreDir, "test.yaml"), testConfig, opts); err != nil {
		return err
	}

	return nil
}

// scaffoldTemplate extracts a dir template into the target path.
func (s *Service) scaffoldTemplate(opts Options, projType ProjectType) error {
	tmplName, err := resolveTemplateName(opts.Template, projType)
	if err != nil {
		return err
	}

	core.Print(nil, "Template: %s", tmplName)

	data := &lib.WorkspaceData{
		Repo:            core.PathBase(opts.Path),
		Branch:          "main",
		Task:            core.Sprintf("Initialise %s project tooling.", projType),
		Agent:           "setup",
		Language:        string(projType),
		Prompt:          "This workspace was scaffolded by pkg/setup. Review the repository and continue from the generated context files.",
		Flow:            formatFlow(projType),
		RepoDescription: s.DetectGitRemote(opts.Path),
		BuildCmd:        defaultBuildCommand(projType),
		TestCmd:         defaultTestCommand(projType),
	}

	if !templateExists(tmplName) {
		return core.E("setup.scaffoldTemplate", core.Concat("template not found: ", tmplName), nil)
	}

	if opts.DryRun {
		core.Print(nil, "Would extract workspace/%s to %s", tmplName, opts.Path)
		core.Print(nil, "  Template found: %s", tmplName)
		return nil
	}

	if err := lib.ExtractWorkspace(tmplName, opts.Path, data); err != nil {
		return core.E("setup.scaffoldTemplate", core.Concat("extract workspace template ", tmplName), err)
	}
	return nil
}

func writeConfig(path, content string, opts Options) error {
	if opts.DryRun {
		core.Print(nil, "  %s", path)
		return nil
	}

	if !opts.Force && fs.Exists(path) {
		core.Print(nil, "  skip %s (exists, use --force to overwrite)", core.PathBase(path))
		return nil
	}

	if r := fs.WriteMode(path, content, 0644); !r.OK {
		err, _ := r.Value.(error)
		return core.E("setup.writeConfig", core.Concat("write ", core.PathBase(path)), err)
	}
	core.Print(nil, "  created %s", path)
	return nil
}

func resolveTemplateName(name string, projType ProjectType) (string, error) {
	if name == "" {
		return "", core.E("setup.resolveTemplateName", "template is required", nil)
	}

	if name == "auto" {
		switch projType {
		case TypeGo, TypeWails, TypePHP, TypeNode, TypeUnknown:
			return "default", nil
		}
	}

	switch name {
	case "agent", "go", "php", "gui":
		return "default", nil
	case "verify", "conventions":
		return "review", nil
	default:
		return name, nil
	}
}

func templateExists(name string) bool {
	for _, tmpl := range lib.ListWorkspaces() {
		if tmpl == name {
			return true
		}
	}
	return false
}

func defaultBuildCommand(projType ProjectType) string {
	switch projType {
	case TypeGo, TypeWails:
		return "go build ./..."
	case TypePHP:
		return "composer test"
	case TypeNode:
		return "npm run build"
	default:
		return "make build"
	}
}

func defaultTestCommand(projType ProjectType) string {
	switch projType {
	case TypeGo, TypeWails:
		return "go test ./..."
	case TypePHP:
		return "composer test"
	case TypeNode:
		return "npm test"
	default:
		return "make test"
	}
}

func formatFlow(projType ProjectType) string {
	builder := core.NewBuilder()
	builder.WriteString("- Build: `")
	builder.WriteString(defaultBuildCommand(projType))
	builder.WriteString("`\n")
	builder.WriteString("- Test: `")
	builder.WriteString(defaultTestCommand(projType))
	builder.WriteString("`")
	return builder.String()
}
