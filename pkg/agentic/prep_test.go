// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
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
	require.True(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)
	assert.Equal(t, "go", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_PHP(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "composer.json"), "{}").OK)
	assert.Equal(t, "php", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_TypeScript(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "package.json"), "{}").OK)
	assert.Equal(t, "ts", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Rust(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "Cargo.toml"), "[package]").OK)
	assert.Equal(t, "rust", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Python(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "requirements.txt"), "flask").OK)
	assert.Equal(t, "py", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Cpp(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "CMakeLists.txt"), "cmake_minimum_required").OK)
	assert.Equal(t, "cpp", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Docker(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "Dockerfile"), "FROM alpine").OK)
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
			require.True(t, fs.Write(core.JoinPath(dir, tt.file), tt.content).OK)
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
			require.True(t, fs.Write(core.JoinPath(dir, tt.file), tt.content).OK)
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
	assert.NotEmpty(t, s.codePath)
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

func TestPrep_Subsystem_Good_Name(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.Equal(t, "agentic", s.Name())
}

func TestPrep_SetCore_Good(t *testing.T) {
	s := &PrepSubsystem{}
	assert.Nil(t, s.ServiceRuntime)

	c := core.New(core.WithOption("name", "test"))
	s.SetCore(c)
	assert.NotNil(t, s.ServiceRuntime)
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

// --- PrepSubsystem Name Bad/Ugly ---

func TestPrep_Name_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	name := s.Name()
	assert.NotEmpty(t, name, "Name should never return empty")
	assert.Equal(t, "agentic", name)
}

func TestPrep_Name_Ugly(t *testing.T) {
	// Zero-value struct — Name() should still work
	var s PrepSubsystem
	assert.NotPanics(t, func() {
		name := s.Name()
		assert.Equal(t, "agentic", name)
	})
}

// --- NewPrep Bad/Ugly ---

func TestPrep_NewPrep_Bad(t *testing.T) {
	// Call without any env — verify doesn't panic, returns valid struct
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("FORGE_URL", "")
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("SPECS_PATH", "")
	t.Setenv("CODE_PATH", "")

	assert.NotPanics(t, func() {
		s := NewPrep()
		assert.NotNil(t, s)
	})
}

func TestPrep_NewPrep_Ugly(t *testing.T) {
	// Verify returned struct has non-nil backoff/failCount maps
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "")

	s := NewPrep()
	assert.NotNil(t, s.backoff, "backoff map must not be nil")
	assert.NotNil(t, s.failCount, "failCount map must not be nil")
	assert.NotNil(t, s.forge, "Forge client must not be nil")
}

// --- SetCore Bad/Ugly ---

func TestPrep_SetCore_Bad(t *testing.T) {
	// SetCore with nil — should not panic
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.NotPanics(t, func() {
		s.SetCore(nil)
	})
}

func TestPrep_SetCore_Ugly(t *testing.T) {
	// SetCore twice — second overwrites first
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	c1 := core.New(core.WithOption("name", "first"))
	c2 := core.New(core.WithOption("name", "second"))

	s.SetCore(c1)
	assert.NotNil(t, s.ServiceRuntime)

	s.SetCore(c2)
	assert.Equal(t, c2, s.Core(), "second SetCore should overwrite first")
}

// --- OnStartup Good/Bad/Ugly ---

func TestPrep_OnStartup_Good_CreatesPokeCh(t *testing.T) {
	// StartRunner is now a no-op — pokeCh is no longer initialised by OnStartup.
	// Verify OnStartup succeeds and pokeCh remains nil.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	assert.Nil(t, s.pokeCh, "pokeCh should be nil before OnStartup")

	r := s.OnStartup(context.Background())
	assert.True(t, r.OK)

	assert.Nil(t, s.pokeCh, "pokeCh should remain nil — queue drain is owned by pkg/runner")
}

func TestPrep_OnStartup_Good_FrozenByDefault(t *testing.T) {
	// Frozen state is now owned by pkg/runner.Service, not agentic.
	// Verify OnStartup succeeds without asserting frozen state.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	assert.True(t, s.OnStartup(context.Background()).OK)
}

func TestPrep_OnStartup_Good_NoError(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.SetCore(c)

	assert.True(t, s.OnStartup(context.Background()).OK)
}

func TestPrep_OnStartup_Bad(t *testing.T) {
	// OnStartup without SetCore (nil ServiceRuntime) — panics because
	// registerCommands calls s.Core().Command().
	s := &PrepSubsystem{
		ServiceRuntime: nil,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.Panics(t, func() {
		_ = s.OnStartup(context.Background())
	}, "OnStartup without core should panic on registerCommands")
}

func TestPrep_OnStartup_Ugly(t *testing.T) {
	// OnStartup called twice with valid core — second call should not panic
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	c := core.New(core.WithOption("name", "test"))
	s.SetCore(c)

	assert.NotPanics(t, func() {
		_ = s.OnStartup(context.Background())
		_ = s.OnStartup(context.Background())
	})
}

// --- OnShutdown Good/Bad ---

func TestPrep_OnShutdown_Good_FreezesQueue(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: false}
	r := s.OnShutdown(context.Background())
	assert.True(t, r.OK)
	assert.True(t, s.frozen, "OnShutdown must set frozen=true")
}

func TestPrep_OnShutdown_Good_AlreadyFrozen(t *testing.T) {
	// Calling OnShutdown twice must be idempotent
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: true}
	r := s.OnShutdown(context.Background())
	assert.True(t, r.OK)
	assert.True(t, s.frozen)
}

func TestPrep_OnShutdown_Good_NoError(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	assert.True(t, s.OnShutdown(context.Background()).OK)
}

func TestPrep_OnShutdown_Ugly_NilCore(t *testing.T) {
	// OnShutdown must not panic even if s.core is nil
	s := &PrepSubsystem{ServiceRuntime: nil, frozen: false}
	assert.NotPanics(t, func() {
		_ = s.OnShutdown(context.Background())
	})
	assert.True(t, s.frozen)
}

func TestPrep_OnShutdown_Bad(t *testing.T) {
	// OnShutdown without Core
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	assert.NotPanics(t, func() {
		r := s.OnShutdown(context.Background())
		assert.True(t, r.OK)
	})
	assert.True(t, s.frozen)
}

// --- Shutdown Bad/Ugly ---

func TestPrep_Shutdown_Bad(t *testing.T) {
	// Shutdown always returns nil
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	err := s.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.Nil(t, err)
}

func TestPrep_Shutdown_Ugly(t *testing.T) {
	// Shutdown on zero-value struct
	var s PrepSubsystem
	assert.NotPanics(t, func() {
		err := s.Shutdown(context.Background())
		assert.NoError(t, err)
	})
}

// --- EnvOr Bad/Ugly ---

func TestPrep_EnvOr_Bad(t *testing.T) {
	// Both env empty and fallback empty
	t.Setenv("TEST_ENVVAR_EMPTY_ALL", "")
	assert.Equal(t, "", envOr("TEST_ENVVAR_EMPTY_ALL", ""))
}

func TestPrep_EnvOr_Ugly(t *testing.T) {
	// Env set to whitespace — whitespace is non-empty, so returned as-is
	t.Setenv("TEST_ENVVAR_WHITESPACE", "   ")
	assert.Equal(t, "   ", envOr("TEST_ENVVAR_WHITESPACE", "fallback"))
}

// --- DetectLanguage Bad/Ugly ---

func TestPrep_DetectLanguage_Bad(t *testing.T) {
	// Empty dir — defaults to go
	dir := t.TempDir()
	assert.Equal(t, "go", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Ugly(t *testing.T) {
	// Dir with multiple project files (go.mod + package.json) — go wins (first match)
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)
	require.True(t, fs.Write(core.JoinPath(dir, "package.json"), "{}").OK)
	assert.Equal(t, "go", detectLanguage(dir), "go.mod checked first, so go wins")
}

// --- DetectBuildCmd Bad/Ugly ---

func TestPrep_DetectBuildCmd_Bad(t *testing.T) {
	// Unknown/non-existent path — defaults to go build
	assert.Equal(t, "go build ./...", detectBuildCmd("/nonexistent/path/that/does/not/exist"))
}

func TestPrep_DetectBuildCmd_Ugly(t *testing.T) {
	// Path that doesn't exist at all — defaults to go build
	assert.NotPanics(t, func() {
		result := detectBuildCmd("")
		assert.Equal(t, "go build ./...", result)
	})
}

// --- TestPrepWorkspace (public API wrapper) ---

func TestPrep_TestPrepWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Valid input but repo won't exist — still exercises the public wrapper delegation
	_, _, err := s.TestPrepWorkspace(context.Background(), PrepInput{
		Repo:  "go-io",
		Issue: 1,
	})
	// Error expected (no local clone) but we verified it delegates to prepWorkspace
	assert.Error(t, err)
}

func TestPrep_TestPrepWorkspace_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Missing repo — should return error
	_, _, err := s.TestPrepWorkspace(context.Background(), PrepInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestPrep_TestPrepWorkspace_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Bare ".." is caught as invalid repo name by PathBase check
	_, _, err := s.TestPrepWorkspace(context.Background(), PrepInput{
		Repo:  "..",
		Issue: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo name")
}

// --- TestBuildPrompt (public API wrapper) ---

func TestPrep_TestBuildPrompt_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.TestBuildPrompt(context.Background(), PrepInput{
		Task: "Review code",
		Org:  "core",
		Repo: "go-io",
	}, "dev", dir)

	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "TASK: Review code")
	assert.Contains(t, prompt, "REPO: core/go-io on branch dev")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_TestBuildPrompt_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Empty inputs — should still return a prompt string without panicking
	prompt, memories, consumers := s.TestBuildPrompt(context.Background(), PrepInput{}, "", "")
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "TASK:")
	assert.Contains(t, prompt, "CONSTRAINTS:")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_TestBuildPrompt_Ugly(t *testing.T) {
	dir := t.TempDir()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Unicode in all fields — should not panic
	prompt, _, _ := s.TestBuildPrompt(context.Background(), PrepInput{
		Task: "\u00e9nchantr\u00efx \u2603 \U0001f600",
		Org:  "c\u00f6re",
		Repo: "g\u00f6-i\u00f6",
	}, "\u00e9-branch", dir)

	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "\u00e9nchantr\u00efx")
}

// --- collapseRepeatedRune / sanitisePlanSlug / trimRuneEdges Good ---

func TestPrep_CollapseRepeatedRune_Good(t *testing.T) {
	assert.Equal(t, "hello-world", collapseRepeatedRune("hello---world", '-'))
}

func TestPrep_SanitisePlanSlug_Good(t *testing.T) {
	assert.Equal(t, "my-cool-plan", sanitisePlanSlug("My Cool Plan"))
}

func TestPrep_TrimRuneEdges_Good(t *testing.T) {
	assert.Equal(t, "hello", trimRuneEdges("--hello--", '-'))
}

// --- DetectTestCmd Bad/Ugly ---

func TestPrep_DetectTestCmd_Bad(t *testing.T) {
	// Unknown path — defaults to go test
	assert.Equal(t, "go test ./...", detectTestCmd("/nonexistent/path/that/does/not/exist"))
}

func TestPrep_DetectTestCmd_Ugly(t *testing.T) {
	// Path that doesn't exist — defaults to go test
	assert.NotPanics(t, func() {
		result := detectTestCmd("")
		assert.Equal(t, "go test ./...", result)
	})
}

// --- getGitLog ---

func TestPrep_GetGitLog_Good(t *testing.T) {
	dir := t.TempDir()
	gitEnv := []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com"}
	run := func(args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		require.True(t, r.OK, "cmd %v failed: %s", args, r.Value)
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Test")
	run("git", "config", "user.email", "test@test.com")
	require.True(t, fs.Write(core.JoinPath(dir, "README.md"), "# Test").OK)
	run("git", "add", "README.md")
	run("git", "commit", "-m", "initial commit")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	log := s.getGitLog(dir)
	assert.NotEmpty(t, log)
	assert.Contains(t, log, "initial commit")
}

func TestPrep_GetGitLog_Bad(t *testing.T) {
	// Non-git dir returns empty
	dir := t.TempDir()
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	log := s.getGitLog(dir)
	assert.Empty(t, log)
}

func TestPrep_GetGitLog_Ugly(t *testing.T) {
	// Git repo with no commits — git log should fail, returns empty
	dir := t.TempDir()
	testCore.Process().RunIn(context.Background(), dir, "git", "init", "-b", "main")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	log := s.getGitLog(dir)
	assert.Empty(t, log)
}

// --- prepWorkspace Good ---

func TestPrep_PrepWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	// Mock Forge API for issue body
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 1,
			"title":  "Fix tests",
			"body":   "Tests are broken",
		})))
	}))
	t.Cleanup(srv.Close)

	// Create a source repo to clone from
	srcRepo := core.JoinPath(root, "src", "core", "test-repo")
	gitEnv := []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com"}
	run := func(dir string, args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		require.True(t, r.OK, "cmd %v failed: %s", args, r.Value)
	}
	require.True(t, fs.EnsureDir(srcRepo).OK)
	run(srcRepo, "git", "init", "-b", "main")
	run(srcRepo, "git", "config", "user.name", "Test")
	run(srcRepo, "git", "config", "user.email", "test@test.com")
	require.True(t, fs.Write(core.JoinPath(srcRepo, "README.md"), "# Test").OK)
	run(srcRepo, "git", "add", "README.md")
	run(srcRepo, "git", "commit", "-m", "initial commit")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		forge:          forge.NewForge(srv.URL, "test-token"),
		codePath:       core.JoinPath(root, "src"),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, out, err := s.TestPrepWorkspace(context.Background(), PrepInput{
		Repo:  "test-repo",
		Issue: 1,
		Task:  "Fix tests",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.NotEmpty(t, out.WorkspaceDir)
	assert.NotEmpty(t, out.Branch)
	assert.Contains(t, out.Branch, "agent/")
}
