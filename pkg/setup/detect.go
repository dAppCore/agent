// SPDX-License-Identifier: EUPL-1.2

// Package setup provides workspace setup and scaffolding using lib templates.
package setup

import (
	"os"
	"path/filepath"
)

// ProjectType identifies what kind of project lives at a path.
type ProjectType string

const (
	TypeGo      ProjectType = "go"
	TypePHP     ProjectType = "php"
	TypeNode    ProjectType = "node"
	TypeWails   ProjectType = "wails"
	TypeUnknown ProjectType = "unknown"
)

// Detect identifies the project type from files present at the given path.
func Detect(path string) ProjectType {
	checks := []struct {
		file     string
		projType ProjectType
	}{
		{"wails.json", TypeWails},
		{"go.mod", TypeGo},
		{"composer.json", TypePHP},
		{"package.json", TypeNode},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(path, c.file)); err == nil {
			return c.projType
		}
	}
	return TypeUnknown
}

// DetectAll returns all project types found at the path (polyglot repos).
func DetectAll(path string) []ProjectType {
	var types []ProjectType
	all := []struct {
		file     string
		projType ProjectType
	}{
		{"go.mod", TypeGo},
		{"composer.json", TypePHP},
		{"package.json", TypeNode},
		{"wails.json", TypeWails},
	}
	for _, c := range all {
		if _, err := os.Stat(filepath.Join(path, c.file)); err == nil {
			types = append(types, c.projType)
		}
	}
	return types
}
