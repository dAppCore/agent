// SPDX-License-Identifier: EUPL-1.2

// Package prompts provides embedded prompt templates and personas for agent dispatch.
// Templates and personas are loaded from lib/ at compile time via go:embed.
//
// Usage:
//
//	template, _ := prompts.Template("bug-fix")
//	persona, _ := prompts.Persona("engineering/engineering-security-engineer")
//	all := prompts.ListTemplates()
package prompts

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed lib/templates/*.yaml lib/templates/*.md
var templateFS embed.FS

//go:embed lib/personas
var personaFS embed.FS

// Template returns the content of a prompt template by slug.
// Slug examples: "bug-fix", "code-review", "security".
func Template(slug string) (string, error) {
	// Try .yaml first, then .yml, then .md
	for _, ext := range []string{".yaml", ".yml", ".md"} {
		data, err := templateFS.ReadFile("lib/templates/" + slug + ext)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fs.ErrNotExist
}

// Persona returns the content of a persona by path.
// Path examples: "engineering/engineering-security-engineer",
// "testing/testing-api-tester", "specialized/blockchain-security-auditor".
func Persona(path string) (string, error) {
	data, err := personaFS.ReadFile("lib/personas/" + path + ".md")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListTemplates returns all available template slugs.
func ListTemplates() []string {
	entries, err := templateFS.ReadDir("lib/templates")
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
		slug := strings.TrimSuffix(name, ext)
		slugs = append(slugs, slug)
	}
	return slugs
}

// ListPersonas returns all available persona paths.
func ListPersonas() []string {
	var paths []string
	fs.WalkDir(personaFS, "lib/personas", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".md") {
			// Strip prefix and extension: lib/personas/engineering/foo.md → engineering/foo
			rel := strings.TrimPrefix(path, "lib/personas/")
			rel = strings.TrimSuffix(rel, ".md")
			paths = append(paths, rel)
		}
		return nil
	})
	return paths
}

// TemplateFS returns the raw embedded filesystem for templates.
func TemplateFS() embed.FS {
	return templateFS
}

// PersonaFS returns the raw embedded filesystem for personas.
func PersonaFS() embed.FS {
	return personaFS
}
