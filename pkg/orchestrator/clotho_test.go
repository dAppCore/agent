package orchestrator

import (
	"context"
	"testing"

	"forge.lthn.ai/core/agent/pkg/jobrunner"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSpinner() *Spinner {
	return NewSpinner(
		ClothoConfig{
			Strategy:            "clotho-verified",
			ValidationThreshold: 0.85,
		},
		map[string]AgentConfig{
			"claude-agent": {
				Host:     "claude@10.0.0.1",
				Model:    "opus",
				Runner:   "claude",
				Active:   true,
				DualRun:  false,
				ForgejoUser: "claude-forge",
			},
			"gemini-agent": {
				Host:        "localhost",
				Model:       "gemini-2.0-flash",
				VerifyModel: "gemini-1.5-pro",
				Runner:      "gemini",
				Active:      true,
				DualRun:     true,
				ForgejoUser: "gemini-forge",
			},
		},
	)
}

func TestNewSpinner_Good(t *testing.T) {
	spinner := newTestSpinner()
	assert.NotNil(t, spinner)
	assert.Equal(t, "clotho-verified", spinner.Config.Strategy)
	assert.Len(t, spinner.Agents, 2)
}

func TestDeterminePlan_Good_Standard(t *testing.T) {
	spinner := newTestSpinner()

	signal := &jobrunner.PipelineSignal{
		RepoOwner: "host-uk",
		RepoName:  "core-php",
	}

	mode := spinner.DeterminePlan(signal, "claude-agent")
	assert.Equal(t, ModeStandard, mode)
}

func TestDeterminePlan_Good_DualRunByAgent(t *testing.T) {
	spinner := newTestSpinner()

	signal := &jobrunner.PipelineSignal{
		RepoOwner: "host-uk",
		RepoName:  "some-repo",
	}

	mode := spinner.DeterminePlan(signal, "gemini-agent")
	assert.Equal(t, ModeDual, mode)
}

func TestDeterminePlan_Good_DualRunByCriticalRepo(t *testing.T) {
	spinner := newTestSpinner()

	tests := []struct {
		name     string
		repoName string
		expected RunMode
	}{
		{name: "core repo", repoName: "core", expected: ModeDual},
		{name: "security repo", repoName: "auth-security", expected: ModeDual},
		{name: "security-audit", repoName: "security-audit", expected: ModeDual},
		{name: "regular repo", repoName: "docs", expected: ModeStandard},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := &jobrunner.PipelineSignal{
				RepoOwner: "host-uk",
				RepoName:  tt.repoName,
			}
			mode := spinner.DeterminePlan(signal, "claude-agent")
			assert.Equal(t, tt.expected, mode)
		})
	}
}

func TestDeterminePlan_Good_NonVerifiedStrategy(t *testing.T) {
	spinner := NewSpinner(
		ClothoConfig{Strategy: "direct"},
		map[string]AgentConfig{
			"agent": {Host: "localhost", DualRun: true, Active: true},
		},
	)

	signal := &jobrunner.PipelineSignal{RepoName: "core"}
	mode := spinner.DeterminePlan(signal, "agent")
	assert.Equal(t, ModeStandard, mode, "non-verified strategy should always return standard")
}

func TestDeterminePlan_Good_UnknownAgent(t *testing.T) {
	spinner := newTestSpinner()

	signal := &jobrunner.PipelineSignal{RepoName: "some-repo"}
	mode := spinner.DeterminePlan(signal, "nonexistent-agent")
	assert.Equal(t, ModeStandard, mode, "unknown agent should return standard")
}

func TestGetVerifierModel_Good(t *testing.T) {
	spinner := newTestSpinner()

	model := spinner.GetVerifierModel("gemini-agent")
	assert.Equal(t, "gemini-1.5-pro", model)
}

func TestGetVerifierModel_Good_Default(t *testing.T) {
	spinner := newTestSpinner()

	// claude-agent has no VerifyModel set.
	model := spinner.GetVerifierModel("claude-agent")
	assert.Equal(t, "gemini-1.5-pro", model, "should fall back to default")
}

func TestGetVerifierModel_Good_UnknownAgent(t *testing.T) {
	spinner := newTestSpinner()

	model := spinner.GetVerifierModel("unknown")
	assert.Equal(t, "gemini-1.5-pro", model, "should fall back to default")
}

func TestFindByForgejoUser_Good_DirectMatch(t *testing.T) {
	spinner := newTestSpinner()

	// Direct match on config key.
	name, agent, found := spinner.FindByForgejoUser("claude-agent")
	assert.True(t, found)
	assert.Equal(t, "claude-agent", name)
	assert.Equal(t, "opus", agent.Model)
}

func TestFindByForgejoUser_Good_ByField(t *testing.T) {
	spinner := newTestSpinner()

	// Match by ForgejoUser field.
	name, agent, found := spinner.FindByForgejoUser("claude-forge")
	assert.True(t, found)
	assert.Equal(t, "claude-agent", name)
	assert.Equal(t, "opus", agent.Model)
}

func TestFindByForgejoUser_Bad_NotFound(t *testing.T) {
	spinner := newTestSpinner()

	_, _, found := spinner.FindByForgejoUser("nonexistent")
	assert.False(t, found)
}

func TestFindByForgejoUser_Bad_Empty(t *testing.T) {
	spinner := newTestSpinner()

	_, _, found := spinner.FindByForgejoUser("")
	assert.False(t, found)
}

func TestWeave_Good_Matching(t *testing.T) {
	spinner := newTestSpinner()

	converge, err := spinner.Weave(context.Background(), []byte("output"), []byte("output"))
	require.NoError(t, err)
	assert.True(t, converge)
}

func TestWeave_Good_Diverging(t *testing.T) {
	spinner := newTestSpinner()

	converge, err := spinner.Weave(context.Background(), []byte("primary"), []byte("different"))
	require.NoError(t, err)
	assert.False(t, converge)
}

func TestRunModeConstants(t *testing.T) {
	assert.Equal(t, RunMode("standard"), ModeStandard)
	assert.Equal(t, RunMode("dual"), ModeDual)
}
