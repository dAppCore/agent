// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/forge"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestPrep_EnvOr_Good_EnvSet(t *testing.T) {
	t.Setenv("TEST_ENVVAR_CUSTOM", "custom-value")
	want := "custom-value"
	got := envOr("TEST_ENVVAR_CUSTOM", "default")
	core.AssertEqual(t, want, got)
}

func TestPrep_EnvOr_Good_Fallback(t *testing.T) {
	t.Setenv("TEST_ENVVAR_MISSING", "")
	want := "default-value"
	got := envOr("TEST_ENVVAR_MISSING", "default-value")
	core.AssertEqual(t, want, got)
}

func TestPrep_EnvOr_Good_UnsetUsesFallback(t *testing.T) {
	t.Setenv("TEST_ENVVAR_TOTALLY_MISSING", "")
	want := "fallback"
	got := envOr("TEST_ENVVAR_TOTALLY_MISSING", "fallback")
	core.AssertEqual(t, want, got)
}

func TestPrep_DetectLanguage_Good_Go(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)
	core.AssertEqual(t, "go", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_PHP(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "composer.json"), "{}").OK)
	core.AssertEqual(t, "php", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_TypeScript(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "package.json"), "{}").OK)
	core.AssertEqual(t, "ts", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Rust(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "Cargo.toml"), "[package]").OK)
	core.AssertEqual(t, "rust", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Python(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "requirements.txt"), "flask").OK)
	core.AssertEqual(t, "py", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Cpp(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "CMakeLists.txt"), "cmake_minimum_required").OK)
	core.AssertEqual(t, "cpp", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_Docker(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "Dockerfile"), "FROM alpine").OK)
	core.AssertEqual(t, "docker", detectLanguage(dir))
}

func TestPrep_DetectLanguage_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	want := "go"
	got := detectLanguage(dir)
	core.AssertEqual(t, want, got)
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
			core.RequireTrue(t, fs.Write(core.JoinPath(dir, tt.file), tt.content).OK)
			core.AssertEqual(t, tt.expected, detectBuildCmd(dir))
		})
	}
}

func TestPrep_DetectBuildCmd_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	want := "go build ./..."
	got := detectBuildCmd(dir)
	core.AssertEqual(t, want, got)
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
			core.RequireTrue(t, fs.Write(core.JoinPath(dir, tt.file), tt.content).OK)
			core.AssertEqual(t, tt.expected, detectTestCmd(dir))
		})
	}
}

func TestPrep_DetectTestCmd_Good_DefaultsToGo(t *testing.T) {
	dir := t.TempDir()
	want := "go test ./..."
	got := detectTestCmd(dir)
	core.AssertEqual(t, want, got)
}

func TestSanitise_SanitiseBranchSlug_Good(t *testing.T) {
	first := sanitiseBranchSlug("Fix login bug!", 40)
	second := sanitiseBranchSlug("---Trim Me---", 40)
	core.AssertEqual(t, "fix-login-bug", first)
	core.AssertEqual(t, "trim-me", second)
}

func TestSanitise_SanitiseBranchSlug_Good_Truncates(t *testing.T) {
	input := "feature--extra"
	got := sanitiseBranchSlug(input, 7)
	core.AssertEqual(t, "feature", got)
}

func TestSanitise_SanitiseFilename_Good(t *testing.T) {
	input := "Core / Agent:Notes"
	got := sanitiseFilename(input)
	core.AssertEqual(t, "Core---Agent-Notes", got)
}

func TestPrepDefaults_NewPrep_Good(t *testing.T) {
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("FORGE_URL", "")
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("SPECS_PATH", "")
	t.Setenv("CODE_PATH", "")

	s := NewPrep()
	core.AssertEqual(t, "https://forge.lthn.ai", s.forgeURL)
	core.AssertEqual(t, "https://api.lthn.sh", s.brainURL)
	core.AssertNotEmpty(t, s.codePath)
}

func TestPrep_NewPrep_Good_EnvOverrides(t *testing.T) {
	t.Setenv("FORGE_URL", "https://custom-forge.example.com")
	t.Setenv("FORGE_TOKEN", "test-token")
	t.Setenv("CORE_BRAIN_URL", "https://custom-brain.example.com")
	t.Setenv("CORE_BRAIN_KEY", "brain-key-123")
	t.Setenv("SPECS_PATH", "/custom/specs")
	t.Setenv("CODE_PATH", "/custom/code")

	s := NewPrep()
	core.AssertEqual(t, "https://custom-forge.example.com", s.forgeURL)
	core.AssertEqual(t, "test-token", s.forgeToken)
	core.AssertEqual(t, "https://custom-brain.example.com", s.brainURL)
	core.AssertEqual(t, "brain-key-123", s.brainKey)
	core.AssertEqual(t, "/custom/code", s.codePath)
}

func TestPrep_NewPrep_Good_CoreHomeOverride(t *testing.T) {
	tmpHome := t.TempDir()
	claudeDir := core.JoinPath(tmpHome, ".claude")
	core.RequireTrue(t, fs.EnsureDir(claudeDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(claudeDir, "brain.key"), "core-home-key").OK)

	t.Setenv("CORE_HOME", tmpHome)
	t.Setenv("HOME", "/ignored-home")
	t.Setenv("DIR_HOME", "/ignored-dir")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("CODE_PATH", "")

	s := NewPrep()
	core.AssertEqual(t, core.JoinPath(tmpHome, "Code"), s.codePath)
	core.AssertEqual(t, "core-home-key", s.brainKey)
}

func TestPrep_NewPrep_Good_GiteaTokenFallback(t *testing.T) {
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "gitea-fallback-token")

	s := NewPrep()
	core.AssertEqual(t, "gitea-fallback-token", s.forgeToken)
}

func TestPrepName_PrepSubsystem_Name_Good(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	name := s.Name()
	core.AssertNotEmpty(t, name)
	core.AssertEqual(t, "agentic", name)
}

func TestPrepSetCore_PrepSubsystem_SetCore_Good(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()

	s.SetCore(c)

	core.AssertNotNil(t, s.ServiceRuntime)
	core.AssertEqual(t, c, s.Core())
}

func TestPrepNilCore_PrepSubsystem_SetCore_Bad(t *testing.T) {
	s := &PrepSubsystem{}

	s.SetCore(nil)

	core.AssertNil(t, s.ServiceRuntime)
}

func TestPrepNilReceiver_PrepSubsystem_SetCore_Ugly(t *testing.T) {
	c := core.New(core.WithOption("name", "test"))

	core.AssertNotPanics(t, func() {
		var s *PrepSubsystem
		s.SetCore(c)
	})
}

// --- sanitiseBranchSlug Bad/Ugly ---

func TestSanitise_SanitiseBranchSlug_Bad_EmptyString(t *testing.T) {
	input := ""
	got := sanitiseBranchSlug(input, 40)
	core.AssertEqual(t, "", got)
}

func TestSanitise_SanitiseBranchSlug_Bad_OnlySpecialChars(t *testing.T) {
	input := "!@#$%^&*()"
	got := sanitiseBranchSlug(input, 40)
	core.AssertEqual(t, "", got)
}

func TestSanitise_SanitiseBranchSlug_Bad_OnlyDashes(t *testing.T) {
	input := "------"
	got := sanitiseBranchSlug(input, 40)
	core.AssertEqual(t, "", got)
}

func TestSanitise_SanitiseBranchSlug_Ugly_VeryLongString(t *testing.T) {
	long := strings.Repeat("abcdefghij", 100)
	result := sanitiseBranchSlug(long, 50)
	core.AssertLessOrEqual(t, len(result), 50)
}

func TestSanitise_SanitiseBranchSlug_Ugly_Unicode(t *testing.T) {
	// Unicode chars should be replaced with dashes, then edges trimmed
	result := sanitiseBranchSlug("\u00e9\u00e0\u00fc\u00f1\u00f0", 40)
	core.AssertNotContains(t, result, "\u00e9")
	// All replaced with dashes, then trimmed = empty
	core.AssertEqual(t, "", result)
}

func TestSanitise_SanitiseBranchSlug_Ugly_ZeroMax(t *testing.T) {
	// max=0 means no limit
	result := sanitiseBranchSlug("hello-world", 0)
	core.AssertNotEmpty(t, result)
	core.AssertEqual(t, "hello-world", result)
}

// --- sanitisePlanSlug Bad/Ugly ---

func TestSanitise_SanitisePlanSlug_Bad_EmptyString(t *testing.T) {
	input := ""
	got := sanitisePlanSlug(input)
	core.AssertEqual(t, "", got)
}

func TestSanitise_SanitisePlanSlug_Bad_OnlySpecialChars(t *testing.T) {
	input := "!@#$%^&*()"
	got := sanitisePlanSlug(input)
	core.AssertEqual(t, "", got)
}

func TestSanitise_SanitisePlanSlug_Bad_OnlySpaces(t *testing.T) {
	// Spaces become dashes, then collapsed, then trimmed
	input := "     "
	got := sanitisePlanSlug(input)
	core.AssertEqual(t, "", got)
}

func TestSanitise_SanitisePlanSlug_Ugly_VeryLongString(t *testing.T) {
	long := strings.Repeat("abcdefghij ", 20)
	result := sanitisePlanSlug(long)
	core.AssertLessOrEqual(t, len(result), 30)
}

func TestSanitise_SanitisePlanSlug_Ugly_Unicode(t *testing.T) {
	result := sanitisePlanSlug("\u00e9\u00e0\u00fc\u00f1\u00f0")
	core.AssertNotContains(t, result, "\u00e9")
	core.AssertEqual(t, "", result, "unicode chars should be stripped, leaving empty string")
}

func TestSanitise_SanitisePlanSlug_Ugly_AllDashInput(t *testing.T) {
	input := "---"
	got := sanitisePlanSlug(input)
	core.AssertEqual(t, "", got)
}

// --- sanitiseFilename Bad/Ugly ---

func TestSanitise_SanitiseFilename_Bad_EmptyString(t *testing.T) {
	input := ""
	got := sanitiseFilename(input)
	core.AssertEqual(t, "", got)
}

func TestSanitise_SanitiseFilename_Bad_OnlySpecialChars(t *testing.T) {
	result := sanitiseFilename("!@#$%^&*()")
	// All replaced with dashes
	core.AssertLen(t, result, 10)
	core.AssertEqual(t, "----------", result)
}

func TestSanitise_SanitiseFilename_Ugly_VeryLongString(t *testing.T) {
	long := strings.Repeat("a", 1000)
	result := sanitiseFilename(long)
	core.AssertEqual(t, 1000, len(result))
}

func TestSanitise_SanitiseFilename_Ugly_Unicode(t *testing.T) {
	result := sanitiseFilename("\u00e9\u00e0\u00fc\u00f1\u00f0")
	// All replaced with dashes
	for _, r := range result {
		core.AssertEqual(t, '-', r)
	}
}

func TestSanitise_SanitiseFilename_Ugly_PreservesDotsUnderscores(t *testing.T) {
	input := "my_file.test.txt"
	got := sanitiseFilename(input)
	core.AssertEqual(t, "my_file.test.txt", got)
}

// --- collapseRepeatedRune Bad/Ugly ---

func TestSanitise_CollapseRepeatedRune_Bad_EmptyString(t *testing.T) {
	input := ""
	got := collapseRepeatedRune(input, '-')
	core.AssertEqual(t, "", got)
}

func TestSanitise_CollapseRepeatedRune_Bad_AllTarget(t *testing.T) {
	input := "-----"
	got := collapseRepeatedRune(input, '-')
	core.AssertEqual(t, "-", got)
}

func TestSanitise_CollapseRepeatedRune_Ugly_Unicode(t *testing.T) {
	input := "h\u00e9\u00e9\u00e9llo"
	got := collapseRepeatedRune(input, '\u00e9')
	core.AssertEqual(t, "h\u00e9llo", got)
}

func TestSanitise_CollapseRepeatedRune_Ugly_VeryLong(t *testing.T) {
	long := strings.Repeat("--a", 500)
	result := collapseRepeatedRune(long, '-')
	core.AssertNotContains(t, result, "--")
}

// --- trimRuneEdges Bad/Ugly ---

func TestSanitise_TrimRuneEdges_Bad_EmptyString(t *testing.T) {
	input := ""
	got := trimRuneEdges(input, '-')
	core.AssertEqual(t, "", got)
}

func TestSanitise_TrimRuneEdges_Bad_AllTarget(t *testing.T) {
	input := "-----"
	got := trimRuneEdges(input, '-')
	core.AssertEqual(t, "", got)
}

func TestSanitise_TrimRuneEdges_Ugly_Unicode(t *testing.T) {
	input := "\u00e9hello\u00e9\u00e9"
	got := trimRuneEdges(input, '\u00e9')
	core.AssertEqual(t, "hello", got)
}

func TestSanitise_TrimRuneEdges_Ugly_NoMatch(t *testing.T) {
	input := "hello"
	got := trimRuneEdges(input, '-')
	core.AssertEqual(t, "hello", got)
}

// --- PrepSubsystem Name Bad/Ugly ---

func TestPrepName_PrepSubsystem_Name_Bad(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	name := s.Name()
	core.AssertNotEmpty(t, name, "Name should never return empty")
	core.AssertEqual(t, "agentic", name)
}

func TestPrepName_PrepSubsystem_Name_Ugly(t *testing.T) {
	// Zero-value struct — Name() should still work
	var s PrepSubsystem
	core.AssertNotPanics(t, func() {
		name := s.Name()
		core.AssertEqual(t, "agentic", name)
	})
}

// --- NewPrep Bad/Ugly ---

func TestPrepBad_NewPrep_Bad(t *testing.T) {
	// Call without any env — verify doesn't panic, returns valid struct
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "")
	t.Setenv("CORE_BRAIN_KEY", "")
	t.Setenv("FORGE_URL", "")
	t.Setenv("CORE_BRAIN_URL", "")
	t.Setenv("SPECS_PATH", "")
	t.Setenv("CODE_PATH", "")

	core.AssertNotPanics(t, func() {
		s := NewPrep()
		core.AssertNotNil(t, s)
	})
}

func TestPrepUgly_NewPrep_Ugly(t *testing.T) {
	// Verify returned struct has non-nil backoff/failCount maps
	t.Setenv("FORGE_TOKEN", "")
	t.Setenv("GITEA_TOKEN", "")

	s := NewPrep()
	core.AssertNotNil(t, s.backoff, "backoff map must not be nil")
	core.AssertNotNil(t, s.failCount, "failCount map must not be nil")
	core.AssertNotNil(t, s.forge, "Forge client must not be nil")
}

// --- OnStartup Good/Bad/Ugly ---

func TestPrepStartup_PrepSubsystem_OnStartup_Good(t *testing.T) {
	// StartRunner is now a no-op — pokeCh is no longer initialised by OnStartup.
	// Verify OnStartup succeeds and pokeCh remains nil.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.AssertNil(t, s.pokeCh, "pokeCh should be nil before OnStartup")

	r := s.OnStartup(context.Background())
	core.AssertTrue(t, r.OK)

	core.AssertNil(t, s.pokeCh, "pokeCh should remain nil — queue drain is owned by pkg/runner")
}

func TestPrep_OnStartup_Good_FrozenByDefault(t *testing.T) {
	// Frozen state is now owned by pkg/runner.Service, not agentic.
	// Verify OnStartup succeeds without asserting frozen state.
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.AssertTrue(t, s.OnStartup(context.Background()).OK)
}

func TestPrep_OnStartup_Good_NoError(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.AssertTrue(t, s.OnStartup(context.Background()).OK)
}

func TestPrep_OnStartup_Good_RegistersPlanActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("agentic.dispatch.sync").Exists())
	core.AssertTrue(t, c.Action("plan.create").Exists())
	core.AssertTrue(t, c.Action("plan.get").Exists())
	core.AssertTrue(t, c.Action("plan.read").Exists())
	core.AssertTrue(t, c.Action("plan.update").Exists())
	core.AssertTrue(t, c.Action("plan.update_status").Exists())
	core.AssertTrue(t, c.Action("plan.from.issue").Exists())
	core.AssertTrue(t, c.Action("plan.check").Exists())
	core.AssertTrue(t, c.Action("plan.archive").Exists())
	core.AssertTrue(t, c.Action("plan.delete").Exists())
	core.AssertTrue(t, c.Action("plan.list").Exists())
	core.AssertTrue(t, c.Action("phase.get").Exists())
	core.AssertTrue(t, c.Action("phase.update_status").Exists())
	core.AssertTrue(t, c.Action("phase.add_checkpoint").Exists())
	core.AssertTrue(t, c.Action("task.create").Exists())
	core.AssertTrue(t, c.Action("task.update").Exists())
	core.AssertTrue(t, c.Action("task.toggle").Exists())
}

func TestPrep_OnStartup_Good_RegistersDispatchControlActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("agentic.dispatch.start").Exists())
	core.AssertTrue(t, c.Action("agentic.dispatch.shutdown").Exists())
	core.AssertTrue(t, c.Action("agentic.dispatch.shutdown_now").Exists())
}

func TestPrep_OnStartup_Good_RegistersSessionActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("session.start").Exists())
	core.AssertTrue(t, c.Action("session.get").Exists())
	core.AssertTrue(t, c.Action("session.list").Exists())
	core.AssertTrue(t, c.Action("session.continue").Exists())
	core.AssertTrue(t, c.Action("session.end").Exists())
	core.AssertTrue(t, c.Action("session.complete").Exists())
	core.AssertTrue(t, c.Action("session.log").Exists())
	core.AssertTrue(t, c.Action("session.artifact").Exists())
	core.AssertTrue(t, c.Action("session.handoff").Exists())
	core.AssertTrue(t, c.Action("session.resume").Exists())
	core.AssertTrue(t, c.Action("session.replay").Exists())
	core.AssertTrue(t, c.Action("state.set").Exists())
	core.AssertTrue(t, c.Action("state.get").Exists())
	core.AssertTrue(t, c.Action("state.list").Exists())
	core.AssertTrue(t, c.Action("state.delete").Exists())
	core.AssertTrue(t, c.Action("issue.create").Exists())
	core.AssertTrue(t, c.Action("issue.get").Exists())
	core.AssertTrue(t, c.Action("issue.list").Exists())
	core.AssertTrue(t, c.Action("issue.update").Exists())
	core.AssertTrue(t, c.Action("issue.assign").Exists())
	core.AssertTrue(t, c.Action("issue.comment").Exists())
	core.AssertTrue(t, c.Action("issue.report").Exists())
	core.AssertTrue(t, c.Action("issue.archive").Exists())
	core.AssertTrue(t, c.Action("agentic.message.send").Exists())
	core.AssertTrue(t, c.Action("agent.message.send").Exists())
	core.AssertTrue(t, c.Action("agentic.message.inbox").Exists())
	core.AssertTrue(t, c.Action("agent.message.inbox").Exists())
	core.AssertTrue(t, c.Action("agentic.message.conversation").Exists())
	core.AssertTrue(t, c.Action("agent.message.conversation").Exists())
	core.AssertTrue(t, c.Action("agentic.issue.update").Exists())
	core.AssertTrue(t, c.Action("agentic.issue.create").Exists())
	core.AssertTrue(t, c.Action("agentic.issue.assign").Exists())
	core.AssertTrue(t, c.Action("agentic.issue.comment").Exists())
	core.AssertTrue(t, c.Action("agentic.issue.report").Exists())
	core.AssertTrue(t, c.Action("agentic.issue.archive").Exists())
	core.AssertTrue(t, c.Action("sprint.create").Exists())
	core.AssertTrue(t, c.Action("sprint.get").Exists())
	core.AssertTrue(t, c.Action("sprint.list").Exists())
	core.AssertTrue(t, c.Action("sprint.update").Exists())
	core.AssertTrue(t, c.Action("sprint.archive").Exists())
}

func TestPrep_OnStartup_Good_RegistersNamespacedActionAliases(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("agentic.plan.create").Exists())
	core.AssertTrue(t, c.Action("agentic.plan.read").Exists())
	core.AssertTrue(t, c.Action("agentic.phase.get").Exists())
	core.AssertTrue(t, c.Action("agentic.task.create").Exists())
	core.AssertTrue(t, c.Action("agentic.session.start").Exists())
	core.AssertTrue(t, c.Action("agentic.state.set").Exists())
	core.AssertTrue(t, c.Action("agentic.content.generate").Exists())
	core.AssertTrue(t, c.Action("agentic.content.schema.generate").Exists())
}

func TestPrep_OnStartup_Good_RegistersForgeActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("agentic.pr.get").Exists())
	core.AssertTrue(t, c.Action("agentic.pr.list").Exists())
	core.AssertTrue(t, c.Action("agentic.pr.merge").Exists())
	core.AssertTrue(t, c.Action("agentic.pr.close").Exists())
	core.AssertTrue(t, c.Action("agentic.commit").Exists())
}

func TestPrep_OnStartup_Good_RegistersContentActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("content.generate").Exists())
	core.AssertTrue(t, c.Action("agentic.generate").Exists())
	core.AssertTrue(t, c.Action("content.batch").Exists())
	core.AssertTrue(t, c.Action("content.batch.generate").Exists())
	core.AssertTrue(t, c.Action("content.batch_generate").Exists())
	core.AssertTrue(t, c.Action("content_batch").Exists())
	core.AssertTrue(t, c.Action("content.brief.create").Exists())
	core.AssertTrue(t, c.Action("content.brief.get").Exists())
	core.AssertTrue(t, c.Action("content.brief.list").Exists())
	core.AssertTrue(t, c.Action("content.status").Exists())
	core.AssertTrue(t, c.Action("content.usage.stats").Exists())
	core.AssertTrue(t, c.Action("content.usage_stats").Exists())
	core.AssertTrue(t, c.Action("content.from.plan").Exists())
	core.AssertTrue(t, c.Action("content.from_plan").Exists())
	core.AssertTrue(t, c.Action("content.schema.generate").Exists())
	core.AssertTrue(t, c.Action("agentic.content.generate").Exists())
	core.AssertTrue(t, c.Action("agentic.content.batch").Exists())
	core.AssertTrue(t, c.Action("agentic.content.schema.generate").Exists())
}

func TestPrep_OnStartup_Good_RegistersTemplateActions(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("template.list").Exists())
	core.AssertTrue(t, c.Action("agentic.template.list").Exists())
	core.AssertTrue(t, c.Action("template.preview").Exists())
	core.AssertTrue(t, c.Action("agentic.template.preview").Exists())
	core.AssertTrue(t, c.Action("template.create_plan").Exists())
	core.AssertTrue(t, c.Action("agentic.template.create_plan").Exists())
}

func TestPrep_OnStartup_Good_RegistersPlatformActionAliases(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertTrue(t, c.Action("agentic.sync.push").Exists())
	core.AssertTrue(t, c.Action("agent.sync.push").Exists())
	core.AssertTrue(t, c.Action("agentic.auth.provision").Exists())
	core.AssertTrue(t, c.Action("agent.auth.provision").Exists())
	core.AssertTrue(t, c.Action("agentic.auth.revoke").Exists())
	core.AssertTrue(t, c.Action("agent.auth.revoke").Exists())
	core.AssertTrue(t, c.Action("agentic.auth.login").Exists())
	core.AssertTrue(t, c.Action("agent.auth.login").Exists())
	core.AssertTrue(t, c.Action("agentic.fleet.register").Exists())
	core.AssertTrue(t, c.Action("agent.fleet.register").Exists())
	core.AssertTrue(t, c.Action("agentic.credits.balance").Exists())
	core.AssertTrue(t, c.Action("agent.credits.balance").Exists())
	core.AssertTrue(t, c.Action("agentic.fleet.events").Exists())
	core.AssertTrue(t, c.Action("agent.fleet.events").Exists())
	core.AssertTrue(t, c.Action("agentic.subscription.budget.update").Exists())
	core.AssertTrue(t, c.Action("agent.subscription.budget.update").Exists())
}

func TestPrep_OnStartup_Good_RegistersPlatformCommandAlias(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertContains(t, c.Commands(), "auth/provision")
	core.AssertContains(t, c.Commands(), "agentic:auth/provision")
	core.AssertContains(t, c.Commands(), "auth/revoke")
	core.AssertContains(t, c.Commands(), "agentic:auth/revoke")
	core.AssertContains(t, c.Commands(), "message/send")
	core.AssertContains(t, c.Commands(), "messages/send")
	core.AssertContains(t, c.Commands(), "agentic:message/send")
	core.AssertContains(t, c.Commands(), "agentic:messages/send")
	core.AssertContains(t, c.Commands(), "message/inbox")
	core.AssertContains(t, c.Commands(), "messages/inbox")
	core.AssertContains(t, c.Commands(), "agentic:message/inbox")
	core.AssertContains(t, c.Commands(), "agentic:messages/inbox")
	core.AssertContains(t, c.Commands(), "message/conversation")
	core.AssertContains(t, c.Commands(), "messages/conversation")
	core.AssertContains(t, c.Commands(), "agentic:message/conversation")
	core.AssertContains(t, c.Commands(), "agentic:messages/conversation")
	core.AssertContains(t, c.Commands(), "subscription/budget/update")
	core.AssertContains(t, c.Commands(), "subscription/update-budget")
	core.AssertContains(t, c.Commands(), "agentic:subscription/budget/update")
	core.AssertContains(t, c.Commands(), "agentic:subscription/update-budget")
	core.AssertContains(t, c.Commands(), "fleet/events")
	core.AssertContains(t, c.Commands(), "agentic:fleet/events")
}

func TestRegistersCompletionTool_PrepSubsystem_RegisterTools_Good(t *testing.T) {
	t.Setenv("CORE_MCP_FULL", "1")
	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(svc)

	server := svc.Server()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	core.RequireNoError(t, err)

	var toolNames []string
	for _, tool := range result.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	core.AssertContains(t, toolNames, "agentic_complete")
	core.AssertContains(t, toolNames, "prompt_version")
	core.AssertContains(t, toolNames, "agentic_prompt_version")
	core.AssertContains(t, toolNames, "agentic_setup")
	core.AssertContains(t, toolNames, "agentic_issue_create")
	core.AssertContains(t, toolNames, "agentic_issue_assign")
	core.AssertContains(t, toolNames, "agentic_session_start")
	core.AssertContains(t, toolNames, "agentic_task_create")
	core.AssertContains(t, toolNames, "agentic_state_set")
	core.AssertContains(t, toolNames, "agentic_sprint_create")
	core.AssertContains(t, toolNames, "agentic_sprint_start")
	core.AssertContains(t, toolNames, "agentic_sprint_complete")
	core.AssertContains(t, toolNames, "session_complete")
	core.AssertContains(t, toolNames, "agentic_message_send")
	core.AssertContains(t, toolNames, "agent_send")
	core.AssertContains(t, toolNames, "agentic_message_inbox")
	core.AssertContains(t, toolNames, "agent_inbox")
	core.AssertContains(t, toolNames, "agentic_message_conversation")
	core.AssertContains(t, toolNames, "agent_conversation")
	// RFC §9 pairing-code bootstrap exposes the login flow as an MCP tool so
	// IDE/CLI callers can exchange a 6-digit code for an AgentApiKey without
	// shelling out.
	core.AssertContains(t, toolNames, "agentic_auth_login")
	core.AssertContains(t, toolNames, "agentic_auth_provision")
	core.AssertContains(t, toolNames, "agentic_auth_revoke")
}

func TestPrep_OnStartup_Good_RegistersGenerateCommand(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("CORE_AGENT_DISPATCH", "")

	c := core.New(core.WithOption("name", "test"))
	s := NewPrep()
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.RequireTrue(t, s.OnStartup(context.Background()).OK)
	core.AssertContains(t, c.Commands(), "generate")
	core.AssertContains(t, c.Commands(), "agentic:generate")
	core.AssertContains(t, c.Commands(), "complete")
	core.AssertContains(t, c.Commands(), "dispatch/sync")
	core.AssertContains(t, c.Commands(), "agentic:plan")
	core.AssertContains(t, c.Commands(), "prep-workspace")
	core.AssertContains(t, c.Commands(), "setup")
	core.AssertContains(t, c.Commands(), "agentic:setup")
	core.AssertTrue(t, c.Action("agentic.setup").Exists())
	core.AssertContains(t, c.Commands(), "watch")
	core.AssertContains(t, c.Commands(), "workspace/watch")
	core.AssertContains(t, c.Commands(), "agentic:watch")
	core.AssertContains(t, c.Commands(), "dispatch/start")
	core.AssertContains(t, c.Commands(), "agentic:dispatch/start")
	core.AssertContains(t, c.Commands(), "dispatch/shutdown")
	core.AssertContains(t, c.Commands(), "agentic:dispatch/shutdown")
	core.AssertContains(t, c.Commands(), "dispatch/shutdown-now")
	core.AssertContains(t, c.Commands(), "agentic:dispatch/shutdown-now")
	core.AssertContains(t, c.Commands(), "brain/ingest")
	core.AssertContains(t, c.Commands(), "brain/seed-memory")
	core.AssertContains(t, c.Commands(), "brain/list")
	core.AssertContains(t, c.Commands(), "brain/forget")
	core.AssertContains(t, c.Commands(), "lang/detect")
	core.AssertContains(t, c.Commands(), "lang/list")
	core.AssertContains(t, c.Commands(), "epic")
	core.AssertContains(t, c.Commands(), "agentic:epic")
	core.AssertContains(t, c.Commands(), "plan-cleanup")
	core.AssertContains(t, c.Commands(), "commit")
	core.AssertContains(t, c.Commands(), "agentic:commit")
	core.AssertContains(t, c.Commands(), "plan/from-issue")
	core.AssertContains(t, c.Commands(), "session/end")
	core.AssertContains(t, c.Commands(), "agentic:session/end")
	core.AssertContains(t, c.Commands(), "session/resume")
	core.AssertContains(t, c.Commands(), "session/replay")
	core.AssertContains(t, c.Commands(), "review-queue")
	core.AssertContains(t, c.Commands(), "agentic:review-queue")
	core.AssertContains(t, c.Commands(), "flow/preview")
	core.AssertContains(t, c.Commands(), "agentic:flow/preview")
	core.AssertContains(t, c.Commands(), "prompt")
	core.AssertContains(t, c.Commands(), "agentic:prompt")
	core.AssertContains(t, c.Commands(), "prompt/version")
	core.AssertContains(t, c.Commands(), "agentic:prompt/version")
	core.AssertTrue(t, c.Action("agentic.prompt.version").Exists())
	core.AssertContains(t, c.Commands(), "task")
	core.AssertContains(t, c.Commands(), "task/create")
	core.AssertContains(t, c.Commands(), "task/update")
	core.AssertContains(t, c.Commands(), "task/toggle")
	core.AssertContains(t, c.Commands(), "phase")
	core.AssertContains(t, c.Commands(), "agentic:phase")
	core.AssertContains(t, c.Commands(), "phase/get")
	core.AssertContains(t, c.Commands(), "agentic:phase/get")
	core.AssertContains(t, c.Commands(), "phase/update_status")
	core.AssertContains(t, c.Commands(), "agentic:phase/update_status")
	core.AssertContains(t, c.Commands(), "phase/add_checkpoint")
	core.AssertContains(t, c.Commands(), "agentic:phase/add_checkpoint")
	core.AssertContains(t, c.Commands(), "state")
	core.AssertContains(t, c.Commands(), "state/set")
	core.AssertContains(t, c.Commands(), "state/get")
	core.AssertContains(t, c.Commands(), "state/list")
	core.AssertContains(t, c.Commands(), "state/delete")
}

func TestPrepStartup_PrepSubsystem_OnStartup_Bad(t *testing.T) {
	// OnStartup with nil ServiceRuntime — panics because
	// registerCommands calls s.Core().Command().
	s := &PrepSubsystem{
		ServiceRuntime: nil,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	core.AssertPanics(t, func() {
		_ = s.OnStartup(context.Background())
	}, "OnStartup without core should panic on registerCommands")
}

func TestPrepStartup_PrepSubsystem_OnStartup_Ugly(t *testing.T) {
	// OnStartup called twice with valid core — second call should not panic
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	c := core.New(core.WithOption("name", "test"))
	s.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

	core.AssertNotPanics(t, func() {
		_ = s.OnStartup(context.Background())
		_ = s.OnStartup(context.Background())
	})
}

// --- OnShutdown Good/Bad ---

func TestPrepOnShutdown_PrepSubsystem_OnShutdown_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: false}
	r := s.OnShutdown(context.Background())
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, s.frozen, "OnShutdown must set frozen=true")
}

func TestPrep_OnShutdown_Good_AlreadyFrozen(t *testing.T) {
	// Calling OnShutdown twice must be idempotent
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}), frozen: true}
	r := s.OnShutdown(context.Background())
	core.AssertTrue(t, r.OK)
	core.AssertTrue(t, s.frozen)
}

func TestPrep_OnShutdown_Good_NoError(t *testing.T) {
	s := &PrepSubsystem{ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{})}
	result := s.OnShutdown(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertTrue(t, s.frozen)
}

func TestPrepOnShutdown_PrepSubsystem_OnShutdown_Ugly(t *testing.T) {
	// OnShutdown must not panic even if s.core is nil
	s := &PrepSubsystem{ServiceRuntime: nil, frozen: false}
	core.AssertNotPanics(t, func() {
		_ = s.OnShutdown(context.Background())
	})
	core.AssertTrue(t, s.frozen)
}

func TestPrepOnShutdown_PrepSubsystem_OnShutdown_Bad(t *testing.T) {
	// OnShutdown without Core
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	core.AssertNotPanics(t, func() {
		r := s.OnShutdown(context.Background())
		core.AssertTrue(t, r.OK)
	})
	core.AssertTrue(t, s.frozen)
}

// --- Shutdown Bad/Ugly ---

func TestPrepShutdown_PrepSubsystem_Shutdown_Bad(t *testing.T) {
	// Shutdown always returns nil
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	err := s.Shutdown(context.Background())
	core.AssertNoError(t, err)
	core.AssertNil(t, err)
}

func TestPrepShutdown_PrepSubsystem_Shutdown_Ugly(t *testing.T) {
	// Shutdown on zero-value struct
	var s PrepSubsystem
	core.AssertNotPanics(t, func() {
		err := s.Shutdown(context.Background())
		core.AssertNoError(t, err)
	})
}

// --- EnvOr Bad/Ugly ---

func TestPrep_EnvOr_Bad(t *testing.T) {
	// Both env empty and fallback empty
	t.Setenv("TEST_ENVVAR_EMPTY_ALL", "")
	want := ""
	got := envOr("TEST_ENVVAR_EMPTY_ALL", "")
	core.AssertEqual(t, want, got)
}

func TestPrep_EnvOr_Ugly(t *testing.T) {
	// Env set to whitespace — whitespace is non-empty, so returned as-is
	t.Setenv("TEST_ENVVAR_WHITESPACE", "   ")
	want := "   "
	got := envOr("TEST_ENVVAR_WHITESPACE", "fallback")
	core.AssertEqual(t, want, got)
}

// --- DetectLanguage Bad/Ugly ---

func TestPrep_DetectLanguage_Bad(t *testing.T) {
	// Empty dir — defaults to go
	dir := t.TempDir()
	want := "go"
	got := detectLanguage(dir)
	core.AssertEqual(t, want, got)
}

func TestPrep_DetectLanguage_Ugly(t *testing.T) {
	// Dir with multiple project files (go.mod + package.json) — go wins (first match)
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "package.json"), "{}").OK)
	core.AssertEqual(t, "go", detectLanguage(dir), "go.mod checked first, so go wins")
}

// --- DetectBuildCmd Bad/Ugly ---

func TestPrep_DetectBuildCmd_Bad(t *testing.T) {
	// Unknown/non-existent path — defaults to go build
	path := "/nonexistent/path/that/does/not/exist"
	got := detectBuildCmd(path)
	core.AssertEqual(t, "go build ./...", got)
}

func TestPrep_DetectBuildCmd_Ugly(t *testing.T) {
	// Path that doesn't exist at all — defaults to go build
	core.AssertNotPanics(t, func() {
		result := detectBuildCmd("")
		core.AssertEqual(t, "go build ./...", result)
	})
}

// --- PrepareWorkspace ---

func TestPrepWorkspace_PrepSubsystem_PrepareWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

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
	core.AssertError(t, err)
}

func TestPrepWorkspace_PrepSubsystem_PrepareWorkspace_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Missing repo — should return error
	_, _, err := s.PrepareWorkspace(context.Background(), PrepInput{})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "repo is required")
}

func TestPrepWorkspace_PrepSubsystem_PrepareWorkspace_Ugly(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

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
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid repo name")
}

// --- BuildPrompt ---

func TestBuildPrompt_PrepSubsystem_BuildPrompt_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module test").OK)

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

	core.AssertNotEmpty(t, prompt)
	core.AssertContains(t, prompt, "TASK: Review code")
	core.AssertContains(t, prompt, "REPO: core/go-io on branch dev")
	core.AssertEqual(t, 0, memories)
	core.AssertEqual(t, 0, consumers)
}

func TestBuildPrompt_PrepSubsystem_BuildPrompt_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       t.TempDir(),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	// Empty inputs — should still return a prompt string without panicking
	prompt, memories, consumers := s.BuildPrompt(context.Background(), PrepInput{}, "", "")
	core.AssertNotEmpty(t, prompt)
	core.AssertContains(t, prompt, "TASK:")
	core.AssertContains(t, prompt, "CONSTRAINTS:")
	core.AssertEqual(t, 0, memories)
	core.AssertEqual(t, 0, consumers)
}

func TestBuildPrompt_PrepSubsystem_BuildPrompt_Ugly(t *testing.T) {
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

	core.AssertNotEmpty(t, prompt)
	core.AssertContains(t, prompt, "\u00e9nchantr\u00efx")
}

// --- collapseRepeatedRune / sanitisePlanSlug / trimRuneEdges Good ---

func TestPrep_CollapseRepeatedRune_Good(t *testing.T) {
	input := "hello---world"
	got := collapseRepeatedRune(input, '-')
	core.AssertEqual(t, "hello-world", got)
}

func TestPrep_SanitisePlanSlug_Good(t *testing.T) {
	input := "My Cool Plan"
	got := sanitisePlanSlug(input)
	core.AssertEqual(t, "my-cool-plan", got)
}

func TestPrep_TrimRuneEdges_Good(t *testing.T) {
	input := "--hello--"
	got := trimRuneEdges(input, '-')
	core.AssertEqual(t, "hello", got)
}

// --- DetectTestCmd Bad/Ugly ---

func TestPrep_DetectTestCmd_Bad(t *testing.T) {
	// Unknown path — defaults to go test
	path := "/nonexistent/path/that/does/not/exist"
	got := detectTestCmd(path)
	core.AssertEqual(t, "go test ./...", got)
}

func TestPrep_DetectTestCmd_Ugly(t *testing.T) {
	// Path that doesn't exist — defaults to go test
	core.AssertNotPanics(t, func() {
		result := detectTestCmd("")
		core.AssertEqual(t, "go test ./...", result)
	})
}

// --- getGitLog ---

func TestPrep_GetGitLog_Good(t *testing.T) {
	dir := t.TempDir()
	gitEnv := []string{"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@test.com"}
	run := func(args ...string) {
		t.Helper()
		r := testCore.Process().RunWithEnv(context.Background(), dir, gitEnv, args[0], args[1:]...)
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}
	run("git", "init", "-b", "main")
	run("git", "config", "user.name", "Test")
	run("git", "config", "user.email", "test@test.com")
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "README.md"), "# Test").OK)
	run("git", "add", "README.md")
	run("git", "commit", "-m", "initial commit")

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	log := s.getGitLog(dir)
	core.AssertNotEmpty(t, log)
	core.AssertContains(t, log, "initial commit")
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
	core.AssertEmpty(t, log)
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
	core.AssertEmpty(t, log)
}

// --- prepWorkspace Good ---

func TestPrep_PrepWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

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
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}
	core.RequireTrue(t, fs.EnsureDir(srcRepo).OK)
	run(srcRepo, "git", "init", "-b", "main")
	run(srcRepo, "git", "config", "user.name", "Test")
	run(srcRepo, "git", "config", "user.email", "test@test.com")
	core.RequireTrue(t, fs.Write(core.JoinPath(srcRepo, "README.md"), "# Test").OK)
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
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertNotEmpty(t, out.WorkspaceDir)
	core.AssertNotEmpty(t, out.Branch)
	core.AssertContains(t, out.Branch, "agent/")
	core.AssertNotEmpty(t, out.PromptVersion)

	promptIndexPath := core.JoinPath(WorkspaceMetaDir(out.WorkspaceDir), "prompt-version.json")
	core.RequireTrue(t, fs.Exists(promptIndexPath))
	promptIndexResult := fs.Read(promptIndexPath)
	core.RequireTrue(t, promptIndexResult.OK)

	var promptSnapshot PromptVersionSnapshot
	core.RequireTrue(t, core.JSONUnmarshalString(promptIndexResult.Value.(string), &promptSnapshot).OK)
	core.AssertEqual(t, out.PromptVersion, promptSnapshot.Hash)
	core.AssertContains(t, promptSnapshot.Content, "TASK: Fix tests")

	promptSnapshotPath := core.JoinPath(WorkspaceMetaDir(out.WorkspaceDir), "prompt-versions", core.Concat(out.PromptVersion, ".json"))
	core.RequireTrue(t, fs.Exists(promptSnapshotPath))

	todoPath := core.JoinPath(out.WorkspaceDir, "TODO.md")
	core.RequireTrue(t, fs.Exists(todoPath))
	todoResult := fs.Read(todoPath)
	core.RequireTrue(t, todoResult.OK)
	core.AssertNotEmpty(t, core.Trim(todoResult.Value.(string)))
}

func TestPrepWorkspace_PrepSubsystem_TestPrepWorkspace_Good(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

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
		if !r.OK {
			t.Fatalf("cmd %v failed: %v", args, r.Value)
		}
	}
	core.RequireTrue(t, fs.EnsureDir(srcRepo).OK)
	run(srcRepo, "git", "init", "-b", "main")
	run(srcRepo, "git", "config", "user.name", "Test")
	run(srcRepo, "git", "config", "user.email", "test@test.com")
	core.RequireTrue(t, fs.Write(core.JoinPath(srcRepo, "README.md"), "# Test").OK)
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
	core.RequireNoError(t, err)
	core.AssertTrue(t, out.Success)
	core.AssertNotEmpty(t, out.WorkspaceDir)

	todoPath := core.JoinPath(out.WorkspaceDir, "TODO.md")
	core.RequireTrue(t, fs.Exists(todoPath))
	todoResult := fs.Read(todoPath)
	core.RequireTrue(t, todoResult.OK)
	core.AssertNotEmpty(t, core.Trim(todoResult.Value.(string)))
}

func TestPrepWorkspace_PrepSubsystem_TestPrepWorkspace_Bad(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.TestPrepWorkspace(context.Background(), PrepInput{Repo: "."})
	core.AssertError(t, err)
}

func TestPrepWorkspace_PrepSubsystem_TestPrepWorkspace_Ugly(t *testing.T) {
	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	_, _, err := s.TestPrepWorkspace(context.Background(), PrepInput{Repo: ".."})
	core.AssertError(t, err)
}

func TestPrep_EnsureWorkspaceTaskFile_Bad(t *testing.T) {
	err := ensureWorkspaceTaskFile("")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "workspace dir is required")
}
