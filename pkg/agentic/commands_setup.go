// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"dappco.re/go/agent/pkg/setup"
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerSetupCommands() {
	c := s.Core()
	c.Command("setup", core.Command{Description: "Scaffold a workspace with .core config files and optional templates", Action: s.cmdSetup})
	c.Command("agentic:setup", core.Command{Description: "Scaffold a workspace with .core config files and optional templates", Action: s.cmdSetup})
}

func (s *PrepSubsystem) cmdSetup(options core.Options) core.Result {
	serviceResult := s.Core().Service("setup")
	if !serviceResult.OK {
		if serviceResult.Value != nil {
			return core.Result{Value: serviceResult.Value, OK: false}
		}
		return core.Result{Value: core.E("agentic.cmdSetup", "setup service is required", nil), OK: false}
	}

	service, ok := serviceResult.Value.(*setup.Service)
	if !ok || service == nil {
		return core.Result{Value: core.E("agentic.cmdSetup", "setup service is required", nil), OK: false}
	}

	result := service.Run(setup.Options{
		Path:     optionStringValue(options, "path", "_arg"),
		DryRun:   optionBoolValue(options, "dry_run", "dry-run"),
		Force:    optionBoolValue(options, "force"),
		Template: optionStringValue(options, "template", "template_slug", "template-slug", "slug"),
	})
	if !result.OK {
		return result
	}

	return result
}
