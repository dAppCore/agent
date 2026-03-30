// SPDX-License-Identifier: EUPL-1.2

// Package setup provisions `.core/` files and workspace scaffolds for a repo.
//
//	service := core.ServiceFor[*setup.Service](core.New(core.WithService(setup.Register)), "setup")
package setup

import (
	core "dappco.re/go/core"
)

// ProjectType records what setup detected in a repository path.
//
//	projType := setup.Detect("/srv/repos/agent")
//	if projType == setup.TypeGo { /* generate Go defaults */ }
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

// Detect inspects a repository path and returns the primary project type.
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
	for _, candidate := range checks {
		if fs.IsFile(core.JoinPath(base, candidate.file)) {
			return candidate.projType
		}
	}
	return TypeUnknown
}

// DetectAll returns every detected project type for a polyglot repository.
//
//	types := setup.DetectAll("./repo")
func DetectAll(path string) []ProjectType {
	base := absolutePath(path)
	var projectTypes []ProjectType
	checks := []struct {
		file     string
		projType ProjectType
	}{
		{"go.mod", TypeGo},
		{"composer.json", TypePHP},
		{"package.json", TypeNode},
		{"wails.json", TypeWails},
	}
	for _, candidate := range checks {
		if fs.IsFile(core.JoinPath(base, candidate.file)) {
			projectTypes = append(projectTypes, candidate.projType)
		}
	}
	return projectTypes
}

func absolutePath(path string) string {
	if path == "" {
		return core.Env("DIR_CWD")
	}
	return core.Path(path)
}
