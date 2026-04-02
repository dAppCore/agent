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

// prompt := agentic.Prompt{Name: "coding", Category: "workspace"}
type Prompt struct {
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	Category     string         `json:"category,omitempty"`
	Description  string         `json:"description,omitempty"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	UserTemplate string         `json:"user_template,omitempty"`
	Variables    map[string]any `json:"variables,omitempty"`
	Model        string         `json:"model,omitempty"`
	ModelConfig  map[string]any `json:"model_config,omitempty"`
	IsActive     bool           `json:"is_active,omitempty"`
}

// version := agentic.PromptVersion{Name: "coding", Version: 3}
type PromptVersion struct {
	ID           int            `json:"id"`
	PromptID     int            `json:"prompt_id"`
	Version      int            `json:"version"`
	SystemPrompt string         `json:"system_prompt,omitempty"`
	UserTemplate string         `json:"user_template,omitempty"`
	Variables    map[string]any `json:"variables,omitempty"`
	CreatedBy    string         `json:"created_by,omitempty"`
	CreatedAt    string         `json:"created_at,omitempty"`
}

// input := agentic.PromptVersionInput{Workspace: "/srv/.core/workspace/core/go-io/task-42"}
type PromptVersionInput struct {
	Workspace string `json:"workspace"`
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

func (s *PrepSubsystem) registerPromptTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prompt_version",
		Description: "Read the current prompt snapshot for a workspace.",
	}, s.promptVersionTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_prompt_version",
		Description: "Read the current prompt snapshot for a workspace.",
	}, s.promptVersionTool)
}

func (s *PrepSubsystem) promptVersionTool(ctx context.Context, _ *mcp.CallToolRequest, input PromptVersionInput) (*mcp.CallToolResult, PromptVersionOutput, error) {
	if core.Trim(input.Workspace) == "" {
		return nil, PromptVersionOutput{}, core.E("promptVersion", "workspace is required", nil)
	}

	return s.promptVersion(ctx, nil, input.Workspace)
}
