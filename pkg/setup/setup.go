// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
)

// Options controls one setup run.
//
//	r := svc.Run(setup.Options{Path: ".", Template: "auto", Force: true})
//	if !r.OK { core.Print(nil, "%v", r.Value) }
type Options struct {
	Path     string // Target directory (default: cwd)
	DryRun   bool   // Preview only, don't write
	Force    bool   // Overwrite existing files
	Template string // Workspace template or compatibility alias (default, review, security, agent, go, php, gui, auto)
}

// Run generates `.core/` files and optional workspace scaffolding for a repo.
//
//	r := svc.Run(setup.Options{Path: ".", Template: "auto"})
//	core.Println(r.OK)
func (s *Service) Run(opts Options) core.Result {
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

	var tmplName string
	if opts.Template != "" {
		templateResult := resolveTemplateName(opts.Template, projType)
		if !templateResult.OK {
			return templateResult
		}
		tmplName = templateResult.Value.(string)
		if !templateExists(tmplName) {
			return core.Result{
				Value: core.E("setup.Run", core.Concat("template not found: ", tmplName), nil),
				OK:    false,
			}
		}
	}

	// Generate .core/ config files
	if result := setupCoreDir(opts, projType); !result.OK {
		return result
	}

	// Scaffold from dir template if requested
	if tmplName != "" {
		return s.scaffoldTemplate(opts, projType, tmplName)
	}

	return core.Result{Value: opts.Path, OK: true}
}

// setupCoreDir creates .core/ with build.yaml and test.yaml.
func setupCoreDir(opts Options, projType ProjectType) core.Result {
	coreDir := core.JoinPath(opts.Path, ".core")

	if opts.DryRun {
		core.Print(nil, "")
		core.Print(nil, "Would create %s/", coreDir)
	} else {
		if r := fs.EnsureDir(coreDir); !r.OK {
			err, _ := r.Value.(error)
			return core.Result{
				Value: core.E("setup.setupCoreDir", "create .core directory", err),
				OK:    false,
			}
		}
	}

	// build.yaml
	buildConfig := GenerateBuildConfig(opts.Path, projType)
	if !buildConfig.OK {
		err, _ := buildConfig.Value.(error)
		return core.Result{
			Value: core.E("setup.setupCoreDir", "generate build config", err),
			OK:    false,
		}
	}
	if result := writeConfig(core.JoinPath(coreDir, "build.yaml"), buildConfig.Value.(string), opts); !result.OK {
		return result
	}

	// test.yaml
	testConfig := GenerateTestConfig(projType)
	if !testConfig.OK {
		err, _ := testConfig.Value.(error)
		return core.Result{
			Value: core.E("setup.setupCoreDir", "generate test config", err),
			OK:    false,
		}
	}
	if result := writeConfig(core.JoinPath(coreDir, "test.yaml"), testConfig.Value.(string), opts); !result.OK {
		return result
	}

	return core.Result{Value: coreDir, OK: true}
}

// scaffoldTemplate extracts a dir template into the target path.
func (s *Service) scaffoldTemplate(opts Options, projType ProjectType, tmplName string) core.Result {
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

	if opts.DryRun {
		core.Print(nil, "Would extract workspace/%s to %s", tmplName, opts.Path)
		core.Print(nil, "  Template found: %s", tmplName)
		return core.Result{Value: opts.Path, OK: true}
	}

	if err := lib.ExtractWorkspace(tmplName, opts.Path, data); err != nil {
		return core.Result{
			Value: core.E("setup.scaffoldTemplate", core.Concat("extract workspace template ", tmplName), err),
			OK:    false,
		}
	}
	return core.Result{Value: opts.Path, OK: true}
}

func writeConfig(path, content string, opts Options) core.Result {
	if opts.DryRun {
		core.Print(nil, "  %s", path)
		return core.Result{Value: path, OK: true}
	}

	if !opts.Force && fs.Exists(path) {
		core.Print(nil, "  skip %s (exists, use --force to overwrite)", core.PathBase(path))
		return core.Result{Value: path, OK: true}
	}

	if r := fs.WriteMode(path, content, 0644); !r.OK {
		err, _ := r.Value.(error)
		return core.Result{
			Value: core.E("setup.writeConfig", core.Concat("write ", core.PathBase(path)), err),
			OK:    false,
		}
	}
	core.Print(nil, "  created %s", path)
	return core.Result{Value: path, OK: true}
}

func resolveTemplateName(name string, projType ProjectType) core.Result {
	if name == "" {
		return core.Result{
			Value: core.E("setup.resolveTemplateName", "template is required", nil),
			OK:    false,
		}
	}

	if name == "auto" {
		switch projType {
		case TypeGo, TypeWails, TypePHP, TypeNode, TypeUnknown:
			return core.Result{Value: "default", OK: true}
		}
	}

	switch name {
	case "agent", "go", "php", "gui":
		return core.Result{Value: "default", OK: true}
	case "verify", "conventions":
		return core.Result{Value: "review", OK: true}
	default:
		return core.Result{Value: name, OK: true}
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
