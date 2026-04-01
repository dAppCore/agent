// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessage_MessageSend_Good_PersistsAndReadsBack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)

	result := s.cmdMessageSend(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "from", Value: "codex"},
		core.Option{Key: "to", Value: "claude"},
		core.Option{Key: "subject", Value: "Review"},
		core.Option{Key: "content", Value: "Please check the prompt."},
	))
	require.True(t, result.OK)

	output, ok := result.Value.(MessageSendOutput)
	require.True(t, ok)
	assert.Equal(t, "core/go-io/task-5", output.Message.Workspace)
	assert.Equal(t, "codex", output.Message.FromAgent)
	assert.Equal(t, "claude", output.Message.ToAgent)
	assert.Equal(t, "Review", output.Message.Subject)
	assert.Equal(t, "Please check the prompt.", output.Message.Content)
	assert.NotEmpty(t, output.Message.ID)
	assert.NotEmpty(t, output.Message.CreatedAt)

	messageStorePath := messagePath("core/go-io/task-5")
	assert.True(t, fs.Exists(messageStorePath))

	inboxResult := s.cmdMessageInbox(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "claude"},
	))
	require.True(t, inboxResult.OK)

	inbox, ok := inboxResult.Value.(MessageListOutput)
	require.True(t, ok)
	assert.Equal(t, 1, inbox.Count)
	require.Len(t, inbox.Messages, 1)
	assert.Equal(t, output.Message.ID, inbox.Messages[0].ID)

	conversationResult := s.cmdMessageConversation(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "with", Value: "claude"},
	))
	require.True(t, conversationResult.OK)

	conversation, ok := conversationResult.Value.(MessageListOutput)
	require.True(t, ok)
	assert.Equal(t, 1, conversation.Count)
	require.Len(t, conversation.Messages, 1)
	assert.Equal(t, output.Message.ID, conversation.Messages[0].ID)
}

func TestMessage_MessageSend_Bad_MissingRequiredFields(t *testing.T) {
	s := newTestPrep(t)

	result := s.cmdMessageSend(core.NewOptions(
		core.Option{Key: "_arg", Value: "core/go-io/task-5"},
		core.Option{Key: "from", Value: "codex"},
	))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "required")
}

func TestMessage_MessageInbox_Ugly_CorruptStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CORE_WORKSPACE", dir)

	s := newTestPrep(t)
	require.True(t, fs.EnsureDir(messageRoot()).OK)
	require.True(t, fs.Write(messagePath("core/go-io/task-5"), "{broken json").OK)

	result := s.handleMessageInbox(context.Background(), core.NewOptions(
		core.Option{Key: "workspace", Value: "core/go-io/task-5"},
		core.Option{Key: "agent", Value: "claude"},
	))

	assert.False(t, result.OK)
	require.Error(t, result.Value.(error))
	assert.Contains(t, result.Value.(error).Error(), "failed to parse message store")
}
