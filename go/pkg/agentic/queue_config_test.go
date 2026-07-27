// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestLoadAgentsConfig_Good_LoadsRepoCoreConfig(t *testing.T) {
	codeRoot := t.TempDir()
	// CoreRoot()/agents.yaml absent → loader must fall through to the repo's
	// core/agent/.core/agents.yaml (the path the stale config/ entry missed).
	t.Setenv("CORE_WORKSPACE", t.TempDir())

	cfgDir := core.JoinPath(codeRoot, "core", "agent", ".core")
	core.RequireTrue(t, fs.EnsureDir(cfgDir).OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(cfgDir, "agents.yaml"),
		"version: 1\nconcurrency:\n  opencode:\n    total: 3\n    opencode-go/deepseek-v4-pro: 1\n").OK)

	s := &PrepSubsystem{codePath: codeRoot}
	config := s.loadAgentsConfig()

	limit := config.Concurrency["opencode"]
	core.AssertEqual(t, 3, limit.Total)
	core.AssertEqual(t, 1, limit.Models["opencode-go/deepseek-v4-pro"])
}

func TestLoadAgentsConfig_Bad_MissingConfigFallsBackToDefault(t *testing.T) {
	// No config at any searched path → hardcoded default (claude + gemini only,
	// no opencode entry → opencode would be unlimited).
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	s := &PrepSubsystem{codePath: t.TempDir()}

	config := s.loadAgentsConfig()

	_, hasOpencode := config.Concurrency["opencode"]
	core.AssertFalse(t, hasOpencode)
	core.AssertEqual(t, 1, config.Concurrency["claude"].Total)
}
