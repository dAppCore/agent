// SPDX-License-Identifier: EUPL-1.2

// Package setup provides workspace setup and scaffolding using lib templates.
package setup

import (
	"path/filepath"
	"unsafe"

	core "dappco.re/go/core"
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

// fs provides unrestricted filesystem access for setup operations.
var fs = newFs("/")

// newFs creates a core.Fs with the given root directory.
func newFs(root string) *core.Fs {
	type fsRoot struct{ root string }
	f := &core.Fs{}
	(*fsRoot)(unsafe.Pointer(f)).root = root
	return f
}

// Detect identifies the project type from files present at the given path.
//
//	projType := setup.Detect("./repo")
func Detect(path string) ProjectType {
	base := absolutePath(path)
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
		if fs.IsFile(filepath.Join(base, c.file)) {
			return c.projType
		}
	}
	return TypeUnknown
}

// DetectAll returns all project types found at the path (polyglot repos).
//
//	types := setup.DetectAll("./repo")
func DetectAll(path string) []ProjectType {
	base := absolutePath(path)
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
		if fs.IsFile(filepath.Join(base, c.file)) {
			types = append(types, c.projType)
		}
	}
	return types
}

func absolutePath(path string) string {
	if path == "" {
		path = "."
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
