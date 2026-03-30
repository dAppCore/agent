// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
)

// Options controls one setup run.
//
//	result := service.Run(setup.Options{Path: ".", Template: "auto", Force: true})
//	if !result.OK { core.Print(nil, "%v", result.Value) }
type Options struct {
	Path     string // Target directory (default: cwd)
	DryRun   bool   // Preview only, don't write
	Force    bool   // Overwrite existing files
	Template string // Workspace template or compatibility alias (default, review, security, agent, go, php, gui, auto)
}

// Run generates `.core/` files and optional workspace scaffolding for a repo.
//
//	result := service.Run(setup.Options{Path: ".", Template: "auto"})
//	core.Println(result.OK)
func (s *Service) Run(options Options) core.Result {
	if options.Path == "" {
		options.Path = core.Env("DIR_CWD")
	}
	options.Path = absolutePath(options.Path)

	projType := Detect(options.Path)
	allTypes := DetectAll(options.Path)

	core.Print(nil, "Project: %s", core.PathBase(options.Path))
	core.Print(nil, "Type:    %s", projType)
	if len(allTypes) > 1 {
		core.Print(nil, "Also:    %v (polyglot)", allTypes)
	}

	var tmplName string
	if options.Template != "" {
		templateResult := resolveTemplateName(options.Template, projType)
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
	if result := setupCoreDir(options, projType); !result.OK {
		return result
	}

	// Scaffold from dir template if requested
	if tmplName != "" {
		return s.scaffoldTemplate(options, projType, tmplName)
	}

	return core.Result{Value: options.Path, OK: true}
}

// setupCoreDir creates .core/ with build.yaml and test.yaml.
func setupCoreDir(options Options, projType ProjectType) core.Result {
	coreDir := core.JoinPath(options.Path, ".core")

	if options.DryRun {
		core.Print(nil, "")
		core.Print(nil, "Would create %s/", coreDir)
	} else {
		if ensureResult := fs.EnsureDir(coreDir); !ensureResult.OK {
			err, _ := ensureResult.Value.(error)
			return core.Result{
				Value: core.E("setup.setupCoreDir", "create .core directory", err),
				OK:    false,
			}
		}
	}

	// build.yaml
	buildConfig := GenerateBuildConfig(options.Path, projType)
	if !buildConfig.OK {
		err, _ := buildConfig.Value.(error)
		return core.Result{
			Value: core.E("setup.setupCoreDir", "generate build config", err),
			OK:    false,
		}
	}
	if result := writeConfig(core.JoinPath(coreDir, "build.yaml"), buildConfig.Value.(string), options); !result.OK {
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
	if result := writeConfig(core.JoinPath(coreDir, "test.yaml"), testConfig.Value.(string), options); !result.OK {
		return result
	}

	return core.Result{Value: coreDir, OK: true}
}

// scaffoldTemplate extracts a dir template into the target path.
func (s *Service) scaffoldTemplate(options Options, projType ProjectType, tmplName string) core.Result {
	core.Print(nil, "Template: %s", tmplName)

	data := &lib.WorkspaceData{
		Repo:            core.PathBase(options.Path),
		Branch:          "main",
		Task:            core.Sprintf("Initialise %s project tooling.", projType),
		Agent:           "setup",
		Language:        string(projType),
		Prompt:          "This workspace was scaffolded by pkg/setup. Review the repository and continue from the generated context files.",
		Flow:            formatFlow(projType),
		RepoDescription: s.DetectGitRemote(options.Path),
		BuildCmd:        defaultBuildCommand(projType),
		TestCmd:         defaultTestCommand(projType),
	}

	if options.DryRun {
		core.Print(nil, "Would extract workspace/%s to %s", tmplName, options.Path)
		core.Print(nil, "  Template found: %s", tmplName)
		return core.Result{Value: options.Path, OK: true}
	}

	if result := lib.ExtractWorkspace(tmplName, options.Path, data); !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Result{
				Value: core.E("setup.scaffoldTemplate", core.Concat("extract workspace template ", tmplName), err),
				OK:    false,
			}
		}
		return core.Result{
			Value: core.E("setup.scaffoldTemplate", core.Concat("extract workspace template ", tmplName), nil),
			OK:    false,
		}
	}
	return core.Result{Value: options.Path, OK: true}
}

func writeConfig(path, content string, options Options) core.Result {
	if options.DryRun {
		core.Print(nil, "  %s", path)
		return core.Result{Value: path, OK: true}
	}

	if !options.Force && fs.Exists(path) {
		core.Print(nil, "  skip %s (exists, use --force to overwrite)", core.PathBase(path))
		return core.Result{Value: path, OK: true}
	}

	if writeResult := fs.WriteMode(path, content, 0644); !writeResult.OK {
		err, _ := writeResult.Value.(error)
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
