// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"os/exec"
	"path/filepath"
	"strings"

	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

// ConfigData holds the data passed to config templates.
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

// Target is a build target (os/arch pair).
type Target struct {
	OS   string
	Arch string
}

// Command is a named runnable command.
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

// GenerateBuildConfig renders a build.yaml for the detected project type.
//
//	content, err := setup.GenerateBuildConfig("/repo", setup.TypeGo)
func GenerateBuildConfig(path string, projType ProjectType) (string, error) {
	name := filepath.Base(path)
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
				{Key: "main", Value: "./cmd/" + name},
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

	return renderConfig(name+" build configuration", sections)
}

// GenerateTestConfig renders a test.yaml for the detected project type.
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
	var builder strings.Builder

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
				return "", core.E("setup.renderConfig", "marshal "+section.Key+"."+value.Key, err)
			}

			builder.WriteString("  ")
			builder.WriteString(value.Key)
			builder.WriteString(": ")
			builder.WriteString(strings.TrimSpace(string(scalar)))
			builder.WriteString("\n")
		}

		if idx < len(sections)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

// detectGitRemote extracts owner/repo from git remote origin.
func detectGitRemote(path string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(output))

	// SSH: git@github.com:owner/repo.git or ssh://git@forge.lthn.ai:2223/core/agent.git
	if strings.Contains(url, ":") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			repo := parts[1]
			repo = strings.TrimSuffix(repo, ".git")
			// Handle port in SSH URL (ssh://git@host:port/path)
			if strings.Contains(repo, "/") {
				segments := strings.SplitN(repo, "/", 2)
				if len(segments) == 2 && strings.ContainsAny(segments[0], "0123456789") {
					repo = segments[1]
				}
			}
			return repo
		}
	}

	// HTTPS: https://github.com/owner/repo.git
	for _, host := range []string{"github.com/", "forge.lthn.ai/"} {
		if idx := strings.Index(url, host); idx >= 0 {
			repo := url[idx+len(host):]
			return strings.TrimSuffix(repo, ".git")
		}
	}

	return ""
}
