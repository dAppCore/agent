// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"path/filepath"
	"strings"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrep_EnvOr_Good_EnvSet(t *testing.T) {
	t.Setenv("TEST_ENVVAR_CUSTOM", "custom-value")
	assert.Equal(t, "custom-value", envOr("TEST_ENVVAR_CUSTOM", "default"))
}

func TestPrep_EnvOr_Good_Fallback(t *testing.T) {
	t.Setenv("TEST_ENVVAR_MISSING", "")
	assert.Equal(t, "default-value", envOr("TEST_ENVVAR_MISSING", "default-value"))
}

func TestPrep_EnvOr_Good_UnsetUsesFallback(t *testing.T) {
	t.Setenv("TEST_ENVVAR_TOTALLY_MISSING", "")
	assert.Equal(t, "fallback", envOr("TEST_ENVVAR_TOTALLY_MISSING", "fallback"))
}

func TestPrep_DetectLanguage_Good_Go(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "go.mod"), "module test").OK)
	assert.Equal(t, "go", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_PHP(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "composer.json"), "{}").OK)
	assert.Equal(t, "php", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_TypeScript(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "package.json"), "{}").OK)
	assert.Equal(t, "ts", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Rust(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "Cargo.toml"), "[package]").OK)
	assert.Equal(t, "rust", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Python(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "requirements.txt"), "flask").OK)
	assert.Equal(t, "py", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Cpp(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "CMakeLists.txt"), "cmake_minimum_required").OK)
	assert.Equal(t, "cpp", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Docker(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(filepath.Join(dir, "Dockerfile"), "FROM alpine").OK)
	assert.Equal(t, "docker", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "go", detectLanguage(dir))
}

func TestPrep_DetectBuildCmd_Good(t *testing.T) {
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

func TestPrep_DetectBuildCmd_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "go build ./...", detectBuildCmd(dir))
}

func TestPrep_DetectTestCmd_Good(t *testing.T) {
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

func TestPrep_DetectTestCmd_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "go test ./...", detectTestCmd(dir))
}

func TestSanitise_SanitiseBranchSlug_Good(t *testing.T) {
	assert.Equal(t, "fix-login-bug", sanitiseBranchSlug("Fix login bug!", 40))
	assert.Equal(t, "trim-me", sanitiseBranchSlug("---Trim Me---", 40))
}

func TestSanitise_SanitiseBranchSlug_Good_Truncates(t *testing.T) {
	assert.Equal(t, "feature", sanitiseBranchSlug("feature--extra", 7))
}

func TestSanitise_SanitiseFilename_Good(t *testing.T) {
	assert.Equal(t, "Core---Agent-Notes", sanitiseFilename("Core / Agent:Notes"))
}

func TestPrep_NewPrep_Good_Defaults(t *testing.T) {
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

func TestPrep_NewPrep_Good_EnvOverrides(t *testing.T) {
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

func TestPrep_NewPrep_Good_GiteaTokenFallback(t *testing.T) {
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "gitea-fallback-token")

	s := NewPrep()
	assert.Equal(t, "gitea-fallback-token", s.forgeToken)
}

func TestPrepSubsystem_Good_Name(t *testing.T) {
	s := &PrepSubsystem{}
	assert.Equal(t, "agentic", s.Name())
}

func TestPrep_SetCore_Good(t *testing.T) {
	s := &PrepSubsystem{}
	assert.Nil(t, s.core)

	c := core.New(core.WithOption("name", "test"))
	s.SetCore(c)
	assert.NotNil(t, s.core)
}

// --- sanitiseBranchSlug Bad/Ugly ---

func TestSanitise_SanitiseBranchSlug_Bad_EmptyString(t *testing.T) {
	assert.Equal(t, "", sanitiseBranchSlug("", 40))
}

func TestSanitise_SanitiseBranchSlug_Bad_OnlySpecialChars(t *testing.T) {
	assert.Equal(t, "", sanitiseBranchSlug("!@#$%^&*()", 40))
}

func TestSanitise_SanitiseBranchSlug_Bad_OnlyDashes(t *testing.T) {
	assert.Equal(t, "", sanitiseBranchSlug("------", 40))
}

func TestSanitise_SanitiseBranchSlug_Ugly_VeryLongString(t *testing.T) {
	long := strings.Repeat("abcdefghij", 100)
	result := sanitiseBranchSlug(long, 50)
	assert.LessOrEqual(t, len(result), 50)
}

func TestSanitise_SanitiseBranchSlug_Ugly_Unicode(t *testing.T) {
	// Unicode chars should be replaced with dashes, then edges trimmed
	result := sanitiseBranchSlug("\u00e9\u00e0\u00fc\u00f1\u00f0", 40)
	assert.NotContains(t, result, "\u00e9")
	// All replaced with dashes, then trimmed = empty
	assert.Equal(t, "", result)
}

func TestSanitise_SanitiseBranchSlug_Ugly_ZeroMax(t *testing.T) {
	// max=0 means no limit
	result := sanitiseBranchSlug("hello-world", 0)
	assert.Equal(t, "hello-world", result)
}

// --- sanitisePlanSlug Bad/Ugly ---

func TestSanitise_SanitisePlanSlug_Bad_EmptyString(t *testing.T) {
	assert.Equal(t, "", sanitisePlanSlug(""))
}

func TestSanitise_SanitisePlanSlug_Bad_OnlySpecialChars(t *testing.T) {
	assert.Equal(t, "", sanitisePlanSlug("!@#$%^&*()"))
}

func TestSanitise_SanitisePlanSlug_Bad_OnlySpaces(t *testing.T) {
	// Spaces become dashes, then collapsed, then trimmed
	assert.Equal(t, "", sanitisePlanSlug("     "))
}

func TestSanitise_SanitisePlanSlug_Ugly_VeryLongString(t *testing.T) {
	long := strings.Repeat("abcdefghij ", 20)
	result := sanitisePlanSlug(long)
	assert.LessOrEqual(t, len(result), 30)
}

func TestSanitise_SanitisePlanSlug_Ugly_Unicode(t *testing.T) {
	result := sanitisePlanSlug("\u00e9\u00e0\u00fc\u00f1\u00f0")
	assert.Equal(t, "", result, "unicode chars should be stripped, leaving empty string")
}

func TestSanitise_SanitisePlanSlug_Ugly_AllDashInput(t *testing.T) {
	assert.Equal(t, "", sanitisePlanSlug("---"))
}

// --- sanitiseFilename Bad/Ugly ---

func TestSanitise_SanitiseFilename_Bad_EmptyString(t *testing.T) {
	assert.Equal(t, "", sanitiseFilename(""))
}

func TestSanitise_SanitiseFilename_Bad_OnlySpecialChars(t *testing.T) {
	result := sanitiseFilename("!@#$%^&*()")
	// All replaced with dashes
	assert.Equal(t, "----------", result)
}

func TestSanitise_SanitiseFilename_Ugly_VeryLongString(t *testing.T) {
	long := strings.Repeat("a", 1000)
	result := sanitiseFilename(long)
	assert.Equal(t, 1000, len(result))
}

func TestSanitise_SanitiseFilename_Ugly_Unicode(t *testing.T) {
	result := sanitiseFilename("\u00e9\u00e0\u00fc\u00f1\u00f0")
	// All replaced with dashes
	for _, r := range result {
		assert.Equal(t, '-', r)
	}
}

func TestSanitise_SanitiseFilename_Ugly_PreservesDotsUnderscores(t *testing.T) {
	assert.Equal(t, "my_file.test.txt", sanitiseFilename("my_file.test.txt"))
}

// --- collapseRepeatedRune Bad/Ugly ---

func TestSanitise_CollapseRepeatedRune_Bad_EmptyString(t *testing.T) {
	assert.Equal(t, "", collapseRepeatedRune("", '-'))
}

func TestSanitise_CollapseRepeatedRune_Bad_AllTarget(t *testing.T) {
	assert.Equal(t, "-", collapseRepeatedRune("-----", '-'))
}

func TestSanitise_CollapseRepeatedRune_Ugly_Unicode(t *testing.T) {
	assert.Equal(t, "h\u00e9llo", collapseRepeatedRune("h\u00e9\u00e9\u00e9llo", '\u00e9'))
}

func TestSanitise_CollapseRepeatedRune_Ugly_VeryLong(t *testing.T) {
	long := strings.Repeat("--a", 500)
	result := collapseRepeatedRune(long, '-')
	assert.NotContains(t, result, "--")
}

// --- trimRuneEdges Bad/Ugly ---

func TestSanitise_TrimRuneEdges_Bad_EmptyString(t *testing.T) {
	assert.Equal(t, "", trimRuneEdges("", '-'))
}

func TestSanitise_TrimRuneEdges_Bad_AllTarget(t *testing.T) {
	assert.Equal(t, "", trimRuneEdges("-----", '-'))
}

func TestSanitise_TrimRuneEdges_Ugly_Unicode(t *testing.T) {
	assert.Equal(t, "hello", trimRuneEdges("\u00e9hello\u00e9\u00e9", '\u00e9'))
}

func TestSanitise_TrimRuneEdges_Ugly_NoMatch(t *testing.T) {
	assert.Equal(t, "hello", trimRuneEdges("hello", '-'))
}
