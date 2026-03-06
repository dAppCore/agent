package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

func loadMarketplace() (Marketplace, string, error) {
	root, err := findRepoRoot()
	if err != nil {
		return Marketplace{}, "", err
	}

	path := filepath.Join(root, marketplacePath)
	var marketplace Marketplace
	if err := readJSONFile(path, &marketplace); err != nil {
		return Marketplace{}, "", err
	}

	return marketplace, root, nil
}

func marketplaceListHandler(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	marketplace, _, err := loadMarketplace()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load marketplace: %v", err)), nil
	}

	return mcp.NewToolResultStructuredOnly(marketplace), nil
}
