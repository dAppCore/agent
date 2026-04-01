// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version := agentic.PromptVersionOutput{Success: true, Snapshot: agentic.PromptVersionSnapshot{Hash: "..." }}
type PromptVersionOutput struct {
	Success   bool                  `json:"success"`
	Workspace string                `json:"workspace"`
	Snapshot  PromptVersionSnapshot `json:"snapshot"`
}

// result := c.Action("agentic.prompt.version").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "/srv/.core/workspace/core/go-io/task-42"},
//
// ))
func (s *PrepSubsystem) handlePromptVersion(ctx context.Context, options core.Options) core.Result {
	workspace := optionStringValue(options, "workspace", "_arg")
	_, output, err := s.promptVersion(ctx, nil, workspace)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) promptVersion(_ context.Context, _ *mcp.CallToolRequest, workspace string) (*mcp.CallToolResult, PromptVersionOutput, error) {
	workspaceDir := s.resolveWorkspaceDir(workspace)
	if workspaceDir == "" {
		return nil, PromptVersionOutput{}, core.E("promptVersion", "workspace is required", nil)
	}

	snapshot, err := readPromptSnapshot(workspaceDir)
	if err != nil {
		return nil, PromptVersionOutput{}, err
	}

	return nil, PromptVersionOutput{
		Success:   true,
		Workspace: workspace,
		Snapshot:  snapshot,
	}, nil
}
