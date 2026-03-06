package main

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const serverName = "host-uk-marketplace"
const serverVersion = "0.1.0"

func newServer() *server.MCPServer {
	srv := server.NewMCPServer(
		serverName,
		serverVersion,
	)

	srv.AddTool(marketplaceListTool(), marketplaceListHandler)
	srv.AddTool(marketplacePluginInfoTool(), marketplacePluginInfoHandler)
	srv.AddTool(coreCliTool(), coreCliHandler)
	srv.AddTool(ethicsCheckTool(), ethicsCheckHandler)

	return srv
}

func marketplaceListTool() mcp.Tool {
	return mcp.NewTool(
		"marketplace_list",
		mcp.WithDescription("List available marketplace plugins"),
	)
}

func marketplacePluginInfoTool() mcp.Tool {
	return mcp.NewTool(
		"marketplace_plugin_info",
		mcp.WithDescription("Return plugin metadata, commands, and skills"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Marketplace plugin name")),
	)
}

func coreCliTool() mcp.Tool {
	rawSchema, err := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type": "string",
				"description": "Core CLI command group (dev, go, php, build)",
			},
			"args": map[string]any{
				"type": "array",
				"items": map[string]any{"type": "string"},
				"description": "Arguments for the command",
			},
		},
		"required": []string{"command"},
	})

	options := []mcp.ToolOption{
		mcp.WithDescription("Run approved core CLI commands"),
	}
	if err == nil {
		options = append(options, mcp.WithRawInputSchema(rawSchema))
	}

	return mcp.NewTool(
		"core_cli",
		options...,
	)
}

func ethicsCheckTool() mcp.Tool {
	return mcp.NewTool(
		"ethics_check",
		mcp.WithDescription("Return the Axioms of Life ethics modal and kernel"),
	)
}
