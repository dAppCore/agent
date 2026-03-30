// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

// data := setup.ConfigData{Name: "agent", Type: "go", Repository: "core/agent"}
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

// target := setup.Target{OS: "linux", Arch: "amd64"}
type Target struct {
	OS   string
	Arch string
}

// command := setup.Command{Name: "unit", Run: "go test ./..."}
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

// r := setup.GenerateBuildConfig("/srv/repos/agent", setup.TypeGo)
// if r.OK { content := r.Value.(string) }
func GenerateBuildConfig(path string, projectType ProjectType) core.Result {
	name := core.PathBase(path)
	sections := []configSection{
		{
			Key: "project",
			Values: []configValue{
				{Key: "name", Value: name},
				{Key: "type", Value: string(projectType)},
			},
		},
	}

	switch projectType {
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

// r := setup.GenerateTestConfig(setup.TypeGo)
// if r.OK { content := r.Value.(string) }
func GenerateTestConfig(projectType ProjectType) core.Result {
	var sections []configSection

	switch projectType {
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

func renderConfig(headerComment string, sections []configSection) core.Result {
	builder := core.NewBuilder()

	if headerComment != "" {
		builder.WriteString("# ")
		builder.WriteString(headerComment)
		builder.WriteString("\n\n")
	}

	for sectionIndex, section := range sections {
		builder.WriteString(section.Key)
		builder.WriteString(":\n")

		for _, value := range section.Values {
			scalar := marshalConfigValue(value.Value)
			if !scalar.OK {
				err, _ := scalar.Value.(error)
				return core.Result{
					Value: core.E("setup.renderConfig", core.Concat("marshal ", section.Key, ".", value.Key), err),
					OK:    false,
				}
			}

			builder.WriteString("  ")
			builder.WriteString(value.Key)
			builder.WriteString(": ")
			builder.WriteString(scalar.Value.(string))
			builder.WriteString("\n")
		}

		if sectionIndex < len(sections)-1 {
			builder.WriteString("\n")
		}
	}

	return core.Result{Value: builder.String(), OK: true}
}

func marshalConfigValue(value any) (result core.Result) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = core.Result{
				Value: core.E("setup.marshalConfigValue", core.Sprint(recovered), nil),
				OK:    false,
			}
		}
	}()

	data, err := yaml.Marshal(value)
	if err != nil {
		return core.Result{
			Value: core.E("setup.marshalConfigValue", "yaml marshal value", err),
			OK:    false,
		}
	}
	return core.Result{Value: core.Trim(string(data)), OK: true}
}

func parseGitRemote(remote string) string {
	remote = core.Trim(remote)
	if remote == "" {
		return ""
	}

	// HTTPS/HTTP URL — extract path after host
	if core.Contains(remote, "://") {
		schemeParts := core.SplitN(remote, "://", 2)
		if len(schemeParts) == 2 {
			rest := schemeParts[1]
			if pathSegments := core.Split(rest, "/"); len(pathSegments) > 1 {
				// Skip host, take path
				pathStart := len(pathSegments[0]) + 1
				if pathStart < len(rest) {
					return trimRemotePath(rest[pathStart:])
				}
			}
		}
	}

	pathParts := core.SplitN(remote, ":", 2)
	if len(pathParts) == 2 && core.Contains(pathParts[0], "@") {
		return trimRemotePath(pathParts[1])
	}

	if core.Contains(remote, "/") {
		return trimRemotePath(remote)
	}

	return ""
}

func trimRemotePath(remote string) string {
	trimmed := core.Trim(remote)
	for core.HasPrefix(trimmed, "/") {
		trimmed = core.TrimPrefix(trimmed, "/")
	}
	for core.HasSuffix(trimmed, "/") {
		trimmed = core.TrimSuffix(trimmed, "/")
	}
	return core.TrimSuffix(trimmed, ".git")
}
