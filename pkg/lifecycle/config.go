package lifecycle

import (
	"os"
	"path/filepath"
	"strings"

	"forge.lthn.ai/core/go-io"
	"forge.lthn.ai/core/go-log"
	"gopkg.in/yaml.v3"
)

// Config holds the configuration for connecting to the core-agentic service.
type Config struct {
	// BaseURL is the URL of the core-agentic API server.
	BaseURL string `yaml:"base_url" json:"base_url"`
	// Token is the authentication token for API requests.
	Token string `yaml:"token" json:"token"`
	// DefaultProject is the project to use when none is specified.
	DefaultProject string `yaml:"default_project" json:"default_project"`
	// AgentID is the identifier for this agent (optional, used for claiming tasks).
	AgentID string `yaml:"agent_id" json:"agent_id"`
}

// configFileName is the name of the YAML config file.
const configFileName = "agentic.yaml"

// envFileName is the name of the environment file.
const envFileName = ".env"

// DefaultBaseURL is the default API endpoint if none is configured.
// Set AGENTIC_BASE_URL to override:
//   - Lab:  https://api.lthn.sh
//   - Prod: https://api.lthn.ai
const DefaultBaseURL = "https://api.lthn.sh"

// LoadConfig loads the agentic configuration from the specified directory.
// It first checks for a .env file, then falls back to ~/.core/agentic.yaml.
// If dir is empty, it checks the current directory first.
//
// Environment variables take precedence:
//   - AGENTIC_BASE_URL: API base URL
//   - AGENTIC_TOKEN: Authentication token
//   - AGENTIC_PROJECT: Default project
//   - AGENTIC_AGENT_ID: Agent identifier
func LoadConfig(dir string) (*Config, error) {
	cfg := &Config{
		BaseURL: DefaultBaseURL,
	}

	// Try loading from .env file in the specified directory
	if dir != "" {
		envPath := filepath.Join(dir, envFileName)
		if err := loadEnvFile(envPath, cfg); err == nil {
			// Successfully loaded from .env
			applyEnvOverrides(cfg)
			if cfg.Token != "" {
				return cfg, nil
			}
		}
	}

	// Try loading from current directory .env
	if dir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			envPath := filepath.Join(cwd, envFileName)
			if err := loadEnvFile(envPath, cfg); err == nil {
				applyEnvOverrides(cfg)
				if cfg.Token != "" {
					return cfg, nil
				}
			}
		}
	}

	// Try loading from ~/.core/agentic.yaml
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, log.E("agentic.LoadConfig", "failed to get home directory", err)
	}

	configPath := filepath.Join(homeDir, ".core", configFileName)
	if err := loadYAMLConfig(configPath, cfg); err != nil && !os.IsNotExist(err) {
		return nil, log.E("agentic.LoadConfig", "failed to load config", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Validate configuration
	if cfg.Token == "" {
		return nil, log.E("agentic.LoadConfig", "no authentication token configured", nil)
	}

	return cfg, nil
}

// loadEnvFile reads a .env file and extracts agentic configuration.
func loadEnvFile(path string, cfg *Config) error {
	content, err := io.Local.Read(path)
	if err != nil {
		return err
	}

	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		switch key {
		case "AGENTIC_BASE_URL":
			cfg.BaseURL = value
		case "AGENTIC_TOKEN":
			cfg.Token = value
		case "AGENTIC_PROJECT":
			cfg.DefaultProject = value
		case "AGENTIC_AGENT_ID":
			cfg.AgentID = value
		}
	}

	return nil
}

// loadYAMLConfig reads configuration from a YAML file.
func loadYAMLConfig(path string, cfg *Config) error {
	content, err := io.Local.Read(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal([]byte(content), cfg)
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("AGENTIC_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("AGENTIC_TOKEN"); v != "" {
		cfg.Token = v
	}
	if v := os.Getenv("AGENTIC_PROJECT"); v != "" {
		cfg.DefaultProject = v
	}
	if v := os.Getenv("AGENTIC_AGENT_ID"); v != "" {
		cfg.AgentID = v
	}
}

// SaveConfig saves the configuration to ~/.core/agentic.yaml.
func SaveConfig(cfg *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return log.E("agentic.SaveConfig", "failed to get home directory", err)
	}

	configDir := filepath.Join(homeDir, ".core")
	if err := io.Local.EnsureDir(configDir); err != nil {
		return log.E("agentic.SaveConfig", "failed to create config directory", err)
	}

	configPath := filepath.Join(configDir, configFileName)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return log.E("agentic.SaveConfig", "failed to marshal config", err)
	}

	if err := io.Local.Write(configPath, string(data)); err != nil {
		return log.E("agentic.SaveConfig", "failed to write config file", err)
	}

	return nil
}

// ConfigPath returns the path to the config file in the user's home directory.
func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", log.E("agentic.ConfigPath", "failed to get home directory", err)
	}
	return filepath.Join(homeDir, ".core", configFileName), nil
}

// AllowanceConfig controls allowance store backend selection.
type AllowanceConfig struct {
	// StoreBackend is the storage backend: "memory", "sqlite", or "redis". Default: "memory".
	StoreBackend string `yaml:"store_backend" json:"store_backend"`
	// StorePath is the file path for the SQLite database.
	// Default: ~/.config/agentic/allowance.db (only used when StoreBackend is "sqlite").
	StorePath string `yaml:"store_path" json:"store_path"`
	// RedisAddr is the host:port for the Redis server (only used when StoreBackend is "redis").
	RedisAddr string `yaml:"redis_addr" json:"redis_addr"`
}

// DefaultAllowanceStorePath returns the default SQLite path for allowance data.
func DefaultAllowanceStorePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", log.E("agentic.DefaultAllowanceStorePath", "failed to get home directory", err)
	}
	return filepath.Join(homeDir, ".config", "agentic", "allowance.db"), nil
}

// NewAllowanceStoreFromConfig creates an AllowanceStore based on the given config.
// It returns a MemoryStore for "memory" (or empty) backend and a SQLiteStore for "sqlite".
func NewAllowanceStoreFromConfig(cfg AllowanceConfig) (AllowanceStore, error) {
	switch cfg.StoreBackend {
	case "", "memory":
		return NewMemoryStore(), nil
	case "sqlite":
		dbPath := cfg.StorePath
		if dbPath == "" {
			var err error
			dbPath, err = DefaultAllowanceStorePath()
			if err != nil {
				return nil, err
			}
		}
		return NewSQLiteStore(dbPath)
	case "redis":
		return NewRedisStore(cfg.RedisAddr)
	default:
		return nil, &APIError{
			Code:    400,
			Message: "unsupported store backend: " + cfg.StoreBackend,
		}
	}
}

// RegistryConfig controls agent registry backend selection.
type RegistryConfig struct {
	// RegistryBackend is the storage backend: "memory", "sqlite", or "redis". Default: "memory".
	RegistryBackend string `yaml:"registry_backend" json:"registry_backend"`
	// RegistryPath is the file path for the SQLite database.
	// Default: ~/.config/agentic/registry.db (only used when RegistryBackend is "sqlite").
	RegistryPath string `yaml:"registry_path" json:"registry_path"`
	// RegistryRedisAddr is the host:port for the Redis server (only used when RegistryBackend is "redis").
	RegistryRedisAddr string `yaml:"registry_redis_addr" json:"registry_redis_addr"`
}

// DefaultRegistryPath returns the default SQLite path for registry data.
func DefaultRegistryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", log.E("agentic.DefaultRegistryPath", "failed to get home directory", err)
	}
	return filepath.Join(homeDir, ".config", "agentic", "registry.db"), nil
}

// NewAgentRegistryFromConfig creates an AgentRegistry based on the given config.
// It returns a MemoryRegistry for "memory" (or empty) backend, a SQLiteRegistry
// for "sqlite", and a RedisRegistry for "redis".
func NewAgentRegistryFromConfig(cfg RegistryConfig) (AgentRegistry, error) {
	switch cfg.RegistryBackend {
	case "", "memory":
		return NewMemoryRegistry(), nil
	case "sqlite":
		dbPath := cfg.RegistryPath
		if dbPath == "" {
			var err error
			dbPath, err = DefaultRegistryPath()
			if err != nil {
				return nil, err
			}
		}
		return NewSQLiteRegistry(dbPath)
	case "redis":
		return NewRedisRegistry(cfg.RegistryRedisAddr)
	default:
		return nil, &APIError{
			Code:    400,
			Message: "unsupported registry backend: " + cfg.RegistryBackend,
		}
	}
}
