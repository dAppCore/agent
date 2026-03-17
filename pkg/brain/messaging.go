// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"fmt"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterMessagingTools adds agent messaging tools to the MCP server.
func (s *DirectSubsystem) RegisterMessagingTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agent_send",
		Description: "Send a message to another agent. Direct, chronological, not semantic.",
	}, s.sendMessage)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agent_inbox",
		Description: "Check your inbox — latest messages sent to you.",
	}, s.inbox)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agent_conversation",
		Description: "View conversation thread with a specific agent.",
	}, s.conversation)
}

// Input/Output types

type SendInput struct {
	To      string `json:"to"`
	Content string `json:"content"`
	Subject string `json:"subject,omitempty"`
}

type SendOutput struct {
	Success bool   `json:"success"`
	ID      int    `json:"id"`
	To      string `json:"to"`
}

type InboxInput struct {
	Agent string `json:"agent,omitempty"`
}

type MessageItem struct {
	ID        int    `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject,omitempty"`
	Content   string `json:"content"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
}

type InboxOutput struct {
	Success  bool          `json:"success"`
	Messages []MessageItem `json:"messages"`
}

type ConversationInput struct {
	Agent string `json:"agent"`
}

type ConversationOutput struct {
	Success  bool          `json:"success"`
	Messages []MessageItem `json:"messages"`
}

// Handlers

func (s *DirectSubsystem) sendMessage(ctx context.Context, _ *mcp.CallToolRequest, input SendInput) (*mcp.CallToolResult, SendOutput, error) {
	if input.To == "" || input.Content == "" {
		return nil, SendOutput{}, coreerr.E("brain.sendMessage", "to and content are required", nil)
	}

	result, err := s.apiCall(ctx, "POST", "/v1/messages/send", map[string]any{
		"to":      input.To,
		"from":    agentName(),
		"content": input.Content,
		"subject": input.Subject,
	})
	if err != nil {
		return nil, SendOutput{}, err
	}

	data, _ := result["data"].(map[string]any)
	id, _ := data["id"].(float64)

	return nil, SendOutput{
		Success: true,
		ID:      int(id),
		To:      input.To,
	}, nil
}

func (s *DirectSubsystem) inbox(ctx context.Context, _ *mcp.CallToolRequest, input InboxInput) (*mcp.CallToolResult, InboxOutput, error) {
	agent := input.Agent
	if agent == "" {
		agent = agentName()
	}
	result, err := s.apiCall(ctx, "GET", "/v1/messages/inbox?agent="+agent, nil)
	if err != nil {
		return nil, InboxOutput{}, err
	}

	return nil, InboxOutput{
		Success:  true,
		Messages: parseMessages(result),
	}, nil
}

func (s *DirectSubsystem) conversation(ctx context.Context, _ *mcp.CallToolRequest, input ConversationInput) (*mcp.CallToolResult, ConversationOutput, error) {
	if input.Agent == "" {
		return nil, ConversationOutput{}, coreerr.E("brain.conversation", "agent is required", nil)
	}

	result, err := s.apiCall(ctx, "GET", "/v1/messages/conversation/"+input.Agent+"?me="+agentName(), nil)
	if err != nil {
		return nil, ConversationOutput{}, err
	}

	return nil, ConversationOutput{
		Success:  true,
		Messages: parseMessages(result),
	}, nil
}

func parseMessages(result map[string]any) []MessageItem {
	var messages []MessageItem
	data, _ := result["data"].([]any)
	for _, m := range data {
		mm, _ := m.(map[string]any)
		messages = append(messages, MessageItem{
			ID:        toInt(mm["id"]),
			From:      fmt.Sprintf("%v", mm["from"]),
			To:        fmt.Sprintf("%v", mm["to"]),
			Subject:   fmt.Sprintf("%v", mm["subject"]),
			Content:   fmt.Sprintf("%v", mm["content"]),
			Read:      mm["read"] == true,
			CreatedAt: fmt.Sprintf("%v", mm["created_at"]),
		})
	}
	return messages
}

func toInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}
