// SPDX-License-Identifier: EUPL-1.2

// service := core.New(core.WithService(setup.Register)).Service("setup")
package setup

import (
	core "dappco.re/go"
)

// projectType := setup.Detect("/srv/repos/agent")
// if projectType == setup.TypeGo { setup.GenerateBuildConfig("/srv/repos/agent", setup.TypeGo) }
type ProjectType string

const (
	TypeGo      ProjectType = "go"
	TypePHP     ProjectType = "php"
	TypeNode    ProjectType = "node"
	TypeWails   ProjectType = "wails"
	TypeUnknown ProjectType = "unknown"
)

// fs := (&core.Fs{}).NewUnrestricted()
var fs = (&core.Fs{}).NewUnrestricted()

// projectType := setup.Detect("./repo")
func Detect(path string) ProjectType {
	base := absolutePath(path)
	checks := []struct {
		file        string
		projectType ProjectType
	}{
		{"wails.json", TypeWails},
		{"go.mod", TypeGo},
		{"composer.json", TypePHP},
		{"package.json", TypeNode},
	}
	for _, candidate := range checks {
		if fs.IsFile(core.JoinPath(base, candidate.file)) {
			return candidate.projectType
		}
	}
	return TypeUnknown
}

// types := setup.DetectAll("./repo")
func DetectAll(path string) []ProjectType {
	base := absolutePath(path)
	var projectTypes []ProjectType
	checks := []struct {
		file        string
		projectType ProjectType
	}{
		{"go.mod", TypeGo},
		{"composer.json", TypePHP},
		{"package.json", TypeNode},
		{"wails.json", TypeWails},
	}
	for _, candidate := range checks {
		if fs.IsFile(core.JoinPath(base, candidate.file)) {
			projectTypes = append(projectTypes, candidate.projectType)
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
