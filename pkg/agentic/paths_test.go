// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoreRoot_Good_EnvVar(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/test-core")
	assert.Equal(t, "/tmp/test-core", CoreRoot())
}

func TestCoreRoot_Good_Fallback(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "")
	home, _ := os.UserHomeDir()
	assert.Equal(t, home+"/Code/.core", CoreRoot())
}

func TestWorkspaceRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/test-core")
	assert.Equal(t, "/tmp/test-core/workspace", WorkspaceRoot())
}

func TestPlansRoot_Good(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", "/tmp/test-core")
	assert.Equal(t, "/tmp/test-core/plans", PlansRoot())
}

func TestAgentName_Good_EnvVar(t *testing.T) {
	t.Setenv("AGENT_NAME", "clotho")
	assert.Equal(t, "clotho", AgentName())
}

func TestAgentName_Good_Fallback(t *testing.T) {
	t.Setenv("AGENT_NAME", "")
	name := AgentName()
	assert.True(t, name == "cladius" || name == "charon", "expected cladius or charon, got %s", name)
}

func TestGitHubOrg_Good_EnvVar(t *testing.T) {
	t.Setenv("GITHUB_ORG", "myorg")
	assert.Equal(t, "myorg", GitHubOrg())
}

func TestGitHubOrg_Good_Fallback(t *testing.T) {
	t.Setenv("GITHUB_ORG", "")
	assert.Equal(t, "dAppCore", GitHubOrg())
}

func TestBaseAgent_Good(t *testing.T) {
	assert.Equal(t, "claude", baseAgent("claude:opus"))
	assert.Equal(t, "claude", baseAgent("claude:haiku"))
	assert.Equal(t, "gemini", baseAgent("gemini:flash"))
	assert.Equal(t, "codex", baseAgent("codex"))
}

func TestExtractPRNumber_Good(t *testing.T) {
	assert.Equal(t, 123, extractPRNumber("https://forge.lthn.ai/core/go-io/pulls/123"))
	assert.Equal(t, 1, extractPRNumber("https://forge.lthn.ai/core/agent/pulls/1"))
}

func TestExtractPRNumber_Bad_Empty(t *testing.T) {
	assert.Equal(t, 0, extractPRNumber(""))
	assert.Equal(t, 0, extractPRNumber("https://forge.lthn.ai/core/agent/pulls/"))
}

func TestTruncate_Good(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hel...", truncate("hello world", 3))
}

func TestCountFindings_Good(t *testing.T) {
	assert.Equal(t, 0, countFindings("No findings"))
	assert.Equal(t, 2, countFindings("- Issue one\n- Issue two\nSummary"))
	assert.Equal(t, 1, countFindings("⚠ Warning here"))
}

func TestParseRetryAfter_Good(t *testing.T) {
	d := parseRetryAfter("please try after 4 minutes and 56 seconds")
	assert.InDelta(t, 296.0, d.Seconds(), 1.0)
}

func TestParseRetryAfter_Good_MinutesOnly(t *testing.T) {
	d := parseRetryAfter("try after 5 minutes")
	assert.InDelta(t, 300.0, d.Seconds(), 1.0)
}

func TestParseRetryAfter_Bad_NoMatch(t *testing.T) {
	d := parseRetryAfter("some random text")
	assert.InDelta(t, 300.0, d.Seconds(), 1.0) // defaults to 5 min
}

func TestResolveHost_Good(t *testing.T) {
	assert.Equal(t, "10.69.69.165:9101", resolveHost("charon"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("cladius"))
	assert.Equal(t, "127.0.0.1:9101", resolveHost("local"))
}

func TestResolveHost_Good_CustomPort(t *testing.T) {
	assert.Equal(t, "192.168.1.1:9101", resolveHost("192.168.1.1"))
	assert.Equal(t, "192.168.1.1:8080", resolveHost("192.168.1.1:8080"))
}

func TestExtractJSONField_Good(t *testing.T) {
	json := `[{"url":"https://github.com/dAppCore/go-io/pull/1"}]`
	assert.Equal(t, "https://github.com/dAppCore/go-io/pull/1", extractJSONField(json, "url"))
}

func TestExtractJSONField_Bad_Missing(t *testing.T) {
	assert.Equal(t, "", extractJSONField(`{"name":"test"}`, "url"))
	assert.Equal(t, "", extractJSONField("", "url"))
}

func TestValidPlanStatus_Good(t *testing.T) {
	assert.True(t, validPlanStatus("draft"))
	assert.True(t, validPlanStatus("in_progress"))
	assert.True(t, validPlanStatus("draft"))
}

func TestValidPlanStatus_Bad(t *testing.T) {
	assert.False(t, validPlanStatus("invalid"))
	assert.False(t, validPlanStatus(""))
}

func TestGeneratePlanID_Good(t *testing.T) {
	id := generatePlanID("Fix the login bug in auth service")
	assert.True(t, len(id) > 0)
	assert.True(t, strings.Contains(id, "fix-the-login-bug"))
}
