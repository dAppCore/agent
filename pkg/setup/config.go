// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"os/exec"
	"path/filepath"
	"strings"

	"dappco.re/go/agent/pkg/lib"
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

// GenerateBuildConfig renders a build.yaml for the detected project type.
func GenerateBuildConfig(path string, projType ProjectType) (string, error) {
	name := filepath.Base(path)
	data := map[string]any{
		"Comment": name + " build configuration",
		"Sections": []map[string]any{
			{
				"Key": "project",
				"Values": []map[string]any{
					{"Key": "name", "Value": name},
					{"Key": "type", "Value": string(projType)},
				},
			},
		},
	}

	switch projType {
	case TypeGo, TypeWails:
		data["Sections"] = append(data["Sections"].([]map[string]any),
			map[string]any{
				"Key": "build",
				"Values": []map[string]any{
					{"Key": "main", "Value": "./cmd/" + name},
					{"Key": "binary", "Value": name},
					{"Key": "cgo", "Value": "false"},
				},
			},
		)
	case TypePHP:
		data["Sections"] = append(data["Sections"].([]map[string]any),
			map[string]any{
				"Key": "build",
				"Values": []map[string]any{
					{"Key": "dockerfile", "Value": "Dockerfile"},
					{"Key": "image", "Value": name},
				},
			},
		)
	case TypeNode:
		data["Sections"] = append(data["Sections"].([]map[string]any),
			map[string]any{
				"Key": "build",
				"Values": []map[string]any{
					{"Key": "script", "Value": "npm run build"},
					{"Key": "output", "Value": "dist"},
				},
			},
		)
	}

	return lib.RenderFile("yaml/config", data)
}

// GenerateTestConfig renders a test.yaml for the detected project type.
func GenerateTestConfig(projType ProjectType) (string, error) {
	data := map[string]any{
		"Comment": "Test configuration",
	}

	switch projType {
	case TypeGo, TypeWails:
		data["Sections"] = []map[string]any{
			{
				"Key": "commands",
				"Values": []map[string]any{
					{"Key": "unit", "Value": "go test ./..."},
					{"Key": "coverage", "Value": "go test -coverprofile=coverage.out ./..."},
					{"Key": "race", "Value": "go test -race ./..."},
				},
			},
		}
	case TypePHP:
		data["Sections"] = []map[string]any{
			{
				"Key": "commands",
				"Values": []map[string]any{
					{"Key": "unit", "Value": "vendor/bin/pest --parallel"},
					{"Key": "lint", "Value": "vendor/bin/pint --test"},
				},
			},
		}
	case TypeNode:
		data["Sections"] = []map[string]any{
			{
				"Key": "commands",
				"Values": []map[string]any{
					{"Key": "unit", "Value": "npm test"},
					{"Key": "lint", "Value": "npm run lint"},
				},
			},
		}
	}

	return lib.RenderFile("yaml/config", data)
}

// detectGitRemote extracts owner/repo from git remote origin.
func detectGitRemote() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
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
