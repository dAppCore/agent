// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"dappco.re/go/agent/pkg/lib"
)

// Options controls setup behaviour.
type Options struct {
	Path     string // Target directory (default: cwd)
	DryRun   bool   // Preview only, don't write
	Force    bool   // Overwrite existing files
	Template string // Dir template to use (agent, php, go, gui)
}

// Run performs the workspace setup at the given path.
// It detects the project type, generates .core/ configs,
// and optionally scaffolds a workspace from a dir template.
func Run(opts Options) error {
	if opts.Path == "" {
		var err error
		opts.Path, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("setup: %w", err)
		}
	}

	projType := Detect(opts.Path)
	allTypes := DetectAll(opts.Path)

	fmt.Printf("Project: %s\n", filepath.Base(opts.Path))
	fmt.Printf("Type:    %s\n", projType)
	if len(allTypes) > 1 {
		fmt.Printf("Also:    %v (polyglot)\n", allTypes)
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
		fmt.Printf("\nWould create %s/\n", coreDir)
	} else {
		if err := os.MkdirAll(coreDir, 0755); err != nil {
			return fmt.Errorf("setup: create .core: %w", err)
		}
	}

	// build.yaml
	buildConfig, err := GenerateBuildConfig(opts.Path, projType)
	if err != nil {
		return fmt.Errorf("setup: build config: %w", err)
	}
	if err := writeConfig(filepath.Join(coreDir, "build.yaml"), buildConfig, opts); err != nil {
		return err
	}

	// test.yaml
	testConfig, err := GenerateTestConfig(projType)
	if err != nil {
		return fmt.Errorf("setup: test config: %w", err)
	}
	if err := writeConfig(filepath.Join(coreDir, "test.yaml"), testConfig, opts); err != nil {
		return err
	}

	return nil
}

// scaffoldTemplate extracts a dir template into the target path.
func scaffoldTemplate(opts Options, projType ProjectType) error {
	tmplName := opts.Template
	if tmplName == "auto" {
		switch projType {
		case TypeGo, TypeWails:
			tmplName = "go"
		case TypePHP:
			tmplName = "php"
		case TypeNode:
			tmplName = "gui"
		default:
			tmplName = "agent"
		}
	}

	fmt.Printf("Template: %s\n", tmplName)

	data := map[string]any{
		"Name":          filepath.Base(opts.Path),
		"Module":        detectGitRemote(),
		"Namespace":     "App",
		"ViewNamespace": filepath.Base(opts.Path),
		"RouteName":     filepath.Base(opts.Path),
		"GoVersion":     "1.26",
		"HasAdmin":      true,
		"HasApi":        true,
		"HasConsole":    true,
	}

	if opts.DryRun {
		fmt.Printf("Would extract template/%s to %s\n", tmplName, opts.Path)
		files := lib.ListDirTemplates()
		for _, f := range files {
			if f == tmplName {
				fmt.Printf("  Template found: %s\n", f)
			}
		}
		return nil
	}

	return lib.ExtractDir(tmplName, opts.Path, data)
}

func writeConfig(path, content string, opts Options) error {
	if opts.DryRun {
		fmt.Printf("  %s\n", path)
		return nil
	}

	if !opts.Force {
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  skip %s (exists, use --force to overwrite)\n", filepath.Base(path))
			return nil
		}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("setup: write %s: %w", filepath.Base(path), err)
	}
	fmt.Printf("  created %s\n", path)
	return nil
}
