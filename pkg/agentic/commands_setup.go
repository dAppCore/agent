// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/setup"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *PrepSubsystem) registerSetupCommands() {
	c := s.Core()
	c.Command("setup", core.Command{Description: "Scaffold a workspace with .core config files and optional templates", Action: s.cmdSetup})
	c.Command("agentic:setup", core.Command{Description: "Scaffold a workspace with .core config files and optional templates", Action: s.cmdSetup})
}

func (s *PrepSubsystem) cmdSetup(options core.Options) core.Result {
	return s.handleSetup(context.Background(), options)
}

// result := c.Action("agentic.setup").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "path", Value: "."},
//	core.Option{Key: "template", Value: "auto"},
//
// ))
func (s *PrepSubsystem) handleSetup(_ context.Context, options core.Options) core.Result {
	serviceResult := s.Core().Service("setup")
	if !serviceResult.OK {
		if serviceResult.Value != nil {
			return core.Result{Value: core.E("agentic.setup", "setup service is required", nil), OK: false}
		}
		return core.Result{Value: core.E("agentic.setup", "setup service is required", nil), OK: false}
	}

	service, ok := serviceResult.Value.(*setup.Service)
	if !ok || service == nil {
		return core.Result{Value: core.E("agentic.setup", "setup service is required", nil), OK: false}
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

func (s *PrepSubsystem) registerSetupTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_setup",
		Description: "Scaffold a workspace with .core config files and optional templates.",
	}, s.setupTool)
}

type SetupInput struct {
	Path     string `json:"path,omitempty"`
	DryRun   bool   `json:"dry_run,omitempty"`
	Force    bool   `json:"force,omitempty"`
	Template string `json:"template,omitempty"`
}

type SetupOutput struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
}

func (s *PrepSubsystem) setupTool(ctx context.Context, _ *mcp.CallToolRequest, input SetupInput) (*mcp.CallToolResult, SetupOutput, error) {
	result := s.handleSetup(ctx, core.NewOptions(
		core.Option{Key: "path", Value: input.Path},
		core.Option{Key: "dry_run", Value: input.DryRun},
		core.Option{Key: "force", Value: input.Force},
		core.Option{Key: "template", Value: input.Template},
	))
	if !result.OK {
		return nil, SetupOutput{}, resultErrorValue("agentic.setup", result)
	}

	path, ok := result.Value.(string)
	if !ok {
		return nil, SetupOutput{}, core.E("agentic.setup", "invalid setup output", nil)
	}

	return nil, SetupOutput{
		Success: true,
		Path:    path,
	}, nil
}
