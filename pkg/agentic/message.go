// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sort"
	"time"

	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// message := agentic.AgentMessage{Workspace: "core/go-io/task-5", FromAgent: "codex", ToAgent: "claude", Subject: "Review", Content: "Please check the prompt."}
type AgentMessage struct {
	ID        string `json:"id"`
	Workspace string `json:"workspace"`
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Subject   string `json:"subject,omitempty"`
	Content   string `json:"content"`
	ReadAt    string `json:"read_at,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// input := agentic.MessageSendInput{Workspace: "core/go-io/task-5", FromAgent: "codex", ToAgent: "claude", Subject: "Review", Content: "Please check the prompt."}
type MessageSendInput struct {
	Workspace string `json:"workspace"`
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Subject   string `json:"subject,omitempty"`
	Content   string `json:"content"`
}

// input := agentic.MessageInboxInput{Workspace: "core/go-io/task-5", Agent: "claude"}
type MessageInboxInput struct {
	Workspace string `json:"workspace"`
	Agent     string `json:"agent"`
	Limit     int    `json:"limit,omitempty"`
}

// input := agentic.MessageConversationInput{Workspace: "core/go-io/task-5", Agent: "codex", WithAgent: "claude"}
type MessageConversationInput struct {
	Workspace string `json:"workspace"`
	Agent     string `json:"agent"`
	WithAgent string `json:"with_agent"`
	Limit     int    `json:"limit,omitempty"`
}

// out := agentic.MessageSendOutput{Success: true, Message: agentic.AgentMessage{ID: "msg-1"}}
type MessageSendOutput struct {
	Success bool         `json:"success"`
	Message AgentMessage `json:"message"`
}

// out := agentic.MessageListOutput{Success: true, Count: 1, Messages: []agentic.AgentMessage{{ID: "msg-1"}}}
type MessageListOutput struct {
	Success  bool           `json:"success"`
	New      int            `json:"new,omitempty"`
	Count    int            `json:"count"`
	Messages []AgentMessage `json:"messages"`
}

// result := c.Action("agentic.message.send").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	core.Option{Key: "from_agent", Value: "codex"},
//	core.Option{Key: "to_agent", Value: "claude"},
//
// ))
func (s *PrepSubsystem) handleMessageSend(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.messageSend(ctx, nil, MessageSendInput{
		Workspace: optionStringValue(options, "workspace", "_arg"),
		FromAgent: optionStringValue(options, "from_agent", "from"),
		ToAgent:   optionStringValue(options, "to_agent", "to"),
		Subject:   optionStringValue(options, "subject"),
		Content:   optionStringValue(options, "content", "body"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.message.inbox").Run(ctx, core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-5"}))
func (s *PrepSubsystem) handleMessageInbox(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.messageInbox(ctx, nil, MessageInboxInput{
		Workspace: optionStringValue(options, "workspace", "_arg"),
		Agent:     optionStringValue(options, "agent", "agent_id", "agent-id"),
		Limit:     optionIntValue(options, "limit"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	if s.Core() != nil {
		s.Core().ACTION(messages.InboxMessage{
			New:   output.New,
			Total: output.Count,
		})
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("agentic.message.conversation").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	core.Option{Key: "agent", Value: "codex"},
//	core.Option{Key: "with_agent", Value: "claude"},
//
// ))
func (s *PrepSubsystem) handleMessageConversation(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.messageConversation(ctx, nil, MessageConversationInput{
		Workspace: optionStringValue(options, "workspace", "_arg"),
		Agent:     optionStringValue(options, "agent", "agent_id", "agent-id"),
		WithAgent: optionStringValue(options, "with_agent", "with-agent", "with", "to_agent", "to-agent"),
		Limit:     optionIntValue(options, "limit"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerMessageTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_message_send",
		Description: "Send a direct message between two agents within a workspace.",
	}, s.messageSend)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_message_inbox",
		Description: "List messages delivered to an agent within a workspace.",
	}, s.messageInbox)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_message_conversation",
		Description: "List the chronological conversation between two agents within a workspace.",
	}, s.messageConversation)
}

func (s *PrepSubsystem) messageSend(_ context.Context, _ *mcp.CallToolRequest, input MessageSendInput) (*mcp.CallToolResult, MessageSendOutput, error) {
	message, err := messageStoreSend(input)
	if err != nil {
		return nil, MessageSendOutput{}, err
	}
	return nil, MessageSendOutput{Success: true, Message: message}, nil
}

func (s *PrepSubsystem) messageInbox(_ context.Context, _ *mcp.CallToolRequest, input MessageInboxInput) (*mcp.CallToolResult, MessageListOutput, error) {
	messages, newCount, err := messageStoreInbox(input.Workspace, input.Agent, input.Limit)
	if err != nil {
		return nil, MessageListOutput{}, err
	}
	return nil, MessageListOutput{Success: true, New: newCount, Count: len(messages), Messages: messages}, nil
}

func (s *PrepSubsystem) messageConversation(_ context.Context, _ *mcp.CallToolRequest, input MessageConversationInput) (*mcp.CallToolResult, MessageListOutput, error) {
	messages, err := messageStoreConversation(input.Workspace, input.Agent, input.WithAgent, input.Limit)
	if err != nil {
		return nil, MessageListOutput{}, err
	}
	return nil, MessageListOutput{Success: true, Count: len(messages), Messages: messages}, nil
}

func messageStoreSend(input MessageSendInput) (AgentMessage, error) {
	if input.Workspace == "" {
		return AgentMessage{}, core.E("messageSend", "workspace is required", nil)
	}
	if input.FromAgent == "" {
		return AgentMessage{}, core.E("messageSend", "from_agent is required", nil)
	}
	if input.ToAgent == "" {
		return AgentMessage{}, core.E("messageSend", "to_agent is required", nil)
	}
	if core.Trim(input.Content) == "" {
		return AgentMessage{}, core.E("messageSend", "content is required", nil)
	}

	messages, err := readWorkspaceMessages(input.Workspace)
	if err != nil {
		return AgentMessage{}, err
	}

	now := time.Now().Format(time.RFC3339)
	message := AgentMessage{
		ID:        messageID(),
		Workspace: core.Trim(input.Workspace),
		FromAgent: core.Trim(input.FromAgent),
		ToAgent:   core.Trim(input.ToAgent),
		Subject:   core.Trim(input.Subject),
		Content:   input.Content,
		CreatedAt: now,
	}
	messages = append(messages, message)

	if err := writeWorkspaceMessages(input.Workspace, messages); err != nil {
		return AgentMessage{}, err
	}

	return message, nil
}

func messageStoreInbox(workspace, agent string, limit int) ([]AgentMessage, int, error) {
	if workspace == "" {
		return nil, 0, core.E("messageInbox", "workspace is required", nil)
	}
	if agent == "" {
		return nil, 0, core.E("messageInbox", "agent is required", nil)
	}

	messages, err := readWorkspaceMessages(workspace)
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 50
	}

	now := time.Now().Format(time.RFC3339)
	inbox := make([]AgentMessage, 0, len(messages))
	newCount := 0
	changed := false

	for i := range messages {
		message := normaliseAgentMessage(messages[i])
		if message.ToAgent != agent {
			messages[i] = message
			continue
		}
		if message.ReadAt == "" {
			message.ReadAt = now
			newCount++
			changed = true
		}
		messages[i] = message
		inbox = append(inbox, message)
	}

	if changed {
		if err := writeWorkspaceMessages(workspace, messages); err != nil {
			return nil, 0, err
		}
	}

	if len(inbox) > limit {
		inbox = inbox[len(inbox)-limit:]
	}

	return inbox, newCount, nil
}

func messageStoreConversation(workspace, agent, withAgent string, limit int) ([]AgentMessage, error) {
	if workspace == "" {
		return nil, core.E("messageConversation", "workspace is required", nil)
	}
	if agent == "" {
		return nil, core.E("messageConversation", "agent is required", nil)
	}
	if withAgent == "" {
		return nil, core.E("messageConversation", "with_agent is required", nil)
	}

	return messageStoreFilter(workspace, limit, func(message AgentMessage) bool {
		return (message.FromAgent == agent && message.ToAgent == withAgent) || (message.FromAgent == withAgent && message.ToAgent == agent)
	})
}

func messageStoreFilter(workspace string, limit int, match func(AgentMessage) bool) ([]AgentMessage, error) {
	messages, err := readWorkspaceMessages(workspace)
	if err != nil {
		return nil, err
	}

	filtered := make([]AgentMessage, 0, len(messages))
	for _, message := range messages {
		message = normaliseAgentMessage(message)
		if match(message) {
			filtered = append(filtered, message)
		}
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt < filtered[j].CreatedAt
	})

	if limit <= 0 {
		limit = 50
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	return filtered, nil
}

func messageRoot() string {
	return core.JoinPath(CoreRoot(), "messages")
}

func messagePath(workspace string) string {
	return core.JoinPath(messageRoot(), core.Concat(core.SanitisePath(workspace), ".json"))
}

func readWorkspaceMessages(workspace string) ([]AgentMessage, error) {
	if workspace == "" {
		return []AgentMessage{}, nil
	}

	result := fs.Read(messagePath(workspace))
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil || core.Contains(err.Error(), "no such file") {
			return []AgentMessage{}, nil
		}
		return nil, core.E("readWorkspaceMessages", "failed to read message store", err)
	}

	content := core.Trim(result.Value.(string))
	if content == "" {
		return []AgentMessage{}, nil
	}

	var messages []AgentMessage
	if parseResult := core.JSONUnmarshalString(content, &messages); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return nil, core.E("readWorkspaceMessages", "failed to parse message store", err)
	}

	for i := range messages {
		messages[i] = normaliseAgentMessage(messages[i])
	}

	sort.SliceStable(messages, func(i, j int) bool {
		return messages[i].CreatedAt < messages[j].CreatedAt
	})

	return messages, nil
}

func writeWorkspaceMessages(workspace string, messages []AgentMessage) error {
	if workspace == "" {
		return core.E("writeWorkspaceMessages", "workspace is required", nil)
	}

	normalised := make([]AgentMessage, 0, len(messages))
	for _, message := range messages {
		normalised = append(normalised, normaliseAgentMessage(message))
	}

	if ensureDirResult := fs.EnsureDir(messageRoot()); !ensureDirResult.OK {
		err, _ := ensureDirResult.Value.(error)
		return core.E("writeWorkspaceMessages", "failed to create message store directory", err)
	}

	if writeResult := fs.WriteAtomic(messagePath(workspace), core.JSONMarshalString(normalised)); !writeResult.OK {
		err, _ := writeResult.Value.(error)
		return core.E("writeWorkspaceMessages", "failed to write message store", err)
	}

	return nil
}

func normaliseAgentMessage(message AgentMessage) AgentMessage {
	message.Workspace = core.Trim(message.Workspace)
	message.FromAgent = core.Trim(message.FromAgent)
	message.ToAgent = core.Trim(message.ToAgent)
	message.Subject = core.Trim(message.Subject)
	if message.ID == "" {
		message.ID = messageID()
	}
	if message.CreatedAt == "" {
		message.CreatedAt = time.Now().Format(time.RFC3339)
	}
	return message
}

func messageID() string {
	return core.Concat("msg-", core.Sprint(time.Now().UnixNano()))
}
