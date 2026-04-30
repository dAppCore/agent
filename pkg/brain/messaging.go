// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// subsystem := brain.NewDirect()
// subsystem.RegisterMessagingTools(svc)
func (s *DirectSubsystem) RegisterMessagingTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "brain", &mcp.Tool{
		Name:        "agent_send",
		Description: "Send a message to another agent. Direct, chronological, not semantic.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input SendInput) (*mcp.CallToolResult, SendOutput, error) {
		return sendMessage(s, ctx, request, input)
	})

	coremcp.AddToolRecorded(svc, svc.Server(), "brain", &mcp.Tool{
		Name:        "agent_inbox",
		Description: "Check your inbox — latest messages sent to you.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input InboxInput) (*mcp.CallToolResult, InboxOutput, error) {
		return inbox(s, ctx, request, input)
	})

	coremcp.AddToolRecorded(svc, svc.Server(), "brain", &mcp.Tool{
		Name:        "agent_conversation",
		Description: "View conversation thread with a specific agent.",
	}, func(ctx context.Context, request *mcp.CallToolRequest, input ConversationInput) (*mcp.CallToolResult, ConversationOutput, error) {
		return conversation(s, ctx, request, input)
	})
}

// brain.SendInput{To: "charon", Subject: "status update", Content: "deploy complete"}
type SendInput struct {
	To      string `json:"to"`
	Content string `json:"content"`
	Subject string `json:"subject,omitempty"`
}

// brain.SendOutput{Success: true, ID: 42, To: "charon"}
type SendOutput struct {
	Success bool   `json:"success"`
	ID      int    `json:"id"`
	To      string `json:"to"`
}

// brain.InboxInput{Agent: "cladius"}
type InboxInput struct {
	Agent string `json:"agent,omitempty"`
}

// brain.MessageItem{ID: 7, WorkspaceID: 1, FromAgent: "cladius", ToAgent: "charon", Content: "all green"}
type MessageItem struct {
	ID          int    `json:"id"`
	WorkspaceID int    `json:"workspace_id,omitempty"`
	From        string `json:"from,omitempty"`
	FromAgent   string `json:"from_agent,omitempty"`
	To          string `json:"to,omitempty"`
	ToAgent     string `json:"to_agent,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Content     string `json:"content"`
	Read        bool   `json:"read"`
	ReadAt      string `json:"read_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// brain.InboxOutput{Success: true, Messages: []brain.MessageItem{{ID: 1, FromAgent: "charon", ToAgent: "cladius"}}}
type InboxOutput struct {
	Success  bool          `json:"success"`
	Messages []MessageItem `json:"messages"`
}

// brain.ConversationInput{Agent: "charon"}
type ConversationInput struct {
	Agent string `json:"agent"`
}

// brain.ConversationOutput{Success: true, Messages: []brain.MessageItem{{ID: 10, FromAgent: "cladius", ToAgent: "charon"}}}
type ConversationOutput struct {
	Success  bool          `json:"success"`
	Messages []MessageItem `json:"messages"`
}

var sendMessage = func(s *DirectSubsystem, ctx context.Context, _ *mcp.CallToolRequest, input SendInput) (*mcp.CallToolResult, SendOutput, error) {
	if input.To == "" || input.Content == "" {
		return nil, SendOutput{}, core.E("brain.sendMessage", "to and content are required", nil)
	}

	if input.To == "self" {
		s.notifySelf(ctx, input)
		return nil, SendOutput{Success: true, ID: 0, To: "self"}, nil
	}

	return sendRemoteMessage(s, ctx, input)
}

func (s *DirectSubsystem) notifySelf(ctx context.Context, input SendInput) {
	// "self" target: push via notifications/claude/channel directly.
	// Claude Code expects: { content: string, meta: Record<string, string> }
	if s.Core() == nil {
		return
	}
	mcpResult := s.Core().Service("mcp")
	if !mcpResult.OK {
		return
	}
	mcpSvc, ok := mcpResult.Value.(*coremcp.Service)
	if !ok {
		return
	}
	for session := range mcpSvc.Sessions() {
		coremcp.NotifySession(ctx, session, "notifications/claude/channel", map[string]any{
			"content": input.Content,
			"meta": map[string]string{
				"from":    agentic.AgentName(),
				"subject": input.Subject,
			},
		})
	}
}

var sendRemoteMessage = func(s *DirectSubsystem, ctx context.Context, input SendInput) (*mcp.CallToolResult, SendOutput, error) {
	result := s.apiCall(ctx, "POST", "/v1/messages/send", map[string]any{
		"to":      input.To,
		"from":    agentic.AgentName(),
		"content": input.Content,
		"subject": input.Subject,
	})
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, SendOutput{}, err
	}

	payload, _ := result.Value.(map[string]any)
	data, _ := payload["data"].(map[string]any)
	id, _ := data["id"].(float64)

	return nil, SendOutput{
		Success: true,
		ID:      int(id),
		To:      input.To,
	}, nil
}

var inbox = func(s *DirectSubsystem, ctx context.Context, _ *mcp.CallToolRequest, input InboxInput) (*mcp.CallToolResult, InboxOutput, error) {
	agent := input.Agent
	if agent == "" {
		agent = agentic.AgentName()
	}
	result := s.apiCall(ctx, "GET", core.Concat("/v1/messages/inbox?agent=", agent), nil)
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, InboxOutput{}, err
	}

	return nil, InboxOutput{
		Success:  true,
		Messages: parseMessages(result.Value.(map[string]any)),
	}, nil
}

var conversation = func(s *DirectSubsystem, ctx context.Context, _ *mcp.CallToolRequest, input ConversationInput) (*mcp.CallToolResult, ConversationOutput, error) {
	if input.Agent == "" {
		return nil, ConversationOutput{}, core.E("brain.conversation", "agent is required", nil)
	}

	result := s.apiCall(ctx, "GET", core.Concat("/v1/messages/conversation/", input.Agent, "?me=", agentic.AgentName()), nil)
	if !result.OK {
		err, _ := result.Value.(error)
		return nil, ConversationOutput{}, err
	}

	return nil, ConversationOutput{
		Success:  true,
		Messages: parseMessages(result.Value.(map[string]any)),
	}, nil
}

func parseMessages(result map[string]any) []MessageItem {
	var messages []MessageItem
	data, _ := result["data"].([]any)
	for _, m := range data {
		mm, _ := m.(map[string]any)
		from := stringFieldAny(mm, "from", "from_agent")
		to := stringFieldAny(mm, "to", "to_agent")
		messages = append(messages, MessageItem{
			ID:          toInt(mm["id"]),
			WorkspaceID: toInt(mm["workspace_id"]),
			From:        from,
			FromAgent:   from,
			To:          to,
			ToAgent:     to,
			Subject:     stringField(mm, "subject"),
			Content:     stringField(mm, "content"),
			Read:        mm["read"] == true,
			ReadAt:      stringField(mm, "read_at"),
			CreatedAt:   stringField(mm, "created_at"),
		})
	}
	return messages
}

func stringFieldAny(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, exists := values[key]
		if !exists {
			continue
		}
		if text := core.Trim(core.Sprint(value)); text != "" {
			return text
		}
	}
	return ""
}

func toInt(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}
