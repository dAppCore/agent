// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
)

// Options controls setup behaviour.
//
//	err := setup.Run(setup.Options{Path: ".", Force: true})
type Options struct {
	Path     string // Target directory (default: cwd)
	DryRun   bool   // Preview only, don't write
	Force    bool   // Overwrite existing files
	Template string // Workspace template or compatibility alias (default, review, security, agent, go, php, gui, auto)
}

// Run performs the workspace setup at the given path.
// It detects the project type, generates .core/ configs,
// and optionally scaffolds a workspace from a dir template.
//
//	err := setup.Run(setup.Options{Path: ".", Template: "auto"})
func Run(opts Options) error {
	if opts.Path == "" {
		var err error
		opts.Path, err = os.Getwd()
		if err != nil {
			return core.E("setup.Run", "resolve working directory", err)
		}
	}
	opts.Path = absolutePath(opts.Path)

	projType := Detect(opts.Path)
	allTypes := DetectAll(opts.Path)

	core.Print(nil, "Project: %s", filepath.Base(opts.Path))
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
		return scaffoldTemplate(opts, projType)
	}

	return nil
}

// setupCoreDir creates .core/ with build.yaml and test.yaml.
func setupCoreDir(opts Options, projType ProjectType) error {
	coreDir := filepath.Join(opts.Path, ".core")

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
	if err := writeConfig(filepath.Join(coreDir, "build.yaml"), buildConfig, opts); err != nil {
		return err
	}

	// test.yaml
	testConfig, err := GenerateTestConfig(projType)
	if err != nil {
		return core.E("setup.setupCoreDir", "generate test config", err)
	}
	if err := writeConfig(filepath.Join(coreDir, "test.yaml"), testConfig, opts); err != nil {
		return err
	}

	return nil
}

// scaffoldTemplate extracts a dir template into the target path.
func scaffoldTemplate(opts Options, projType ProjectType) error {
	tmplName, err := resolveTemplateName(opts.Template, projType)
	if err != nil {
		return err
	}

	core.Print(nil, "Template: %s", tmplName)

	data := &lib.WorkspaceData{
		Repo:            filepath.Base(opts.Path),
		Branch:          "main",
		Task:            fmt.Sprintf("Initialise %s project tooling.", projType),
		Agent:           "setup",
		Language:        string(projType),
		Prompt:          "This workspace was scaffolded by pkg/setup. Review the repository and continue from the generated context files.",
		Flow:            formatFlow(projType),
		RepoDescription: detectGitRemote(opts.Path),
		BuildCmd:        defaultBuildCommand(projType),
		TestCmd:         defaultTestCommand(projType),
	}

	if !templateExists(tmplName) {
		return core.E("setup.scaffoldTemplate", "template not found: "+tmplName, nil)
	}

	if opts.DryRun {
		core.Print(nil, "Would extract workspace/%s to %s", tmplName, opts.Path)
		core.Print(nil, "  Template found: %s", tmplName)
		return nil
	}

	if err := lib.ExtractWorkspace(tmplName, opts.Path, data); err != nil {
		return core.E("setup.scaffoldTemplate", "extract workspace template "+tmplName, err)
	}
	return nil
}

func writeConfig(path, content string, opts Options) error {
	if opts.DryRun {
		core.Print(nil, "  %s", path)
		return nil
	}

	if !opts.Force && fs.Exists(path) {
		core.Print(nil, "  skip %s (exists, use --force to overwrite)", filepath.Base(path))
		return nil
	}

	if r := fs.WriteMode(path, content, 0644); !r.OK {
		err, _ := r.Value.(error)
		return core.E("setup.writeConfig", "write "+filepath.Base(path), err)
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
	var builder strings.Builder
	builder.WriteString("- Build: `")
	builder.WriteString(defaultBuildCommand(projType))
	builder.WriteString("`\n")
	builder.WriteString("- Test: `")
	builder.WriteString(defaultTestCommand(projType))
	builder.WriteString("`")
	return builder.String()
}
