// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

// ConfigData supplies values when setup renders workspace templates.
//
//	data := setup.ConfigData{Name: "agent", Type: "go", Repository: "core/agent"}
type ConfigData struct {
	Name        string
	Description string
	Type        string
	Module      string
	Repository  string
	GoVersion   string
	Targets     []Target
	Commands    []Command
	Env         map[string]string
}

// Target describes one build target.
//
//	target := setup.Target{OS: "linux", Arch: "amd64"}
type Target struct {
	OS   string
	Arch string
}

// Command defines one named command in generated config.
//
//	command := setup.Command{Name: "unit", Run: "go test ./..."}
type Command struct {
	Name string
	Run  string
}

type configSection struct {
	Key    string
	Values []configValue
}

type configValue struct {
	Key   string
	Value any
}

// GenerateBuildConfig renders `build.yaml` content for a detected repo type.
//
//	content, err := setup.GenerateBuildConfig("/srv/repos/agent", setup.TypeGo)
func GenerateBuildConfig(path string, projType ProjectType) (string, error) {
	name := core.PathBase(path)
	sections := []configSection{
		{
			Key: "project",
			Values: []configValue{
				{Key: "name", Value: name},
				{Key: "type", Value: string(projType)},
			},
		},
	}

	switch projType {
	case TypeGo, TypeWails:
		sections = append(sections, configSection{
			Key: "build",
			Values: []configValue{
				{Key: "main", Value: core.Concat("./cmd/", name)},
				{Key: "binary", Value: name},
				{Key: "cgo", Value: false},
			},
		})
	case TypePHP:
		sections = append(sections, configSection{
			Key: "build",
			Values: []configValue{
				{Key: "dockerfile", Value: "Dockerfile"},
				{Key: "image", Value: name},
			},
		})
	case TypeNode:
		sections = append(sections, configSection{
			Key: "build",
			Values: []configValue{
				{Key: "script", Value: "npm run build"},
				{Key: "output", Value: "dist"},
			},
		})
	}

	return renderConfig(core.Concat(name, " build configuration"), sections)
}

// GenerateTestConfig renders `test.yaml` content for a detected repo type.
//
//	content, err := setup.GenerateTestConfig(setup.TypeGo)
func GenerateTestConfig(projType ProjectType) (string, error) {
	var sections []configSection

	switch projType {
	case TypeGo, TypeWails:
		sections = []configSection{
			{
				Key: "commands",
				Values: []configValue{
					{Key: "unit", Value: "go test ./..."},
					{Key: "coverage", Value: "go test -coverprofile=coverage.out ./..."},
					{Key: "race", Value: "go test -race ./..."},
				},
			},
		}
	case TypePHP:
		sections = []configSection{
			{
				Key: "commands",
				Values: []configValue{
					{Key: "unit", Value: "vendor/bin/pest --parallel"},
					{Key: "lint", Value: "vendor/bin/pint --test"},
				},
			},
		}
	case TypeNode:
		sections = []configSection{
			{
				Key: "commands",
				Values: []configValue{
					{Key: "unit", Value: "npm test"},
					{Key: "lint", Value: "npm run lint"},
				},
			},
		}
	}

	return renderConfig("Test configuration", sections)
}

func renderConfig(comment string, sections []configSection) (string, error) {
	builder := core.NewBuilder()

	if comment != "" {
		builder.WriteString("# ")
		builder.WriteString(comment)
		builder.WriteString("\n\n")
	}

	for idx, section := range sections {
		builder.WriteString(section.Key)
		builder.WriteString(":\n")

		for _, value := range section.Values {
			scalar, err := yaml.Marshal(value.Value)
			if err != nil {
				return "", core.E("setup.renderConfig", core.Concat("marshal ", section.Key, ".", value.Key), err)
			}

			builder.WriteString("  ")
			builder.WriteString(value.Key)
			builder.WriteString(": ")
			builder.WriteString(core.Trim(string(scalar)))
			builder.WriteString("\n")
		}

		if idx < len(sections)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

func parseGitRemote(remote string) string {
	if remote == "" {
		return ""
	}

	// HTTPS/HTTP URL — extract path after host
	if core.Contains(remote, "://") {
		parts := core.SplitN(remote, "://", 2)
		if len(parts) == 2 {
			rest := parts[1]
			if idx := core.Split(rest, "/"); len(idx) > 1 {
				// Skip host, take path
				pathStart := len(idx[0]) + 1
				if pathStart < len(rest) {
					return trimRemotePath(rest[pathStart:])
				}
			}
		}
	}

	parts := core.SplitN(remote, ":", 2)
	if len(parts) == 2 && core.Contains(parts[0], "@") {
		return trimRemotePath(parts[1])
	}

	if core.Contains(remote, "/") {
		return trimRemotePath(remote)
	}

	return ""
}

func trimRemotePath(remote string) string {
	trimmed := core.TrimPrefix(remote, "/")
	return core.TrimSuffix(trimmed, ".git")
}
