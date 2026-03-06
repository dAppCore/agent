package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
)

func marketplacePluginInfoHandler(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}

	marketplace, root, err := loadMarketplace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load marketplace: %v", err)), nil
	}

	plugin, ok := findMarketplacePlugin(marketplace, name)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("plugin not found: %s", name)), nil
	}

	path := filepath.Join(root, plugin.Source)
	commands, _ := listCommands(path)
	skills, _ := listSkills(path)
	manifest, _ := loadPluginManifest(path)

	info := PluginInfo{
		Plugin:   plugin,
		Path:     path,
		Manifest: manifest,
		Commands: commands,
		Skills:   skills,
	}

	return mcp.NewToolResultStructuredOnly(info), nil
}

func findMarketplacePlugin(marketplace Marketplace, name string) (MarketplacePlugin, bool) {
	for _, plugin := range marketplace.Plugins {
		if plugin.Name == name {
			return plugin, true
		}
	}

	return MarketplacePlugin{}, false
}

func listCommands(path string) ([]string, error) {
	commandsPath := filepath.Join(path, "commands")
	info, err := os.Stat(commandsPath)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	var commands []string
	_ = filepath.WalkDir(commandsPath, func(entryPath string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(commandsPath, entryPath)
		if relErr != nil {
			return nil
		}
		commands = append(commands, filepath.ToSlash(rel))
		return nil
	})

	slices.Sort(commands)
	return commands, nil
}

func listSkills(path string) ([]string, error) {
	skillsPath := filepath.Join(path, "skills")
	info, err := os.Stat(skillsPath)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		return nil, err
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() {
			skills = append(skills, entry.Name())
		}
	}

	slices.Sort(skills)
	return skills, nil
}

func loadPluginManifest(path string) (map[string]any, error) {
	candidates := []string{
		filepath.Join(path, ".claude-plugin", "plugin.json"),
		filepath.Join(path, ".codex-plugin", "plugin.json"),
		filepath.Join(path, "gemini-extension.json"),
	}

	for _, candidate := range candidates {
		payload, err := readJSONMap(candidate)
		if err == nil {
			return payload, nil
		}
	}

	return nil, nil
}
