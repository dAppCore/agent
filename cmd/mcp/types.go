package main

type Marketplace struct {
	Schema      string              `json:"$schema,omitempty"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Owner       MarketplaceOwner    `json:"owner"`
	Plugins     []MarketplacePlugin `json:"plugins"`
}

type MarketplaceOwner struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type MarketplacePlugin struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	Category    string `json:"category"`
}

type PluginInfo struct {
	Plugin   MarketplacePlugin  `json:"plugin"`
	Path     string             `json:"path"`
	Manifest map[string]any     `json:"manifest,omitempty"`
	Commands []string           `json:"commands,omitempty"`
	Skills   []string           `json:"skills,omitempty"`
}

type CoreCliResult struct {
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Stdout   string   `json:"stdout"`
	Stderr   string   `json:"stderr"`
	ExitCode int      `json:"exit_code"`
}

type EthicsContext struct {
	Modal  string         `json:"modal"`
	Axioms map[string]any `json:"axioms"`
}
