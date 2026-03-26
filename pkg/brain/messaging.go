// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterMessagingTools adds direct agent messaging tools to an MCP server.
//
//	sub := brain.NewDirect()
//	sub.RegisterMessagingTools(server)
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

// SendInput sends a direct message to another agent.
//
//	brain.SendInput{To: "charon", Subject: "status update", Content: "deploy complete"}
type SendInput struct {
	To      string `json:"to"`
	Content string `json:"content"`
	Subject string `json:"subject,omitempty"`
}

// SendOutput reports the created direct message.
//
//	brain.SendOutput{Success: true, ID: 42, To: "charon"}
type SendOutput struct {
	Success bool   `json:"success"`
	ID      int    `json:"id"`
	To      string `json:"to"`
}

// InboxInput selects which agent inbox to read.
//
//	brain.InboxInput{Agent: "cladius"}
type InboxInput struct {
	Agent string `json:"agent,omitempty"`
}

// MessageItem is one inbox or conversation message.
//
//	brain.MessageItem{ID: 7, From: "cladius", To: "charon", Content: "all green"}
type MessageItem struct {
	ID        int    `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Subject   string `json:"subject,omitempty"`
	Content   string `json:"content"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
}

// InboxOutput returns the latest direct messages for an agent.
//
//	brain.InboxOutput{Success: true, Messages: []brain.MessageItem{{ID: 1, From: "charon", To: "cladius"}}}
type InboxOutput struct {
	Success  bool          `json:"success"`
	Messages []MessageItem `json:"messages"`
}

// ConversationInput selects the agent thread to load.
//
//	brain.ConversationInput{Agent: "charon"}
type ConversationInput struct {
	Agent string `json:"agent"`
}

// ConversationOutput returns a direct message thread with another agent.
//
//	brain.ConversationOutput{Success: true, Messages: []brain.MessageItem{{ID: 10, From: "cladius", To: "charon"}}}
type ConversationOutput struct {
	Success  bool          `json:"success"`
	Messages []MessageItem `json:"messages"`
}

// Handlers

func (s *DirectSubsystem) sendMessage(ctx context.Context, _ *mcp.CallToolRequest, input SendInput) (*mcp.CallToolResult, SendOutput, error) {
	if input.To == "" || input.Content == "" {
		return nil, SendOutput{}, core.E("brain.sendMessage", "to and content are required", nil)
	}

	result, err := s.apiCall(ctx, "POST", "/v1/messages/send", map[string]any{
		"to":      input.To,
		"from":    agentic.AgentName(),
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
		agent = agentic.AgentName()
	}
	// Agent names are validated identifiers — no URL escaping needed.
	result, err := s.apiCall(ctx, "GET", core.Concat("/v1/messages/inbox?agent=", agent), nil)
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
		return nil, ConversationOutput{}, core.E("brain.conversation", "agent is required", nil)
	}

	result, err := s.apiCall(ctx, "GET", core.Concat("/v1/messages/conversation/", input.Agent, "?me=", agentic.AgentName()), nil)
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
			From:      fieldString(mm, "from"),
			To:        fieldString(mm, "to"),
			Subject:   fieldString(mm, "subject"),
			Content:   fieldString(mm, "content"),
			Read:      mm["read"] == true,
			CreatedAt: fieldString(mm, "created_at"),
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
