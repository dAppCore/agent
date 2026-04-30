// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func (s *PrepSubsystem) cmdMessageSend(options core.Options) core.Result {
	workspace := optionStringValue(options, "workspace", "_arg")
	fromAgent := optionStringValue(options, "from_agent", "from")
	toAgent := optionStringValue(options, "to_agent", "to")
	content := optionStringValue(options, "content", "body")
	if workspace == "" || fromAgent == "" || toAgent == "" || core.Trim(content) == "" {
		core.Print(nil, "usage: core-agent message send <workspace> --from=codex --to=claude --subject=\"Review\" --content=\"Please check the prompt.\"")
		return core.Result{Value: core.E("agentic.cmdMessageSend", "workspace, from_agent, to_agent, and content are required", nil), OK: false}
	}

	result := s.handleMessageSend(s.commandContext(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspace},
		core.Option{Key: "from_agent", Value: fromAgent},
		core.Option{Key: "to_agent", Value: toAgent},
		core.Option{Key: "subject", Value: optionStringValue(options, "subject")},
		core.Option{Key: "content", Value: content},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdMessageSend", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(MessageSendOutput)
	if !ok {
		err := core.E("agentic.cmdMessageSend", "invalid message send output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "sent:     %s", output.Message.ID)
	core.Print(nil, "from:     %s", output.Message.FromAgent)
	core.Print(nil, "to:       %s", output.Message.ToAgent)
	if output.Message.Subject != "" {
		core.Print(nil, "subject:  %s", output.Message.Subject)
	}
	core.Print(nil, "content:  %s", output.Message.Content)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdMessageInbox(options core.Options) core.Result {
	workspace := optionStringValue(options, "workspace", "_arg")
	agent := optionStringValue(options, "agent", "agent_id", "agent-id")
	if workspace == "" || agent == "" {
		core.Print(nil, "usage: core-agent message inbox <workspace> --agent=claude [--limit=50]")
		return core.Result{Value: core.E("agentic.cmdMessageInbox", "workspace and agent are required", nil), OK: false}
	}

	result := s.handleMessageInbox(s.commandContext(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspace},
		core.Option{Key: "agent", Value: agent},
		core.Option{Key: "limit", Value: optionIntValue(options, "limit")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdMessageInbox", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(MessageListOutput)
	if !ok {
		err := core.E("agentic.cmdMessageInbox", "invalid message inbox output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if len(output.Messages) == 0 {
		core.Print(nil, "no messages")
		return core.Result{Value: output, OK: true}
	}

	core.Print(nil, "count: %d", output.Count)
	for _, message := range output.Messages {
		core.Print(nil, "  [%s] %s -> %s", message.CreatedAt, message.FromAgent, message.ToAgent)
		if message.Subject != "" {
			core.Print(nil, "    subject: %s", message.Subject)
		}
		core.Print(nil, "    %s", message.Content)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdMessageConversation(options core.Options) core.Result {
	workspace := optionStringValue(options, "workspace", "_arg")
	agent := optionStringValue(options, "agent", "agent_id", "agent-id")
	withAgent := optionStringValue(options, "with_agent", "with-agent", "with", "to_agent", "to-agent")
	if workspace == "" || agent == "" || withAgent == "" {
		core.Print(nil, "usage: core-agent message conversation <workspace> --agent=codex --with=claude [--limit=50]")
		return core.Result{Value: core.E("agentic.cmdMessageConversation", "workspace, agent, and with_agent are required", nil), OK: false}
	}

	result := s.handleMessageConversation(s.commandContext(), core.NewOptions(
		core.Option{Key: "workspace", Value: workspace},
		core.Option{Key: "agent", Value: agent},
		core.Option{Key: "with_agent", Value: withAgent},
		core.Option{Key: "limit", Value: optionIntValue(options, "limit")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdMessageConversation", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(MessageListOutput)
	if !ok {
		err := core.E("agentic.cmdMessageConversation", "invalid message conversation output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if len(output.Messages) == 0 {
		core.Print(nil, "no messages")
		return core.Result{Value: output, OK: true}
	}

	core.Print(nil, "count: %d", output.Count)
	for _, message := range output.Messages {
		core.Print(nil, "  [%s] %s -> %s", message.CreatedAt, message.FromAgent, message.ToAgent)
		if message.Subject != "" {
			core.Print(nil, "    subject: %s", message.Subject)
		}
		core.Print(nil, "    %s", message.Content)
	}
	return core.Result{Value: output, OK: true}
}
