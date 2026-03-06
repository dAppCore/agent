package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Good_FromEnvFile(t *testing.T) {
	// Create temp directory with .env file
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	envContent := `
AGENTIC_BASE_URL=https://test.api.com
AGENTIC_TOKEN=test-token-123
AGENTIC_PROJECT=my-project
AGENTIC_AGENT_ID=agent-001
`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(tmpDir)

	require.NoError(t, err)
	assert.Equal(t, "https://test.api.com", cfg.BaseURL)
	assert.Equal(t, "test-token-123", cfg.Token)
	assert.Equal(t, "my-project", cfg.DefaultProject)
	assert.Equal(t, "agent-001", cfg.AgentID)
}

func TestLoadConfig_Good_FromEnvVars(t *testing.T) {
	// Create temp directory with .env file (partial config)
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	envContent := `
AGENTIC_TOKEN=env-file-token
`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	// Set environment variables that should override
	_ = os.Setenv("AGENTIC_BASE_URL", "https://env-override.com")
	_ = os.Setenv("AGENTIC_TOKEN", "env-override-token")
	defer func() {
		_ = os.Unsetenv("AGENTIC_BASE_URL")
		_ = os.Unsetenv("AGENTIC_TOKEN")
	}()

	cfg, err := LoadConfig(tmpDir)

	require.NoError(t, err)
	assert.Equal(t, "https://env-override.com", cfg.BaseURL)
	assert.Equal(t, "env-override-token", cfg.Token)
}

func TestLoadConfig_Bad_NoToken(t *testing.T) {
	// Create temp directory without config
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create empty .env
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(""), 0644)
	require.NoError(t, err)

	// Ensure no env vars are set
	_ = os.Unsetenv("AGENTIC_TOKEN")
	_ = os.Unsetenv("AGENTIC_BASE_URL")

	_, err = LoadConfig(tmpDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no authentication token")
}

func TestLoadConfig_Good_EnvFileWithQuotes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test with quoted values
	envContent := `
AGENTIC_TOKEN="quoted-token"
AGENTIC_BASE_URL='single-quoted-url'
`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(tmpDir)

	require.NoError(t, err)
	assert.Equal(t, "quoted-token", cfg.Token)
	assert.Equal(t, "single-quoted-url", cfg.BaseURL)
}

func TestLoadConfig_Good_EnvFileWithComments(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	envContent := `
# This is a comment
AGENTIC_TOKEN=token-with-comments

# Another comment
AGENTIC_PROJECT=commented-project
`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(tmpDir)

	require.NoError(t, err)
	assert.Equal(t, "token-with-comments", cfg.Token)
	assert.Equal(t, "commented-project", cfg.DefaultProject)
}

func TestSaveConfig_Good(t *testing.T) {
	// Create temp home directory
	tmpHome, err := os.MkdirTemp("", "agentic-home")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpHome) }()

	// Override HOME for the test
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	cfg := &Config{
		BaseURL:        "https://saved.api.com",
		Token:          "saved-token",
		DefaultProject: "saved-project",
		AgentID:        "saved-agent",
	}

	err = SaveConfig(cfg)
	require.NoError(t, err)

	// Verify file was created
	configPath := filepath.Join(tmpHome, ".core", "agentic.yaml")
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Read back the config
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "saved.api.com")
	assert.Contains(t, string(data), "saved-token")
}

func TestConfigPath_Good(t *testing.T) {
	path, err := ConfigPath()

	require.NoError(t, err)
	assert.Contains(t, path, ".core")
	assert.Contains(t, path, "agentic.yaml")
}

func TestLoadConfig_Good_DefaultBaseURL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Only provide token, should use default base URL
	envContent := `
AGENTIC_TOKEN=test-token
`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	// Clear any env overrides
	_ = os.Unsetenv("AGENTIC_BASE_URL")

	cfg, err := LoadConfig(tmpDir)

	require.NoError(t, err)
	assert.Equal(t, DefaultBaseURL, cfg.BaseURL)
}

func TestLoadConfig_Good_FromYAMLFallback(t *testing.T) {
	// Set up a temp home with ~/.core/agentic.yaml
	tmpHome, err := os.MkdirTemp("", "agentic-home")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpHome) }()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Clear all env vars so we fall through to YAML.
	_ = os.Unsetenv("AGENTIC_TOKEN")
	_ = os.Unsetenv("AGENTIC_BASE_URL")
	_ = os.Unsetenv("AGENTIC_PROJECT")
	_ = os.Unsetenv("AGENTIC_AGENT_ID")

	// Create ~/.core/agentic.yaml
	configDir := filepath.Join(tmpHome, ".core")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	yamlContent := `base_url: https://yaml.api.com
token: yaml-token
default_project: yaml-project
agent_id: yaml-agent
`
	err = os.WriteFile(filepath.Join(configDir, "agentic.yaml"), []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Load from a dir with no .env to force YAML fallback.
	tmpDir, err := os.MkdirTemp("", "agentic-noenv")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "https://yaml.api.com", cfg.BaseURL)
	assert.Equal(t, "yaml-token", cfg.Token)
	assert.Equal(t, "yaml-project", cfg.DefaultProject)
	assert.Equal(t, "yaml-agent", cfg.AgentID)
}

func TestLoadConfig_Good_EnvOverridesYAML(t *testing.T) {
	// Set up a temp home with ~/.core/agentic.yaml
	tmpHome, err := os.MkdirTemp("", "agentic-home")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpHome) }()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create ~/.core/agentic.yaml
	configDir := filepath.Join(tmpHome, ".core")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	yamlContent := `base_url: https://yaml.api.com
token: yaml-token
`
	err = os.WriteFile(filepath.Join(configDir, "agentic.yaml"), []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Set env overrides for project and agent.
	_ = os.Setenv("AGENTIC_PROJECT", "env-project")
	_ = os.Setenv("AGENTIC_AGENT_ID", "env-agent")
	defer func() {
		_ = os.Unsetenv("AGENTIC_TOKEN")
		_ = os.Unsetenv("AGENTIC_BASE_URL")
		_ = os.Unsetenv("AGENTIC_PROJECT")
		_ = os.Unsetenv("AGENTIC_AGENT_ID")
	}()

	tmpDir, err := os.MkdirTemp("", "agentic-noenv")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	cfg, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "env-project", cfg.DefaultProject, "env var should override YAML")
	assert.Equal(t, "env-agent", cfg.AgentID, "env var should override YAML")
}

func TestLoadConfig_Good_EnvFileWithTokenNoOverride(t *testing.T) {
	// Test that .env with a token returns immediately without
	// falling through to YAML.
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	envContent := `AGENTIC_TOKEN=env-file-only`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	_ = os.Unsetenv("AGENTIC_TOKEN")
	_ = os.Unsetenv("AGENTIC_BASE_URL")
	_ = os.Unsetenv("AGENTIC_PROJECT")
	_ = os.Unsetenv("AGENTIC_AGENT_ID")

	cfg, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "env-file-only", cfg.Token)
	assert.Equal(t, DefaultBaseURL, cfg.BaseURL)
}

func TestLoadConfig_Good_EnvFileWithMalformedLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Lines without = sign should be skipped.
	envContent := `
AGENTIC_TOKEN=valid-token
MALFORMED_LINE_NO_EQUALS
ANOTHER_BAD_LINE
AGENTIC_PROJECT=valid-project
`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "valid-token", cfg.Token)
	assert.Equal(t, "valid-project", cfg.DefaultProject)
}

func TestApplyEnvOverrides_Good_AllVars(t *testing.T) {
	_ = os.Setenv("AGENTIC_BASE_URL", "https://override-url.com")
	_ = os.Setenv("AGENTIC_TOKEN", "override-token")
	_ = os.Setenv("AGENTIC_PROJECT", "override-project")
	_ = os.Setenv("AGENTIC_AGENT_ID", "override-agent")
	defer func() {
		_ = os.Unsetenv("AGENTIC_BASE_URL")
		_ = os.Unsetenv("AGENTIC_TOKEN")
		_ = os.Unsetenv("AGENTIC_PROJECT")
		_ = os.Unsetenv("AGENTIC_AGENT_ID")
	}()

	cfg := &Config{}
	applyEnvOverrides(cfg)

	assert.Equal(t, "https://override-url.com", cfg.BaseURL)
	assert.Equal(t, "override-token", cfg.Token)
	assert.Equal(t, "override-project", cfg.DefaultProject)
	assert.Equal(t, "override-agent", cfg.AgentID)
}

func TestApplyEnvOverrides_Good_NoVarsSet(t *testing.T) {
	_ = os.Unsetenv("AGENTIC_BASE_URL")
	_ = os.Unsetenv("AGENTIC_TOKEN")
	_ = os.Unsetenv("AGENTIC_PROJECT")
	_ = os.Unsetenv("AGENTIC_AGENT_ID")

	cfg := &Config{
		BaseURL: "original-url",
		Token:   "original-token",
	}
	applyEnvOverrides(cfg)

	assert.Equal(t, "original-url", cfg.BaseURL, "should not change without env var")
	assert.Equal(t, "original-token", cfg.Token, "should not change without env var")
}

func TestSaveConfig_Good_RoundTrip(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "agentic-home")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpHome) }()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Clear env vars so LoadConfig falls through to YAML.
	_ = os.Unsetenv("AGENTIC_TOKEN")
	_ = os.Unsetenv("AGENTIC_BASE_URL")
	_ = os.Unsetenv("AGENTIC_PROJECT")
	_ = os.Unsetenv("AGENTIC_AGENT_ID")

	original := &Config{
		BaseURL:        "https://roundtrip.api.com",
		Token:          "roundtrip-token",
		DefaultProject: "roundtrip-project",
		AgentID:        "roundtrip-agent",
	}

	err = SaveConfig(original)
	require.NoError(t, err)

	// Load it back by pointing to a dir with no .env.
	tmpDir, err := os.MkdirTemp("", "agentic-noenv")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	loaded, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, original.BaseURL, loaded.BaseURL)
	assert.Equal(t, original.Token, loaded.Token)
	assert.Equal(t, original.DefaultProject, loaded.DefaultProject)
	assert.Equal(t, original.AgentID, loaded.AgentID)
}

func TestLoadConfig_Good_EmptyDirUsesCurrentDir(t *testing.T) {
	// Create a temp directory with .env and chdir into it.
	tmpDir, err := os.MkdirTemp("", "agentic-cwd")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	envContent := `AGENTIC_TOKEN=cwd-token
AGENTIC_BASE_URL=https://cwd.api.com
`
	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0644)
	require.NoError(t, err)

	// Clear env vars.
	_ = os.Unsetenv("AGENTIC_TOKEN")
	_ = os.Unsetenv("AGENTIC_BASE_URL")
	_ = os.Unsetenv("AGENTIC_PROJECT")
	_ = os.Unsetenv("AGENTIC_AGENT_ID")

	// Save and restore cwd.
	originalCwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalCwd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	cfg, err := LoadConfig("")
	require.NoError(t, err)
	assert.Equal(t, "cwd-token", cfg.Token)
	assert.Equal(t, "https://cwd.api.com", cfg.BaseURL)
}

func TestLoadConfig_Good_EnvFileNoToken_FallsToYAML(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "agentic-home")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpHome) }()

	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	_ = os.Unsetenv("AGENTIC_TOKEN")
	_ = os.Unsetenv("AGENTIC_BASE_URL")
	_ = os.Unsetenv("AGENTIC_PROJECT")
	_ = os.Unsetenv("AGENTIC_AGENT_ID")

	// Create .env without a token.
	tmpDir, err := os.MkdirTemp("", "agentic-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	err = os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("AGENTIC_PROJECT=env-proj\n"), 0644)
	require.NoError(t, err)

	// Create YAML with token.
	configDir := filepath.Join(tmpHome, ".core")
	err = os.MkdirAll(configDir, 0755)
	require.NoError(t, err)

	yamlContent := `token: yaml-fallback-token
`
	err = os.WriteFile(filepath.Join(configDir, "agentic.yaml"), []byte(yamlContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "yaml-fallback-token", cfg.Token)
}
