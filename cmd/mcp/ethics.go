package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

const ethicsModalPath = "codex/ethics/MODAL.md"
const ethicsAxiomsPath = "codex/ethics/kernel/axioms.json"

func ethicsCheckHandler(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	root, err := findRepoRoot()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to locate repo root: %v", err)), nil
	}

	modalBytes, err := os.ReadFile(filepath.Join(root, ethicsModalPath))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read modal: %v", err)), nil
	}

	axioms, err := readJSONMap(filepath.Join(root, ethicsAxiomsPath))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read axioms: %v", err)), nil
	}

	payload := EthicsContext{
		Modal:  string(modalBytes),
		Axioms: axioms,
	}

	return mcp.NewToolResultStructuredOnly(payload), nil
}
