// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Subsystem) Shutdown(ctx context.Context) error {
	return Shutdown(s, ctx)
}

func (s *Subsystem) brainRemember(ctx context.Context, request *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	return brainRemember(s, ctx, request, input)
}

func (s *Subsystem) brainRecall(ctx context.Context, request *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	return brainRecall(s, ctx, request, input)
}

func (s *Subsystem) brainForget(ctx context.Context, request *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	return brainForget(s, ctx, request, input)
}

func (s *Subsystem) brainList(ctx context.Context, request *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ListOutput, error) {
	return brainList(s, ctx, request, input)
}

func (s *DirectSubsystem) remember(ctx context.Context, request *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	return remember(s, ctx, request, input)
}

func (s *DirectSubsystem) recall(ctx context.Context, request *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	return recall(s, ctx, request, input)
}

func (s *DirectSubsystem) forget(ctx context.Context, request *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	return forget(s, ctx, request, input)
}

func (s *DirectSubsystem) list(ctx context.Context, request *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ListOutput, error) {
	return list(s, ctx, request, input)
}

func (s *DirectSubsystem) sendMessage(ctx context.Context, request *mcp.CallToolRequest, input SendInput) (*mcp.CallToolResult, SendOutput, error) {
	return sendMessage(s, ctx, request, input)
}

func (s *DirectSubsystem) inbox(ctx context.Context, request *mcp.CallToolRequest, input InboxInput) (*mcp.CallToolResult, InboxOutput, error) {
	return inbox(s, ctx, request, input)
}

func (s *DirectSubsystem) conversation(ctx context.Context, request *mcp.CallToolRequest, input ConversationInput) (*mcp.CallToolResult, ConversationOutput, error) {
	return conversation(s, ctx, request, input)
}
