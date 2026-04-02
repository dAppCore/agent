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
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
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

func TestPrep_NewPrep_Good_CoreHomeOverride(t *testing.T) {
	tmpHome := t.TempDir()
	claudeDir := core.JoinPath(tmpHome, ".claude")
	require.True(t, fs.EnsureDir(claudeDir).OK)
	require.True(t, fs.Write(core.JoinPath(claudeDir, "brain.key"), "core-home-key").OK)

	t.Setenv("CORE_HOME", tmpHome)
	t.Setenv("HOME", "/ignored-home")
	t.Setenv("DIR_HOME", "/ignored-dir")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CODE_PATH", "")

	s := NewPrep()
	assert.Equal(t, core.JoinPath(tmpHome, "Code"), s.codePath)
	assert.Equal(t, "core-home-key", s.brainKey)
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

func TestPrep_Subsystem_SetCore_Good_WiresServiceRuntime(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()

	s.SetCore(c)

	require.NotNil(t, s.ServiceRuntime)
	assert.Equal(t, c, s.Core())
}

func TestPrep_Subsystem_SetCore_Bad_NilCoreDoesNothing(t *testing.T) {
	s := &PrepSubsystem{}

	s.SetCore(nil)

	assert.Nil(t, s.ServiceRuntime)
}

func TestPrep_Subsystem_SetCore_Ugly_NilReceiverDoesNotPanic(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))

	assert.NotPanics(t, func() {
		var s *PrepSubsystem
		s.SetCore(c)
	})
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

// --- OnStartup Good/Bad/Ugly ---

func TestPrep_OnStartup_Good_CreatesPokeCh(t *testing.T) {
	// StartRunner is now a no-op — pokeCh is no longer initialised by OnStartup.
	// Verify OnStartup succeeds and pokeCh remains nil.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

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
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	assert.True(t, s.OnStartup(context.Background()).OK)
}

func TestPrep_OnStartup_Good_NoError(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	assert.True(t, s.OnStartup(context.Background()).OK)
}

func TestPrep_OnStartup_Good_RegistersPlanActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.True(t, c.Action("agentic.dispatch.sync").Exists())
	assert.True(t, c.Action("plan.create").Exists())
	assert.True(t, c.Action("plan.get").Exists())
	assert.True(t, c.Action("plan.read").Exists())
	assert.True(t, c.Action("plan.update").Exists())
	assert.True(t, c.Action("plan.update_status").Exists())
	assert.True(t, c.Action("plan.from.issue").Exists())
	assert.True(t, c.Action("plan.check").Exists())
	assert.True(t, c.Action("plan.archive").Exists())
	assert.True(t, c.Action("plan.delete").Exists())
	assert.True(t, c.Action("plan.list").Exists())
	assert.True(t, c.Action("phase.get").Exists())
	assert.True(t, c.Action("phase.update_status").Exists())
	assert.True(t, c.Action("phase.add_checkpoint").Exists())
	assert.True(t, c.Action("task.create").Exists())
	assert.True(t, c.Action("task.update").Exists())
	assert.True(t, c.Action("task.toggle").Exists())
}

func TestPrep_OnStartup_Good_RegistersDispatchControlActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.True(t, c.Action("agentic.dispatch.start").Exists())
	assert.True(t, c.Action("agentic.dispatch.shutdown").Exists())
	assert.True(t, c.Action("agentic.dispatch.shutdown_now").Exists())
}

func TestPrep_OnStartup_Good_RegistersSessionActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.True(t, c.Action("session.start").Exists())
	assert.True(t, c.Action("session.get").Exists())
	assert.True(t, c.Action("session.list").Exists())
	assert.True(t, c.Action("session.continue").Exists())
	assert.True(t, c.Action("session.end").Exists())
	assert.True(t, c.Action("session.complete").Exists())
	assert.True(t, c.Action("session.log").Exists())
	assert.True(t, c.Action("session.artifact").Exists())
	assert.True(t, c.Action("session.handoff").Exists())
	assert.True(t, c.Action("session.resume").Exists())
	assert.True(t, c.Action("session.replay").Exists())
	assert.True(t, c.Action("state.set").Exists())
	assert.True(t, c.Action("state.get").Exists())
	assert.True(t, c.Action("state.list").Exists())
	assert.True(t, c.Action("state.delete").Exists())
	assert.True(t, c.Action("issue.create").Exists())
	assert.True(t, c.Action("issue.get").Exists())
	assert.True(t, c.Action("issue.list").Exists())
	assert.True(t, c.Action("issue.update").Exists())
	assert.True(t, c.Action("issue.assign").Exists())
	assert.True(t, c.Action("issue.comment").Exists())
	assert.True(t, c.Action("issue.report").Exists())
	assert.True(t, c.Action("issue.archive").Exists())
	assert.True(t, c.Action("agentic.message.send").Exists())
	assert.True(t, c.Action("agent.message.send").Exists())
	assert.True(t, c.Action("agentic.message.inbox").Exists())
	assert.True(t, c.Action("agent.message.inbox").Exists())
	assert.True(t, c.Action("agentic.message.conversation").Exists())
	assert.True(t, c.Action("agent.message.conversation").Exists())
	assert.True(t, c.Action("agentic.issue.update").Exists())
	assert.True(t, c.Action("agentic.issue.create").Exists())
	assert.True(t, c.Action("agentic.issue.assign").Exists())
	assert.True(t, c.Action("agentic.issue.comment").Exists())
	assert.True(t, c.Action("agentic.issue.report").Exists())
	assert.True(t, c.Action("agentic.issue.archive").Exists())
	assert.True(t, c.Action("sprint.create").Exists())
	assert.True(t, c.Action("sprint.get").Exists())
	assert.True(t, c.Action("sprint.list").Exists())
	assert.True(t, c.Action("sprint.update").Exists())
	assert.True(t, c.Action("sprint.archive").Exists())
}

func TestPrep_OnStartup_Good_RegistersForgeActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.True(t, c.Action("agentic.pr.get").Exists())
	assert.True(t, c.Action("agentic.pr.list").Exists())
	assert.True(t, c.Action("agentic.pr.merge").Exists())
	assert.True(t, c.Action("agentic.pr.close").Exists())
	assert.True(t, c.Action("agentic.commit").Exists())
}

func TestPrep_OnStartup_Good_RegistersContentActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.True(t, c.Action("content.generate").Exists())
	assert.True(t, c.Action("content.batch").Exists())
	assert.True(t, c.Action("content.batch.generate").Exists())
	assert.True(t, c.Action("content.batch_generate").Exists())
	assert.True(t, c.Action("content_batch").Exists())
	assert.True(t, c.Action("content.brief.create").Exists())
	assert.True(t, c.Action("content.brief.get").Exists())
	assert.True(t, c.Action("content.brief.list").Exists())
	assert.True(t, c.Action("content.status").Exists())
	assert.True(t, c.Action("content.usage.stats").Exists())
	assert.True(t, c.Action("content.usage_stats").Exists())
	assert.True(t, c.Action("content.from.plan").Exists())
	assert.True(t, c.Action("content.from_plan").Exists())
	assert.True(t, c.Action("content.schema.generate").Exists())
}

func TestPrep_OnStartup_Good_RegistersPlatformActionAliases(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.True(t, c.Action("agentic.sync.push").Exists())
	assert.True(t, c.Action("agent.sync.push").Exists())
	assert.True(t, c.Action("agentic.auth.provision").Exists())
	assert.True(t, c.Action("agent.auth.provision").Exists())
	assert.True(t, c.Action("agentic.auth.revoke").Exists())
	assert.True(t, c.Action("agent.auth.revoke").Exists())
	assert.True(t, c.Action("agentic.fleet.register").Exists())
	assert.True(t, c.Action("agent.fleet.register").Exists())
	assert.True(t, c.Action("agentic.credits.balance").Exists())
	assert.True(t, c.Action("agent.credits.balance").Exists())
	assert.True(t, c.Action("agentic.fleet.events").Exists())
	assert.True(t, c.Action("agent.fleet.events").Exists())
	assert.True(t, c.Action("agentic.subscription.budget.update").Exists())
	assert.True(t, c.Action("agent.subscription.budget.update").Exists())
}

func TestPrep_OnStartup_Good_RegistersPlatformCommandAlias(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.Contains(t, c.Commands(), "auth/provision")
	assert.Contains(t, c.Commands(), "auth/revoke")
	assert.Contains(t, c.Commands(), "message/send")
	assert.Contains(t, c.Commands(), "messages/send")
	assert.Contains(t, c.Commands(), "message/inbox")
	assert.Contains(t, c.Commands(), "messages/inbox")
	assert.Contains(t, c.Commands(), "message/conversation")
	assert.Contains(t, c.Commands(), "messages/conversation")
	assert.Contains(t, c.Commands(), "subscription/budget/update")
	assert.Contains(t, c.Commands(), "subscription/update-budget")
	assert.Contains(t, c.Commands(), "fleet/events")
}

func TestPrep_RegisterTools_Good_RegistersCompletionTool(t *testing.T) {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, &mcpsdk.ServerOptions{
		Capabilities: &mcpsdk.ServerCapabilities{
			Tools: &mcpsdk.ToolCapabilities{ListChanged: true},
		},
	})

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(server)

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	require.NoError(t, err)

	var toolNames []string
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	assert.Contains(t, toolNames, "agentic_complete")
	assert.Contains(t, toolNames, "prompt_version")
	assert.Contains(t, toolNames, "agentic_prompt_version")
	assert.Contains(t, toolNames, "session_complete")
	assert.Contains(t, toolNames, "agentic_message_send")
	assert.Contains(t, toolNames, "agent_send")
	assert.Contains(t, toolNames, "agentic_message_inbox")
	assert.Contains(t, toolNames, "agent_inbox")
	assert.Contains(t, toolNames, "agentic_message_conversation")
	assert.Contains(t, toolNames, "agent_conversation")
}

func TestPrep_OnStartup_Good_RegistersGenerateCommand(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	require.True(t, s.OnStartup(context.Background()).OK)
	assert.Contains(t, c.Commands(), "generate")
	assert.Contains(t, c.Commands(), "agentic:generate")
	assert.Contains(t, c.Commands(), "complete")
	assert.Contains(t, c.Commands(), "dispatch/sync")
	assert.Contains(t, c.Commands(), "agentic:plan")
	assert.Contains(t, c.Commands(), "prep-workspace")
	assert.Contains(t, c.Commands(), "setup")
	assert.Contains(t, c.Commands(), "agentic:setup")
	assert.Contains(t, c.Commands(), "watch")
	assert.Contains(t, c.Commands(), "workspace/watch")
	assert.Contains(t, c.Commands(), "agentic:watch")
	assert.Contains(t, c.Commands(), "dispatch/start")
	assert.Contains(t, c.Commands(), "agentic:dispatch/start")
	assert.Contains(t, c.Commands(), "dispatch/shutdown")
	assert.Contains(t, c.Commands(), "agentic:dispatch/shutdown")
	assert.Contains(t, c.Commands(), "dispatch/shutdown-now")
	assert.Contains(t, c.Commands(), "agentic:dispatch/shutdown-now")
	assert.Contains(t, c.Commands(), "brain/ingest")
	assert.Contains(t, c.Commands(), "brain/seed-memory")
	assert.Contains(t, c.Commands(), "brain/list")
	assert.Contains(t, c.Commands(), "brain/forget")
	assert.Contains(t, c.Commands(), "lang/detect")
	assert.Contains(t, c.Commands(), "lang/list")
	assert.Contains(t, c.Commands(), "epic")
	assert.Contains(t, c.Commands(), "agentic:epic")
	assert.Contains(t, c.Commands(), "plan-cleanup")
	assert.Contains(t, c.Commands(), "commit")
	assert.Contains(t, c.Commands(), "agentic:commit")
	assert.Contains(t, c.Commands(), "plan/from-issue")
	assert.Contains(t, c.Commands(), "session/end")
	assert.Contains(t, c.Commands(), "agentic:session/end")
	assert.Contains(t, c.Commands(), "session/resume")
	assert.Contains(t, c.Commands(), "session/replay")
	assert.Contains(t, c.Commands(), "review-queue")
	assert.Contains(t, c.Commands(), "agentic:review-queue")
	assert.Contains(t, c.Commands(), "flow/preview")
	assert.Contains(t, c.Commands(), "agentic:flow/preview")
	assert.Contains(t, c.Commands(), "prompt/version")
	assert.Contains(t, c.Commands(), "agentic:prompt/version")
	assert.True(t, c.Action("agentic.prompt.version").Exists())
	assert.Contains(t, c.Commands(), "task")
	assert.Contains(t, c.Commands(), "task/create")
	assert.Contains(t, c.Commands(), "task/update")
	assert.Contains(t, c.Commands(), "task/toggle")
	assert.Contains(t, c.Commands(), "state")
	assert.Contains(t, c.Commands(), "state/set")
	assert.Contains(t, c.Commands(), "state/get")
	assert.Contains(t, c.Commands(), "state/list")
	assert.Contains(t, c.Commands(), "state/delete")
}

func TestPrep_OnStartup_Bad(t *testing.T) {
	// OnStartup with nil ServiceRuntime — panics because
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
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

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

// --- PrepareWorkspace ---

func TestPrep_PrepareWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Valid input but repo won't exist — still exercises the public wrapper delegation
	_, _, err := s.PrepareWorkspace(context.Background(), PrepInput{
		Repo:  "go-io",
		Issue: 1,
	})
	// Error expected (no local clone) but we verified it delegates to prepWorkspace
	assert.Error(t, err)
}

func TestPrep_PrepareWorkspace_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Missing repo — should return error
	_, _, err := s.PrepareWorkspace(context.Background(), PrepInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestPrep_PrepareWorkspace_Ugly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Bare ".." is caught as invalid repo name by PathBase check
	_, _, err := s.PrepareWorkspace(context.Background(), PrepInput{
		Repo:  "..",
		Issue: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid repo name")
}

// --- BuildPrompt ---

func TestPrep_BuildPrompt_Good(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	prompt, memories, consumers := s.BuildPrompt(context.Background(), PrepInput{
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

func TestPrep_BuildPrompt_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Empty inputs — should still return a prompt string without panicking
	prompt, memories, consumers := s.BuildPrompt(context.Background(), PrepInput{}, "", "")
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "TASK:")
	assert.Contains(t, prompt, "CONSTRAINTS:")
	assert.Equal(t, 0, memories)
	assert.Equal(t, 0, consumers)
}

func TestPrep_BuildPrompt_Ugly(t *testing.T) {
	dir := t.TempDir()

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Unicode in all fields — should not panic
	prompt, _, _ := s.BuildPrompt(context.Background(), PrepInput{
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

	_, out, err := s.PrepareWorkspace(context.Background(), PrepInput{
		Repo:  "test-repo",
		Issue: 1,
		Task:  "Fix tests",
	})
	require.NoError(t, err)
	assert.True(t, out.Success)
	assert.NotEmpty(t, out.WorkspaceDir)
	assert.NotEmpty(t, out.Branch)
	assert.Contains(t, out.Branch, "agent/")
	assert.NotEmpty(t, out.PromptVersion)

	promptIndexPath := core.JoinPath(WorkspaceMetaDir(out.WorkspaceDir), "prompt-version.json")
	require.True(t, fs.Exists(promptIndexPath))
	promptIndexResult := fs.Read(promptIndexPath)
	require.True(t, promptIndexResult.OK)

	var promptSnapshot PromptVersionSnapshot
	require.True(t, core.JSONUnmarshalString(promptIndexResult.Value.(string), &promptSnapshot).OK)
	assert.Equal(t, out.PromptVersion, promptSnapshot.Hash)
	assert.Contains(t, promptSnapshot.Content, "TASK: Fix tests")

	promptSnapshotPath := core.JoinPath(WorkspaceMetaDir(out.WorkspaceDir), "prompt-versions", core.Concat(out.PromptVersion, ".json"))
	require.True(t, fs.Exists(promptSnapshotPath))
}

func TestPrep_TestPrepWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CORE_WORKSPACE", root)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(core.JSONMarshalString(map[string]any{
			"number": 1,
			"title":  "Fix tests",
			"body":   "Tests are broken",
		})))
	}))
	t.Cleanup(srv.Close)

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
}

func TestPrep_TestPrepWorkspace_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.TestPrepWorkspace(context.Background(), PrepInput{Repo: "."})
	require.Error(t, err)
}

func TestPrep_TestPrepWorkspace_Ugly(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.TestPrepWorkspace(context.Background(), PrepInput{Repo: ".."})
	require.Error(t, err)
}
