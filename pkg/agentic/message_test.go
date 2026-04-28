// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

func TestMessage_MessageSend_Good_PersistsAndReadsBack(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	result := s.cmdMessageSend(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "from", Value: "codex"},
		core.Option{Key: "to", Value: "claude"},
		core.Option{Key: "subject", Value: "Review"},
		core.Option{Key: "content", Value: "Please check the prompt."},
	))
	core.RequireTrue(t, result.OK)

	output, ok := result.Value.(MessageSendOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "core/go-io/task-5", output.Message.Workspace)
	core.AssertEqual(t, "codex", output.Message.FromAgent)
	core.AssertEqual(t, "claude", output.Message.ToAgent)
	core.AssertEqual(t, "Review", output.Message.Subject)
	core.AssertEqual(t, "Please check the prompt.", output.Message.Content)
	core.AssertNotEmpty(t, output.Message.ID)
	core.AssertNotEmpty(t, output.Message.CreatedAt)

	messageStorePath := messagePath("core/go-io/task-5")
	core.AssertTrue(t, fs.Exists(messageStorePath))

	inboxResult := s.cmdMessageInbox(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "claude"},
	))
	core.RequireTrue(t, inboxResult.OK)

	inbox, ok := inboxResult.Value.(MessageListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, inbox.Count)
	core.AssertLen(t, inbox.Messages, 1)
	core.AssertEqual(t, output.Message.ID, inbox.Messages[0].ID)

	conversationResult := s.cmdMessageConversation(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "with", Value: "claude"},
	))
	core.RequireTrue(t, conversationResult.OK)

	conversation, ok := conversationResult.Value.(MessageListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, conversation.Count)
	core.AssertLen(t, conversation.Messages, 1)
	core.AssertEqual(t, output.Message.ID, conversation.Messages[0].ID)
}

func TestMessage_MessageInbox_Good_MarksReadAndEmitsCounts(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	c := core.New()
	var inboxEvents []messages.InboxMessage
	c.RegisterAction(func(_ *core.Core, msg core.Message) core.Result {
		if ev, ok := msg.(messages.InboxMessage); ok {
			inboxEvents = append(inboxEvents, ev)
		}
		return core.Result{OK: true}
	})

	s := &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}

	sendResult := s.handleMessageSend(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: "core/go-io/task-5"},
		core.Option{Key: "from_agent", Value: "codex"},
		core.Option{Key: "to_agent", Value: "claude"},
		core.Option{Key: "content", Value: "Please review this."},
	))
	core.RequireTrue(t, sendResult.OK)

	inboxResult := s.handleMessageInbox(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "claude"},
	))
	core.RequireTrue(t, inboxResult.OK)

	inbox, ok := inboxResult.Value.(MessageListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 1, inbox.New)
	core.AssertEqual(t, 1, inbox.Count)
	core.AssertLen(t, inbox.Messages, 1)
	core.AssertNotEmpty(t, inbox.Messages[0].ReadAt)

	secondResult := s.handleMessageInbox(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "claude"},
	))
	core.RequireTrue(t, secondResult.OK)

	secondInbox, ok := secondResult.Value.(MessageListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 0, secondInbox.New)
	core.AssertLen(t, inboxEvents, 2)
	core.AssertEqual(t, 1, inboxEvents[0].New)
	core.AssertEqual(t, 1, inboxEvents[0].Total)
	core.AssertEqual(t, 0, inboxEvents[1].New)
	core.AssertEqual(t, 1, inboxEvents[1].Total)
}

func TestMessage_MessageSend_Bad_MissingRequiredFields(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdMessageSend(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "from", Value: "codex"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "required")
}

func TestMessage_MessageSend_Ugly_WhitespaceContent(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdMessageSend(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "from", Value: "codex"},
		core.Option{Key: "to", Value: "claude"},
		core.Option{Key: "content", Value: "   "},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "required")
}

func TestMessage_MessageInbox_Good_NoMessages(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	result := s.cmdMessageInbox(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-empty"},
		core.Option{Key: "agent", Value: "claude"},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(MessageListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 0, output.Count)
	core.AssertEmpty(t, output.Messages)
}

func TestMessage_MessageInbox_Bad_MissingRequiredFields(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdMessageInbox(core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "required")
}

func TestMessage_HandleMessageInbox_Ugly_CorruptStore(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	core.RequireTrue(t, fs.EnsureDir(messageRoot()).OK)
	core.RequireTrue(t, fs.Write(messagePath("core/go-io/task-5"), "{broken json").OK)

	s := newTestPrep(t)

	result := s.cmdMessageInbox(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "claude"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "failed to parse message store")
}

func TestMessage_MessageConversation_Good_NoMessages(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)

	result := s.cmdMessageConversation(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-empty"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "with", Value: "claude"},
	))

	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(MessageListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 0, output.Count)
	core.AssertEmpty(t, output.Messages)
}

func TestMessage_MessageConversation_Bad_MissingRequiredFields(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdMessageConversation(core.NewOptions())

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "required")
}

func TestMessage_MessageConversation_Ugly_CorruptStore(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	core.RequireTrue(t, fs.EnsureDir(messageRoot()).OK)
	core.RequireTrue(t, fs.Write(messagePath("core/go-io/task-5"), "{broken json").OK)

	s := newTestPrep(t)

	result := s.cmdMessageConversation(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "with", Value: "claude"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "failed to parse message store")
}

func TestMessage_MessageInbox_Ugly_CorruptStore(t *testing.T) {
	dir := t.TempDir()
	setTestWorkspace(t, dir)

	s := newTestPrep(t)
	core.RequireTrue(t, fs.EnsureDir(messageRoot()).OK)
	core.RequireTrue(t, fs.Write(messagePath("core/go-io/task-5"), "{broken json").OK)

	result := s.handleMessageInbox(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "claude"},
	))

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
	core.AssertContains(t, result.Value.(error).Error(), "failed to parse message store")
}
