// SPDX-License-Identifier: EUPL-1.2

// Package prompts provides embedded prompt content for agent dispatch.
// All content is loaded from lib/ at compile time via go:embed.
//
// Structure:
//
//	lib/prompts/   — System prompts (PROMPT.md content, HOW to work)
//	lib/tasks/     — Structured task plans (PLAN.md, WHAT to do)
//	lib/flows/     — Multi-phase workflows (orchestration sequences)
//	lib/personas/  — Domain/role system prompts (WHO you are)
//
// Usage:
//
//	prompt, _ := prompts.Prompt("coding")
//	task, _ := prompts.Task("bug-fix")
//	persona, _ := prompts.Persona("secops/developer")
package prompts

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed lib/prompts/*.md
var promptFS embed.FS

//go:embed lib/tasks/*.yaml
var taskFS embed.FS

//go:embed lib/flows/*.md
var flowFS embed.FS

//go:embed lib/personas
var personaFS embed.FS

// Prompt returns a system prompt by slug (written as PROMPT.md).
// Slugs: "coding", "verify", "conventions", "security", "default".
func Prompt(slug string) (string, error) {
	data, err := promptFS.ReadFile("lib/prompts/" + slug + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Template is an alias for Prompt (backwards compatibility).
func Template(slug string) (string, error) {
	// Try prompts first, then tasks
	if content, err := Prompt(slug); err == nil {
		return content, nil
	}
	return Task(slug)
}

// Task returns a structured task plan by slug (written as PLAN.md).
// Slugs: "bug-fix", "new-feature", "refactor", "code-review", etc.
func Task(slug string) (string, error) {
	for _, ext := range []string{".yaml", ".yml"} {
		data, err := taskFS.ReadFile("lib/tasks/" + slug + ext)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fs.ErrNotExist
}

// Flow returns a multi-phase workflow by slug.
func Flow(slug string) (string, error) {
	data, err := flowFS.ReadFile("lib/flows/" + slug + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Persona returns a domain/role system prompt by path.
// Paths: "secops/developer", "code/backend-architect", "smm/tiktok-strategist".
func Persona(path string) (string, error) {
	data, err := personaFS.ReadFile("lib/personas/" + path + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListPrompts returns all available prompt slugs.
func ListPrompts() []string {
	return listDir(promptFS, "lib/prompts")
}

// ListTasks returns all available task plan slugs.
func ListTasks() []string {
	return listDir(taskFS, "lib/tasks")
}

// ListFlows returns all available flow slugs.
func ListFlows() []string {
	return listDir(flowFS, "lib/flows")
}

// ListTemplates returns all prompt + task slugs (backwards compatibility).
func ListTemplates() []string {
	return append(ListPrompts(), ListTasks()...)
}

// ListPersonas returns all available persona paths.
func ListPersonas() []string {
	var paths []string
	fs.WalkDir(personaFS, "lib/personas", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			rel := strings.TrimPrefix(path, "lib/personas/")
			rel = strings.TrimSuffix(rel, ".md")
			paths = append(paths, rel)
		}
		return nil
	})
	return paths
}

// listDir returns slugs (filename without extension) from an embedded directory.
func listDir(fsys embed.FS, dir string) []string {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		slugs = append(slugs, strings.TrimSuffix(name, ext))
	}
	return slugs
}
