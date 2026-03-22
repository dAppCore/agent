// SPDX-License-Identifier: EUPL-1.2

// Package lib provides embedded content for agent dispatch.
// Prompts, tasks, flows, personas, and workspace templates.
//
// Structure:
//
//	prompt/      — System prompts (HOW to work)
//	task/        — Structured task plans (WHAT to do)
//	task/code/   — Code-specific tasks (review, refactor, etc.)
//	flow/        — Build/release workflows per language/tool
//	persona/     — Domain/role system prompts (WHO you are)
//	workspace/   — Agent workspace templates (WHERE to work)
//
// Usage:
//
//	prompt, _ := lib.Prompt("coding")
//	task, _ := lib.Task("code/review")
//	persona, _ := lib.Persona("secops/developer")
//	flow, _ := lib.Flow("go")
//	lib.ExtractWorkspace("default", "/tmp/ws", data)
package lib

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed prompt/*.md
var promptFS embed.FS

//go:embed all:task
var taskFS embed.FS

//go:embed flow/*.md
var flowFS embed.FS

//go:embed persona
var personaFS embed.FS

//go:embed all:workspace
var workspaceFS embed.FS

// --- Prompts ---

// Template tries Prompt then Task (backwards compat).
func Template(slug string) (string, error) {
	if content, err := Prompt(slug); err == nil {
		return content, nil
	}
	return Task(slug)
}

func Prompt(slug string) (string, error) {
	data, err := promptFS.ReadFile("prompt/" + slug + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Task(slug string) (string, error) {
	for _, ext := range []string{".md", ".yaml", ".yml"} {
		data, err := taskFS.ReadFile("task/" + slug + ext)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fs.ErrNotExist
}

func TaskBundle(slug string) (string, map[string]string, error) {
	main, err := Task(slug)
	if err != nil {
		return "", nil, err
	}
	bundleDir := "task/" + slug
	entries, err := fs.ReadDir(taskFS, bundleDir)
	if err != nil {
		return main, nil, nil
	}
	bundle := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := taskFS.ReadFile(bundleDir + "/" + e.Name())
		if err == nil {
			bundle[e.Name()] = string(data)
		}
	}
	return main, bundle, nil
}

func Flow(slug string) (string, error) {
	data, err := flowFS.ReadFile("flow/" + slug + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Persona(path string) (string, error) {
	data, err := personaFS.ReadFile("persona/" + path + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// --- Workspace Templates ---

// WorkspaceData is the data passed to workspace templates.
type WorkspaceData struct {
	Repo            string
	Branch          string
	Task            string
	Agent           string
	Language        string
	Prompt          string
	Persona         string
	Flow            string
	Context         string
	Recent          string
	Dependencies    string
	Conventions     string
	RepoDescription string
	BuildCmd        string
	TestCmd         string
}

// ExtractWorkspace creates an agent workspace from a template.
// Template names: "default", "security", "review".
func ExtractWorkspace(tmplName, targetDir string, data *WorkspaceData) error {
	wsDir := "workspace/" + tmplName
	entries, err := fs.ReadDir(workspaceFS, wsDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		content, err := fs.ReadFile(workspaceFS, wsDir+"/"+name)
		if err != nil {
			return err
		}

		// Process .tmpl files through text/template
		outputName := name
		if strings.HasSuffix(name, ".tmpl") {
			outputName = strings.TrimSuffix(name, ".tmpl")
			tmpl, err := template.New(name).Parse(string(content))
			if err != nil {
				return err
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				return err
			}
			content = buf.Bytes()
		}

		if err := os.WriteFile(filepath.Join(targetDir, outputName), content, 0644); err != nil {
			return err
		}
	}

	return nil
}

// --- List Functions ---

func ListPrompts() []string    { return listDir(promptFS, "prompt") }
func ListFlows() []string      { return listDir(flowFS, "flow") }
func ListWorkspaces() []string { return listDir(workspaceFS, "workspace") }

func ListTasks() []string {
	var slugs []string
	fs.WalkDir(taskFS, "task", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "task/")
		ext := filepath.Ext(rel)
		slugs = append(slugs, strings.TrimSuffix(rel, ext))
		return nil
	})
	return slugs
}

func ListPersonas() []string {
	var paths []string
	fs.WalkDir(personaFS, "persona", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			rel := strings.TrimPrefix(path, "persona/")
			rel = strings.TrimSuffix(rel, ".md")
			paths = append(paths, rel)
		}
		return nil
	})
	return paths
}

func listDir(fsys embed.FS, dir string) []string {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			slugs = append(slugs, name)
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		slugs = append(slugs, strings.TrimSuffix(name, ext))
	}
	return slugs
}
