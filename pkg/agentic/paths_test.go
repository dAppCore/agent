// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core "dappco.re/go/core"
)

func TestPaths_CoreRoot_Good_EnvVar(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/test-core")
	assert.Equal(t, "/tmp/test-core", CoreRoot())
}

func TestPaths_CoreRoot_Good_Fallback(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home, _ := os.UserHomeDir()
	assert.Equal(t, home+"/Code/.core", CoreRoot())
}

func TestPaths_WorkspaceRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/test-core")
	assert.Equal(t, "/tmp/test-core/workspace", WorkspaceRoot())
}

func TestPaths_PlansRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/test-core")
	assert.Equal(t, "/tmp/test-core/plans", PlansRoot())
}

func TestPaths_AgentName_Good_EnvVar(t *testing.T) {
	t.Setenv("AGENT_NAME", "clotho")
	assert.Equal(t, "clotho", AgentName())
}

func TestPaths_AgentName_Good_Fallback(t *testing.T) {
	t.Setenv("AGENT_NAME", "")
	name := AgentName()
	assert.True(t, name == "cladius" || name == "charon", "expected cladius or charon, got %s", name)
}

func TestPaths_GitHubOrg_Good_EnvVar(t *testing.T) {
	t.Setenv("GITHUB_ORG", "myorg")
	assert.Equal(t, "myorg", GitHubOrg())
}

func TestPaths_GitHubOrg_Good_Fallback(t *testing.T) {
	t.Setenv("GITHUB_ORG", "")
	assert.Equal(t, "dAppCore", GitHubOrg())
}

func TestQueue_BaseAgent_Good(t *testing.T) {
	assert.Equal(t, "claude", baseAgent("claude:opus"))
	assert.Equal(t, "claude", baseAgent("claude:haiku"))
	assert.Equal(t, "gemini", baseAgent("gemini:flash"))
	assert.Equal(t, "codex", baseAgent("codex"))
}

func TestVerify_ExtractPRNumber_Good(t *testing.T) {
	assert.Equal(t, 123, extractPRNumber("https://forge.lthn.ai/core/go-io/pulls/123"))
	assert.Equal(t, 1, extractPRNumber("https://forge.lthn.ai/core/agent/pulls/1"))
}

func TestVerify_ExtractPRNumber_Bad_Empty(t *testing.T) {
	assert.Equal(t, 0, extractPRNumber(""))
	assert.Equal(t, 0, extractPRNumber("https://forge.lthn.ai/core/agent/pulls/"))
}

func TestAutoPr_Truncate_Good(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hel...", truncate("hello world", 3))
}

func TestReviewQueue_CountFindings_Good(t *testing.T) {
	assert.Equal(t, 0, countFindings("No findings"))
	assert.Equal(t, 2, countFindings("- Issue one\n- Issue two\nSummary"))
	assert.Equal(t, 1, countFindings("⚠ Warning here"))
}

func TestReviewQueue_ParseRetryAfter_Good(t *testing.T) {
	d := parseRetryAfter("please try after 4 minutes and 56 seconds")
	assert.InDelta(t, 296.0, d.Seconds(), 1.0)
}

func TestReviewQueue_ParseRetryAfter_Good_MinutesOnly(t *testing.T) {
	d := parseRetryAfter("try after 5 minutes")
	assert.InDelta(t, 300.0, d.Seconds(), 1.0)
}

func TestReviewQueue_ParseRetryAfter_Bad_NoMatch(t *testing.T) {
	d := parseRetryAfter("some random text")
	assert.InDelta(t, 300.0, d.Seconds(), 1.0) // defaults to 5 min
}

func TestRemote_ResolveHost_Good(t *testing.T) {
	assert.Equal(t, "10.69.69.165:9101", resolveHost("charon"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("cladius"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("local"))
}

func TestRemote_ResolveHost_Good_CustomPort(t *testing.T) {
	assert.Equal(t, "192.168.1.1:9101", resolveHost("192.168.1.1"))
	assert.Equal(t, "192.168.1.1:8080", resolveHost("192.168.1.1:8080"))
}

func TestMirror_ExtractJSONField_Good(t *testing.T) {
	json := `[{"url":"https://github.com/dAppCore/go-io/pull/1"}]`
	assert.Equal(t, "https://github.com/dAppCore/go-io/pull/1", extractJSONField(json, "url"))
}

func TestMirror_ExtractJSONField_Good_Object(t *testing.T) {
	json := `{"url":"https://github.com/dAppCore/go-io/pull/2"}`
	assert.Equal(t, "https://github.com/dAppCore/go-io/pull/2", extractJSONField(json, "url"))
}

func TestMirror_ExtractJSONField_Good_PrettyPrinted(t *testing.T) {
	json := "[\n  {\n    \"url\": \"https://github.com/dAppCore/go-io/pull/3\"\n  }\n]"
	assert.Equal(t, "https://github.com/dAppCore/go-io/pull/3", extractJSONField(json, "url"))
}

func TestMirror_ExtractJSONField_Bad_Missing(t *testing.T) {
	assert.Equal(t, "", extractJSONField(`{"name":"test"}`, "url"))
	assert.Equal(t, "", extractJSONField("", "url"))
}

func TestPlan_ValidPlanStatus_Good(t *testing.T) {
	assert.True(t, validPlanStatus("draft"))
	assert.True(t, validPlanStatus("in_progress"))
	assert.True(t, validPlanStatus("draft"))
}

func TestPlan_ValidPlanStatus_Bad(t *testing.T) {
	assert.False(t, validPlanStatus("invalid"))
	assert.False(t, validPlanStatus(""))
}

func TestPlan_GeneratePlanID_Good(t *testing.T) {
	id := generatePlanID("Fix the login bug in auth service")
	assert.True(t, len(id) > 0)
	assert.True(t, strings.Contains(id, "fix-the-login-bug"))
}

// --- DefaultBranch ---

func TestPaths_DefaultBranch_Good(t *testing.T) {
	dir := t.TempDir()

	// Init git repo with "main" branch
	cmd := exec.Command("git", "init", "-b", "main", dir)
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", dir, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	require.NoError(t, os.WriteFile(dir+"/README.md", []byte("# Test"), 0o644))
	cmd = exec.Command("git", "-C", dir, "add", ".")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "init")
	require.NoError(t, cmd.Run())

	branch := DefaultBranch(dir)
	assert.Equal(t, "main", branch)
}

func TestPaths_DefaultBranch_Bad(t *testing.T) {
	// Non-git directory — should return "main" (default)
	dir := t.TempDir()
	branch := DefaultBranch(dir)
	assert.Equal(t, "main", branch)
}

func TestPaths_DefaultBranch_Ugly(t *testing.T) {
	dir := t.TempDir()

	// Init git repo with "master" branch
	cmd := exec.Command("git", "init", "-b", "master", dir)
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "-C", dir, "config", "user.name", "Test")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", dir, "config", "user.email", "test@test.com")
	require.NoError(t, cmd.Run())

	require.NoError(t, os.WriteFile(dir+"/README.md", []byte("# Test"), 0o644))
	cmd = exec.Command("git", "-C", dir, "add", ".")
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "-C", dir, "commit", "-m", "init")
	require.NoError(t, cmd.Run())

	branch := DefaultBranch(dir)
	assert.Equal(t, "master", branch)
}

// --- LocalFs Bad/Ugly ---

func TestPaths_LocalFs_Bad_ReadNonExistent(t *testing.T) {
	f := LocalFs()
	r := f.Read("/tmp/nonexistent-path-" + strings.Repeat("x", 20) + "/file.txt")
	assert.False(t, r.OK, "reading a non-existent file should fail")
}

func TestPaths_LocalFs_Ugly_EmptyPath(t *testing.T) {
	f := LocalFs()
	assert.NotPanics(t, func() {
		f.Read("")
	})
}

// --- WorkspaceRoot Bad/Ugly ---

func TestPaths_WorkspaceRoot_Bad_EmptyEnv(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home, _ := os.UserHomeDir()
	// Should fall back to ~/Code/.core/workspace
	assert.Equal(t, home+"/Code/.core/workspace", WorkspaceRoot())
}

func TestPaths_WorkspaceRoot_Ugly_TrailingSlash(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/test-core/")
	// Verify it still constructs a valid path (JoinPath handles trailing slash)
	ws := WorkspaceRoot()
	assert.NotEmpty(t, ws)
	assert.Contains(t, ws, "workspace")
}

// --- CoreRoot Bad/Ugly ---

func TestPaths_CoreRoot_Bad_WhitespaceEnv(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "   ")
	// Non-empty string (whitespace) will be used as-is
	root := CoreRoot()
	assert.Equal(t, "   ", root)
}

func TestPaths_CoreRoot_Ugly_UnicodeEnv(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/\u00e9\u00e0\u00fc")
	assert.NotPanics(t, func() {
		root := CoreRoot()
		assert.Equal(t, "/tmp/\u00e9\u00e0\u00fc", root)
	})
}

// --- PlansRoot Bad/Ugly ---

func TestPaths_PlansRoot_Bad_EmptyEnv(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home, _ := os.UserHomeDir()
	assert.Equal(t, home+"/Code/.core/plans", PlansRoot())
}

func TestPaths_PlansRoot_Ugly_NestedPath(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/a/b/c/d/e/f")
	assert.Equal(t, "/a/b/c/d/e/f/plans", PlansRoot())
}

// --- AgentName Bad/Ugly ---

func TestPaths_AgentName_Bad_WhitespaceEnv(t *testing.T) {
	t.Setenv("AGENT_NAME", "   ")
	// Whitespace is non-empty, so returned as-is
	assert.Equal(t, "   ", AgentName())
}

func TestPaths_AgentName_Ugly_UnicodeEnv(t *testing.T) {
	t.Setenv("AGENT_NAME", "\u00e9nchantr\u00efx")
	assert.NotPanics(t, func() {
		name := AgentName()
		assert.Equal(t, "\u00e9nchantr\u00efx", name)
	})
}

// --- GitHubOrg Bad/Ugly ---

func TestPaths_GitHubOrg_Bad_WhitespaceEnv(t *testing.T) {
	t.Setenv("GITHUB_ORG", "   ")
	assert.Equal(t, "   ", GitHubOrg())
}

func TestPaths_GitHubOrg_Ugly_SpecialChars(t *testing.T) {
	t.Setenv("GITHUB_ORG", "org/with/slashes")
	assert.NotPanics(t, func() {
		org := GitHubOrg()
		assert.Equal(t, "org/with/slashes", org)
	})
}

// --- parseInt Bad/Ugly ---

func TestPaths_ParseInt_Bad_EmptyString(t *testing.T) {
	assert.Equal(t, 0, parseInt(""))
}

func TestPaths_ParseInt_Bad_NonNumeric(t *testing.T) {
	assert.Equal(t, 0, parseInt("abc"))
	assert.Equal(t, 0, parseInt("12.5"))
	assert.Equal(t, 0, parseInt("0xff"))
}

func TestPaths_ParseInt_Bad_WhitespaceOnly(t *testing.T) {
	assert.Equal(t, 0, parseInt("   "))
}

func TestPaths_ParseInt_Ugly_NegativeNumber(t *testing.T) {
	assert.Equal(t, -42, parseInt("-42"))
}

func TestPaths_ParseInt_Ugly_VeryLargeNumber(t *testing.T) {
	assert.Equal(t, 0, parseInt("99999999999999999999999"))
}

func TestPaths_ParseInt_Ugly_LeadingTrailingWhitespace(t *testing.T) {
	assert.Equal(t, 42, parseInt("  42  "))
}

// --- newFs Good/Bad/Ugly ---

func TestPaths_NewFs_Good(t *testing.T) {
	f := newFs("/tmp")
	assert.NotNil(t, f, "newFs should return a non-nil Fs")
	assert.IsType(t, &core.Fs{}, f)
}

// --- parseInt Good ---

func TestPaths_ParseInt_Good(t *testing.T) {
	assert.Equal(t, 42, parseInt("42"))
	assert.Equal(t, 0, parseInt("0"))
}

func TestPaths_NewFs_Bad_EmptyRoot(t *testing.T) {
	f := newFs("")
	assert.NotNil(t, f, "newFs with empty root should not return nil")
}

func TestPaths_NewFs_Ugly_UnicodeRoot(t *testing.T) {
	assert.NotPanics(t, func() {
		f := newFs("/tmp/\u00e9\u00e0\u00fc/\u00f1o\u00f0\u00e9s")
		assert.NotNil(t, f)
	})
}

func TestPaths_NewFs_Ugly_VerifyIsFs(t *testing.T) {
	f := newFs("/tmp")
	assert.IsType(t, &core.Fs{}, f)
}
