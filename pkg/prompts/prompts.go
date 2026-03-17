// SPDX-License-Identifier: EUPL-1.2

// Package prompts provides embedded prompt content for agent dispatch.
// All content is loaded from lib/ at compile time via go:embed.
//
// Structure:
//
//	lib/prompt/    — System prompts (PROMPT.md content, HOW to work)
//	lib/task/      — Structured task plans (PLAN.md, WHAT to do)
//	lib/task/code/ — Code-specific tasks (review, refactor, dead-code, test-gaps)
//	lib/flow/      — Build/release workflows per language/tool
//	lib/persona/   — Domain/role system prompts (WHO you are)
//
// Usage:
//
//	prompt, _ := prompts.Prompt("coding")
//	task, _ := prompts.Task("bug-fix")
//	task, _ := prompts.Task("code/review")
//	persona, _ := prompts.Persona("secops/developer")
//	flow, _ := prompts.Flow("go")
package prompts

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed lib/prompt/*.md
var promptFS embed.FS

//go:embed lib/task
var taskFS embed.FS

//go:embed lib/flow/*.md
var flowFS embed.FS

//go:embed lib/persona
var personaFS embed.FS

// Prompt returns a system prompt by slug (written as PROMPT.md).
// Slugs: "coding", "verify", "conventions", "security", "default".
func Prompt(slug string) (string, error) {
	data, err := promptFS.ReadFile("lib/prompt/" + slug + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Template is an alias for Prompt (backwards compatibility).
func Template(slug string) (string, error) {
	if content, err := Prompt(slug); err == nil {
		return content, nil
	}
	return Task(slug)
}

// Task returns a structured task plan by slug (written as PLAN.md).
// Slugs: "bug-fix", "new-feature", "code/review", "code/refactor", etc.
func Task(slug string) (string, error) {
	for _, ext := range []string{".yaml", ".yml", ".md"} {
		data, err := taskFS.ReadFile("lib/task/" + slug + ext)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fs.ErrNotExist
}

// Flow returns a build/release workflow by slug.
// Slugs: "go", "php", "ts", "docker", "release", etc.
func Flow(slug string) (string, error) {
	data, err := flowFS.ReadFile("lib/flow/" + slug + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Persona returns a domain/role system prompt by path.
// Paths: "secops/developer", "code/backend-architect", "smm/tiktok-strategist".
func Persona(path string) (string, error) {
	data, err := personaFS.ReadFile("lib/persona/" + path + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListPrompts returns all available prompt slugs.
func ListPrompts() []string {
	return listDir(promptFS, "lib/prompt")
}

// ListTasks returns all available task plan slugs (including nested like code/review).
func ListTasks() []string {
	var slugs []string
	fs.WalkDir(taskFS, "lib/task", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "lib/task/")
		ext := filepath.Ext(rel)
		slugs = append(slugs, strings.TrimSuffix(rel, ext))
		return nil
	})
	return slugs
}

// ListFlows returns all available flow slugs.
func ListFlows() []string {
	return listDir(flowFS, "lib/flow")
}

// ListTemplates returns all prompt + task slugs (backwards compatibility).
func ListTemplates() []string {
	return append(ListPrompts(), ListTasks()...)
}

// ListPersonas returns all available persona paths.
func ListPersonas() []string {
	var paths []string
	fs.WalkDir(personaFS, "lib/persona", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			rel := strings.TrimPrefix(path, "lib/persona/")
			rel = strings.TrimSuffix(rel, ".md")
			paths = append(paths, rel)
		}
		return nil
	})
	return paths
}

// listDir returns slugs from an embedded directory (non-recursive).
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
