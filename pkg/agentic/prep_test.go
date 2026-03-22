// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestEnvOr_Good_EnvSet(t *testing.T) {
	t.Setenv("TEST_ENVVAR_CUSTOM", "custom-value")
	assert.Equal(t, "custom-value", envOr("TEST_ENVVAR_CUSTOM", "default"))
}

func TestEnvOr_Good_Fallback(t *testing.T) {
	t.Setenv("TEST_ENVVAR_MISSING", "")
	assert.Equal(t, "default-value", envOr("TEST_ENVVAR_MISSING", "default-value"))
}

func TestEnvOr_Good_UnsetUsesFallback(t *testing.T) {
	t.Setenv("TEST_ENVVAR_TOTALLY_MISSING", "")
	assert.Equal(t, "fallback", envOr("TEST_ENVVAR_TOTALLY_MISSING", "fallback"))
}

func TestDetectLanguage_Good_Go(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "go.mod"), "module test").OK)
	assert.Equal(t, "go", detectLanguage(dir))
}

func TestDetectLanguage_Good_PHP(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "composer.json"), "{}").OK)
	assert.Equal(t, "php", detectLanguage(dir))
}

func TestDetectLanguage_Good_TypeScript(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "package.json"), "{}").OK)
	assert.Equal(t, "ts", detectLanguage(dir))
}

func TestDetectLanguage_Good_Rust(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "Cargo.toml"), "[package]").OK)
	assert.Equal(t, "rust", detectLanguage(dir))
}

func TestDetectLanguage_Good_Python(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "requirements.txt"), "flask").OK)
	assert.Equal(t, "py", detectLanguage(dir))
}

func TestDetectLanguage_Good_Cpp(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "CMakeLists.txt"), "cmake_minimum_required").OK)
	assert.Equal(t, "cpp", detectLanguage(dir))
}

func TestDetectLanguage_Good_Docker(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "Dockerfile"), "FROM alpine").OK)
	assert.Equal(t, "docker", detectLanguage(dir))
}

func TestDetectLanguage_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "go", detectLanguage(dir))
}

func TestDetectBuildCmd_Good(t *testing.T) {
	tests := []struct {
		file     string
		content  string
		expected string
	}{
		{"go.mod", "module test", "go build ./..."},
		{"composer.json", "{}", "composer install"},
		{"package.json", "{}", "npm run build"},
		{"requirements.txt", "flask", "pip install -e ."},
		{"Cargo.toml", "[package]", "cargo build"},
		{"CMakeLists.txt", "cmake", "cmake --build ."},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			dir := t.TempDir()
			require.True(t, fs.Write(filepath.Join(dir, tt.file), tt.content).OK)
			assert.Equal(t, tt.expected, detectBuildCmd(dir))
		})
	}
}

func TestDetectBuildCmd_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "go build ./...", detectBuildCmd(dir))
}

func TestDetectTestCmd_Good(t *testing.T) {
	tests := []struct {
		file     string
		content  string
		expected string
	}{
		{"go.mod", "module test", "go test ./..."},
		{"composer.json", "{}", "composer test"},
		{"package.json", "{}", "npm test"},
		{"requirements.txt", "flask", "pytest"},
		{"Cargo.toml", "[package]", "cargo test"},
		{"CMakeLists.txt", "cmake", "ctest"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			dir := t.TempDir()
			require.True(t, fs.Write(filepath.Join(dir, tt.file), tt.content).OK)
			assert.Equal(t, tt.expected, detectTestCmd(dir))
		})
	}
}

func TestDetectTestCmd_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "go test ./...", detectTestCmd(dir))
}

func TestSanitiseBranchSlug_Good(t *testing.T) {
	assert.Equal(t, "fix-login-bug", sanitiseBranchSlug("Fix login bug!", 40))
	assert.Equal(t, "trim-me", sanitiseBranchSlug("---Trim Me---", 40))
}

func TestSanitiseBranchSlug_Good_Truncates(t *testing.T) {
	assert.Equal(t, "feature", sanitiseBranchSlug("feature--extra", 7))
}

func TestSanitiseFilename_Good(t *testing.T) {
	assert.Equal(t, "Core---Agent-Notes", sanitiseFilename("Core / Agent:Notes"))
}

func TestNewPrep_Good_Defaults(t *testing.T) {
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("FORGE_URL", "")
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("SPECS_PATH", "")
	t.Setenv("CODE_PATH", "")

	s := NewPrep()
	assert.Equal(t, "https://forge.lthn.ai", s.forgeURL)
	assert.Equal(t, "https://api.lthn.sh", s.brainURL)
	assert.NotNil(t, s.client)
}

func TestNewPrep_Good_EnvOverrides(t *testing.T) {
	t.Setenv("FORGE_URL", "https://custom-forge.example.com")
	t.Setenv("FORGE_TOKEN", "test-token")
	t.Setenv("CORE_BRAIN_URL", "https://custom-brain.example.com")
	t.Setenv("CORE_BRAIN_KEY", "brain-key-123")
	t.Setenv("SPECS_PATH", "/custom/specs")
	t.Setenv("CODE_PATH", "/custom/code")

	s := NewPrep()
	assert.Equal(t, "https://custom-forge.example.com", s.forgeURL)
	assert.Equal(t, "test-token", s.forgeToken)
	assert.Equal(t, "https://custom-brain.example.com", s.brainURL)
	assert.Equal(t, "brain-key-123", s.brainKey)
	assert.Equal(t, "/custom/code", s.codePath)
}

func TestNewPrep_Good_GiteaTokenFallback(t *testing.T) {
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "gitea-fallback-token")

	s := NewPrep()
	assert.Equal(t, "gitea-fallback-token", s.forgeToken)
}

func TestPrepSubsystem_Good_Name(t *testing.T) {
	s := &PrepSubsystem{}
	assert.Equal(t, "agentic", s.Name())
}

func TestSetCompletionNotifier_Good(t *testing.T) {
	s := &PrepSubsystem{}
	assert.Nil(t, s.onComplete)

	notifier := &mockNotifier{}
	s.SetCompletionNotifier(notifier)
	assert.NotNil(t, s.onComplete)
}

type mockNotifier struct {
	started   bool
	completed bool
}

func (m *mockNotifier) AgentStarted(agent, repo, workspace string) {
	m.started = true
}

func (m *mockNotifier) AgentCompleted(agent, repo, workspace, status string) {
	m.completed = true
}
