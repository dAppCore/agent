// SPDX-License-Identifier: EUPL-1.2

// Package setup provides workspace setup and scaffolding using lib templates.
package setup

import (
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
var fs = (&core.Fs{}).NewUnrestricted()

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
		if fs.IsFile(core.JoinPath(base, c.file)) {
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
		if fs.IsFile(core.JoinPath(base, c.file)) {
			types = append(types, c.projType)
		}
	}
	return types
}

func absolutePath(path string) string {
	if path == "" {
		return core.Env("DIR_CWD")
	}
	return core.Path(path)
}
