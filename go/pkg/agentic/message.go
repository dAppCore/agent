// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"slices"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// message := agentic.AgentMessage{Workspace: "core/go-io/task-5", FromAgent: "codex", ToAgent: "claude", Subject: "Review", Content: "Please check the prompt."}
type AgentMessage struct {
	ID          string `json:"id"`
	WorkspaceID int    `json:"workspace_id,omitempty"`
	Workspace   string `json:"workspace"`
	FromAgent   string `json:"from_agent"`
	ToAgent     string `json:"to_agent"`
	Subject     string `json:"subject,omitempty"`
	Content     string `json:"content"`
	ReadAt      string `json:"read_at,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// input := agentic.MessageSendInput{Workspace: "core/go-io/task-5", FromAgent: "codex", ToAgent: "claude", Subject: "Review", Content: "Please check the prompt."}
type MessageSendInput struct {
	WorkspaceID int    `json:"workspace_id,omitempty"`
	Workspace   string `json:"workspace"`
	FromAgent   string `json:"from_agent"`
	ToAgent     string `json:"to_agent"`
	Subject     string `json:"subject,omitempty"`
	Content     string `json:"content"`
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
	return typedResultValue[MessageSendOutput]("message.send", "invalid message send output", s.messageSend(ctx, MessageSendInput{
		WorkspaceID: optionIntValue(options, "workspace_id", "workspace-id"),
		Workspace:   optionStringValue(options, "workspace", "_arg"),
		FromAgent:   optionStringValue(options, "from_agent", "from"),
		ToAgent:     optionStringValue(options, "to_agent", "to"),
		Subject:     optionStringValue(options, "subject"),
		Content:     optionStringValue(options, "content", "body"),
	}))
}

// result := c.Action("agentic.message.inbox").Run(ctx, core.NewOptions(core.Option{Key: "workspace", Value: "core/go-io/task-5"}))
func (s *PrepSubsystem) handleMessageInbox(ctx context.Context, options core.Options) core.Result {
	result := typedResultValue[MessageListOutput]("message.inbox", "invalid message inbox output", s.messageInbox(ctx, MessageInboxInput{
		Workspace: optionStringValue(options, "workspace", "_arg"),
		Agent:     optionStringValue(options, "agent", "agent_id", "agent-id"),
		Limit:     optionIntValue(options, "limit"),
	}))
	if !result.OK {
		return result
	}
	output, _ := result.Value.(MessageListOutput)
	if s.Core() != nil {
		// Best-effort: a listener that has gone away must not fail this.
		_ = s.Core().ACTION(messages.InboxMessage{
			New:   output.New,
			Total: output.Count,
		})
	}
	return core.Ok(output)
}

// result := c.Action("agentic.message.conversation").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "workspace", Value: "core/go-io/task-5"},
//	core.Option{Key: "agent", Value: "codex"},
//	core.Option{Key: "with_agent", Value: "claude"},
//
// ))
func (s *PrepSubsystem) handleMessageConversation(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[MessageListOutput]("message.conversation", "invalid message conversation output", s.messageConversation(ctx, MessageConversationInput{
		Workspace: optionStringValue(options, "workspace", "_arg"),
		Agent:     optionStringValue(options, "agent", "agent_id", "agent-id"),
		WithAgent: optionStringValue(options, "with_agent", "with-agent", "with", "to_agent", "to-agent"),
		Limit:     optionIntValue(options, "limit"),
	}))
}

func (s *PrepSubsystem) registerMessageTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_message_send",
		Description: "Send a direct message between two agents within a workspace.",
	}, toolHandlerFor[MessageSendInput, MessageSendOutput]("message.send", "invalid message send output", s.messageSend))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agent_send",
		Description: "Send a direct message between two agents within a workspace.",
	}, toolHandlerFor[MessageSendInput, MessageSendOutput]("message.send", "invalid message send output", s.messageSend))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_message_inbox",
		Description: "List messages delivered to an agent within a workspace.",
	}, toolHandlerFor[MessageInboxInput, MessageListOutput]("message.inbox", "invalid message inbox output", s.messageInbox))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agent_inbox",
		Description: "List messages delivered to an agent within a workspace.",
	}, toolHandlerFor[MessageInboxInput, MessageListOutput]("message.inbox", "invalid message inbox output", s.messageInbox))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_message_conversation",
		Description: "List the chronological conversation between two agents within a workspace.",
	}, toolHandlerFor[MessageConversationInput, MessageListOutput]("message.conversation", "invalid message conversation output", s.messageConversation))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agent_conversation",
		Description: "List the chronological conversation between two agents within a workspace.",
	}, toolHandlerFor[MessageConversationInput, MessageListOutput]("message.conversation", "invalid message conversation output", s.messageConversation))
}

func (s *PrepSubsystem) messageSend(_ context.Context, input MessageSendInput) core.Result {
	// "self" target: push directly via MCP channel, skip the brain API.
	// Use for testing channel notifications without a running server.
	if input.ToAgent == "self" {
		msg := AgentMessage{
			ID:        core.ID(),
			Workspace: input.Workspace,
			FromAgent: input.FromAgent,
			ToAgent:   "self",
			Subject:   input.Subject,
			Content:   input.Content,
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		if s.ServiceRuntime != nil {
			// Best-effort: a listener that has gone away must not fail this.
			_ = s.Core().ACTION(coremcp.ChannelPush{
				Channel: coremcp.ChannelInboxMessage,
				Data: map[string]any{
					"id":      msg.ID,
					"from":    msg.FromAgent,
					"to":      "self",
					"subject": msg.Subject,
					"content": msg.Content,
				},
			})
		}
		return core.Ok(MessageSendOutput{Success: true, Message: msg})
	}

	messageResult := messageStoreSend(input)
	if !messageResult.OK {
		return messageResult
	}
	message, _ := messageResult.Value.(AgentMessage)
	return core.Ok(MessageSendOutput{Success: true, Message: message})
}

func (s *PrepSubsystem) messageInbox(_ context.Context, input MessageInboxInput) core.Result {
	inboxResult := messageStoreInbox(input.Workspace, input.Agent, input.Limit)
	if !inboxResult.OK {
		return inboxResult
	}
	inbox, _ := inboxResult.Value.(messageInboxValue)
	return core.Ok(MessageListOutput{Success: true, New: inbox.New, Count: len(inbox.Messages), Messages: inbox.Messages})
}

func (s *PrepSubsystem) messageConversation(_ context.Context, input MessageConversationInput) core.Result {
	conversationResult := messageStoreConversation(input.Workspace, input.Agent, input.WithAgent, input.Limit)
	if !conversationResult.OK {
		return conversationResult
	}
	messages, _ := conversationResult.Value.([]AgentMessage)
	return core.Ok(MessageListOutput{Success: true, Count: len(messages), Messages: messages})
}

func messageStoreSend(input MessageSendInput) core.Result {
	if input.Workspace == "" {
		return core.Fail(core.E("messageSend", "workspace is required", nil))
	}
	if input.FromAgent == "" {
		return core.Fail(core.E("messageSend", "from_agent is required", nil))
	}
	if input.ToAgent == "" {
		return core.Fail(core.E("messageSend", "to_agent is required", nil))
	}
	if core.Trim(input.Content) == "" {
		return core.Fail(core.E("messageSend", "content is required", nil))
	}

	messagesResult := readWorkspaceMessages(input.Workspace)
	if !messagesResult.OK {
		return messagesResult
	}
	messages, _ := messagesResult.Value.([]AgentMessage)

	now := time.Now().Format(time.RFC3339)
	message := AgentMessage{
		ID:          messageID(),
		WorkspaceID: input.WorkspaceID,
		Workspace:   core.Trim(input.Workspace),
		FromAgent:   core.Trim(input.FromAgent),
		ToAgent:     core.Trim(input.ToAgent),
		Subject:     core.Trim(input.Subject),
		Content:     input.Content,
		CreatedAt:   now,
	}
	messages = append(messages, message)

	writeResult := writeWorkspaceMessages(input.Workspace, messages)
	if !writeResult.OK {
		return writeResult
	}

	return core.Ok(message)
}

type messageInboxValue struct {
	Messages []AgentMessage
	New      int
}

func messageStoreInbox(workspace, agent string, limit int) core.Result {
	if workspace == "" {
		return core.Fail(core.E("messageInbox", "workspace is required", nil))
	}
	if agent == "" {
		return core.Fail(core.E("messageInbox", "agent is required", nil))
	}

	messagesResult := readWorkspaceMessages(workspace)
	if !messagesResult.OK {
		return messagesResult
	}
	messages, _ := messagesResult.Value.([]AgentMessage)

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
		if writeResult := writeWorkspaceMessages(workspace, messages); !writeResult.OK {
			return writeResult
		}
	}

	if len(inbox) > limit {
		inbox = inbox[len(inbox)-limit:]
	}

	return core.Ok(messageInboxValue{Messages: inbox, New: newCount})
}

func messageStoreConversation(workspace, agent, withAgent string, limit int) core.Result {
	if workspace == "" {
		return core.Fail(core.E("messageConversation", "workspace is required", nil))
	}
	if agent == "" {
		return core.Fail(core.E("messageConversation", "agent is required", nil))
	}
	if withAgent == "" {
		return core.Fail(core.E("messageConversation", "with_agent is required", nil))
	}

	return messageStoreFilter(workspace, limit, func(message AgentMessage) bool {
		return (message.FromAgent == agent && message.ToAgent == withAgent) || (message.FromAgent == withAgent && message.ToAgent == agent)
	})
}

func messageStoreFilter(workspace string, limit int, match func(AgentMessage) bool) core.Result {
	messagesResult := readWorkspaceMessages(workspace)
	if !messagesResult.OK {
		return messagesResult
	}
	messages, _ := messagesResult.Value.([]AgentMessage)

	filtered := make([]AgentMessage, 0, len(messages))
	for _, message := range messages {
		message = normaliseAgentMessage(message)
		if match(message) {
			filtered = append(filtered, message)
		}
	}

	slices.SortStableFunc(filtered, func(a, b AgentMessage) int {
		switch {
		case a.CreatedAt < b.CreatedAt:
			return -1
		case a.CreatedAt > b.CreatedAt:
			return 1
		default:
			return 0
		}
	})

	if limit <= 0 {
		limit = 50
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	return core.Ok(filtered)
}

func messageRoot() string {
	return core.JoinPath(CoreRoot(), "messages")
}

func messagePath(workspace string) string {
	return core.JoinPath(messageRoot(), core.Concat(pathKey(workspace), ".json"))
}

func readWorkspaceMessages(workspace string) core.Result {
	if workspace == "" {
		return core.Ok([]AgentMessage{})
	}

	result := fs.Read(messagePath(workspace))
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil || core.Contains(err.Error(), "no such file") {
			return core.Ok([]AgentMessage{})
		}
		return core.Fail(core.E("readWorkspaceMessages", "failed to read message store", err))
	}

	content := core.Trim(result.Value.(string))
	if content == "" {
		return core.Ok([]AgentMessage{})
	}

	var messages []AgentMessage
	if parseResult := core.JSONUnmarshalString(content, &messages); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return core.Fail(core.E("readWorkspaceMessages", "failed to parse message store", err))
	}

	for i := range messages {
		messages[i] = normaliseAgentMessage(messages[i])
	}

	slices.SortStableFunc(messages, func(a, b AgentMessage) int {
		switch {
		case a.CreatedAt < b.CreatedAt:
			return -1
		case a.CreatedAt > b.CreatedAt:
			return 1
		default:
			return 0
		}
	})

	return core.Ok(messages)
}

func writeWorkspaceMessages(workspace string, messages []AgentMessage) core.Result {
	if workspace == "" {
		return core.Fail(core.E("writeWorkspaceMessages", "workspace is required", nil))
	}

	normalised := make([]AgentMessage, 0, len(messages))
	for _, message := range messages {
		normalised = append(normalised, normaliseAgentMessage(message))
	}

	if ensureDirResult := fs.EnsureDir(messageRoot()); !ensureDirResult.OK {
		err, _ := ensureDirResult.Value.(error)
		return core.Fail(core.E("writeWorkspaceMessages", "failed to create message store directory", err))
	}

	if writeResult := fs.WriteAtomic(messagePath(workspace), core.JSONMarshalString(normalised)); !writeResult.OK {
		err, _ := writeResult.Value.(error)
		return core.Fail(core.E("writeWorkspaceMessages", "failed to write message store", err))
	}

	return core.Ok(nil)
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
