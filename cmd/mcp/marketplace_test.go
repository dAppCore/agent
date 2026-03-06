package main

import (
	"path/filepath"
	"testing"
)

func TestMarketplaceLoad_Good(t *testing.T) {
	marketplace, root, err := loadMarketplace()
	if err != nil {
		t.Fatalf("expected marketplace to load: %v", err)
	}
	if marketplace.Name == "" {
		t.Fatalf("expected marketplace name to be set")
	}
	if len(marketplace.Plugins) == 0 {
		t.Fatalf("expected marketplace plugins")
	}
	if root == "" {
		t.Fatalf("expected repo root")
	}
}

func TestMarketplacePluginInfo_Bad(t *testing.T) {
	marketplace, _, err := loadMarketplace()
	if err != nil {
		t.Fatalf("expected marketplace to load: %v", err)
	}
	if _, ok := findMarketplacePlugin(marketplace, "missing-plugin"); ok {
		t.Fatalf("expected missing plugin")
	}
}

func TestMarketplacePluginInfo_Good(t *testing.T) {
	marketplace, root, err := loadMarketplace()
	if err != nil {
		t.Fatalf("expected marketplace to load: %v", err)
	}

	plugin, ok := findMarketplacePlugin(marketplace, "code")
	if !ok {
		t.Fatalf("expected code plugin")
	}

	commands, err := listCommands(filepath.Join(root, plugin.Source))
	if err != nil {
		t.Fatalf("expected commands to list: %v", err)
	}
	if len(commands) == 0 {
		t.Fatalf("expected commands for code plugin")
	}
}
