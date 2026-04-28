// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestCommandsCore_RegisterCoreCommands_Good(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCoreCommands()

	for _, spec := range coreCommandSpecs {
		core.AssertContains(t, c.Commands(), spec.Path)

		result := c.Command(spec.Path)
		core.RequireTrue(t, result.OK, spec.Path)

		command, ok := result.Value.(*core.Command)
		core.RequireTrue(t, ok, spec.Path)
		core.AssertEqual(t, spec.Description, command.Description)
		core.AssertNotEmpty(t, command.Description)
	}
}

func TestCommandsCore_CliHelp_Good_ListsAllSubcommands(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCoreCommands()

	var result core.Result
	output := captureStdout(t, func() {
		result = c.Cli().Run("core", "--help")
	})

	core.RequireTrue(t, result.OK)
	core.AssertContains(t, output, "usage: core [pipeline] [--help]")

	for _, spec := range coreCommandSpecs {
		if spec.Path == "core" {
			continue
		}
		core.AssertContains(t, output, spec.Usage)
	}
}

func TestCommandsCore_CliRoute_Bad_AuditPlaceholder(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerCoreCommands()

	var result core.Result
	output := captureStdout(t, func() {
		result = c.Cli().Run("core", "pipeline", "audit", "go-io")
	})

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "core pipeline audit is not yet implemented")
	core.AssertContains(t, output, "usage: core pipeline audit <repo> [--help]")
	core.AssertContains(t, output, "about: Stage 1: audit issues into implementation work")
	core.AssertContains(t, output, "status: not yet implemented")
	core.AssertContains(t, output, "docs/flow/RFC.flow-audit-issues.md")
}
