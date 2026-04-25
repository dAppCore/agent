// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsCore_RegisterCoreCommands_Good(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCoreCommands()

	for _, spec := range coreCommandSpecs {
		assert.Contains(t, c.Commands(), spec.Path)

		result := c.Command(spec.Path)
		require.True(t, result.OK, spec.Path)

		command, ok := result.Value.(*core.Command)
		require.True(t, ok, spec.Path)
		assert.Equal(t, spec.Description, command.Description)
		assert.NotEmpty(t, command.Description)
	}
}

func TestCommandsCore_CliHelp_Good_ListsAllSubcommands(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCoreCommands()

	var result core.Result
	output := captureStdout(t, func() {
		result = c.Cli().Run("core", "--help")
	})

	require.True(t, result.OK)
	assert.Contains(t, output, "usage: core [pipeline] [--help]")

	for _, spec := range coreCommandSpecs {
		if spec.Path == "core" {
			continue
		}
		assert.Contains(t, output, spec.Usage)
	}
}

func TestCommandsCore_CliRoute_Bad_AuditPlaceholder(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCoreCommands()

	var result core.Result
	output := captureStdout(t, func() {
		result = c.Cli().Run("core", "pipeline", "audit", "go-io")
	})

	assert.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "core pipeline audit is not yet implemented")
	assert.Contains(t, output, "usage: core pipeline audit <repo> [--help]")
	assert.Contains(t, output, "about: Stage 1: audit issues into implementation work")
	assert.Contains(t, output, "status: not yet implemented")
	assert.Contains(t, output, "docs/flow/RFC.flow-audit-issues.md")
}
