// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lib"
)

// result := service.Run(setup.Options{Path: ".", Template: "auto", Force: true})
// if !result.OK { core.Print(nil, "%v", result.Value) }
type Options struct {
	Path     string
	DryRun   bool
	Force    bool
	Template string
}

// result := service.Run(setup.Options{Path: ".", Template: "auto"})
// core.Println(result.OK)
func (s *Service) Run(options Options) core.Result {
	if options.Path == "" {
		options.Path = core.Env("DIR_CWD")
	}
	options.Path = absolutePath(options.Path)

	projectType := Detect(options.Path)
	allTypes := DetectAll(options.Path)

	core.Print(nil, "Project: %s", core.PathBase(options.Path))
	core.Print(nil, "Type:    %s", projectType)
	if len(allTypes) > 1 {
		core.Print(nil, "Also:    %v (polyglot)", allTypes)
	}

	var templateName string
	if options.Template != "" {
		templateResult := resolveTemplateName(options.Template, projectType)
		if !templateResult.OK {
			return templateResult
		}
		templateName = templateResult.Value.(string)
		if !templateExists(templateName) {
			return core.Result{
				Value: core.E("setup.Run", core.Concat("template not found: ", templateName), nil),
				OK:    false,
			}
		}
	}

	if result := setupCoreDir(options, projectType); !result.OK {
		return result
	}

	if templateName != "" {
		return s.scaffoldTemplate(options, projectType, templateName)
	}

	return core.Result{Value: options.Path, OK: true}
}

// result := setupCoreDir(Options{Path: ".", Force: true}, TypeGo)
func setupCoreDir(options Options, projectType ProjectType) core.Result {
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

	buildConfig := GenerateBuildConfig(options.Path, projectType)
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

	testConfig := GenerateTestConfig(projectType)
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

// result := s.scaffoldTemplate(Options{Path: ".", Template: "default"}, TypeGo, "default")
func (s *Service) scaffoldTemplate(options Options, projectType ProjectType, templateName string) core.Result {
	core.Print(nil, "Template: %s", templateName)

	data := &lib.WorkspaceData{
		Repo:            core.PathBase(options.Path),
		Branch:          "main",
		Task:            core.Sprintf("Initialise %s project tooling.", projectType),
		Agent:           "setup",
		Language:        string(projectType),
		Prompt:          "This workspace was scaffolded by pkg/setup. Review the repository and continue from the generated context files.",
		Flow:            formatFlow(projectType),
		RepoDescription: s.DetectGitRemote(options.Path),
		BuildCmd:        defaultBuildCommand(projectType),
		TestCmd:         defaultTestCommand(projectType),
	}

	if options.DryRun {
		core.Print(nil, "Would extract workspace/%s to %s", templateName, options.Path)
		core.Print(nil, "  Template found: %s", templateName)
		return core.Result{Value: options.Path, OK: true}
	}

	if result := lib.ExtractWorkspace(templateName, options.Path, data); !result.OK {
		if err, ok := result.Value.(error); ok {
			return core.Result{
				Value: core.E("setup.scaffoldTemplate", core.Concat("extract workspace template ", templateName), err),
				OK:    false,
			}
		}
		return core.Result{
			Value: core.E("setup.scaffoldTemplate", core.Concat("extract workspace template ", templateName), nil),
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

func resolveTemplateName(name string, projectType ProjectType) core.Result {
	if name == "" {
		return core.Result{
			Value: core.E("setup.resolveTemplateName", "template is required", nil),
			OK:    false,
		}
	}

	if name == "auto" {
		switch projectType {
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

func defaultBuildCommand(projectType ProjectType) string {
	switch projectType {
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

func defaultTestCommand(projectType ProjectType) string {
	switch projectType {
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

func formatFlow(projectType ProjectType) string {
	builder := core.NewBuilder()
	builder.WriteString("- Build: `")
	builder.WriteString(defaultBuildCommand(projectType))
	builder.WriteString("`\n")
	builder.WriteString("- Test: `")
	builder.WriteString(defaultTestCommand(projectType))
	builder.WriteString("`")
	return builder.String()
}
