// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	core "dappco.re/go/core"
)

type DirectOptions struct{}

// subsystem := brain.NewDirect()
// _ = subsystem.OnStartup(context.Background())
func (s *DirectSubsystem) OnStartup(_ context.Context) core.Result {
	if s.ServiceRuntime == nil || s.Core() == nil {
		return core.Result{OK: true}
	}

	c := s.Core()
	c.Action("brain.remember", s.handleRemember).Description = "Store knowledge in OpenBrain"
	c.Action("brain.recall", s.handleRecall).Description = "Recall knowledge from OpenBrain"
	c.Action("brain.forget", s.handleForget).Description = "Forget knowledge in OpenBrain"
	c.Action("brain.list", s.handleList).Description = "List knowledge in OpenBrain"
	c.Action("message.send", s.handleSend).Description = "Send a direct message to another agent"
	c.Action("message.inbox", s.handleInbox).Description = "Read direct messages for an agent"
	c.Action("message.conversation", s.handleConversation).Description = "Read the conversation thread with another agent"
	c.Action("agent.send", s.handleSend).Description = "Send a direct message to another agent"
	c.Action("agent.inbox", s.handleInbox).Description = "Read direct messages for an agent"
	c.Action("agent.conversation", s.handleConversation).Description = "Read the conversation thread with another agent"
	return core.Result{OK: true}
}

// result := c.Action("brain.remember").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "content", Value: "Use OpenBrain for cross-agent context"},
//	core.Option{Key: "type", Value: "architecture"},
//
// ))
func (s *DirectSubsystem) handleRemember(ctx context.Context, options core.Options) core.Result {
	input := RememberInput{
		Content:    actionStringValue(options, "content"),
		Type:       actionStringValue(options, "type"),
		Tags:       actionStringSliceValue(options, "tags"),
		Org:        actionStringValue(options, "org"),
		Project:    actionStringValue(options, "project"),
		Confidence: actionFloatValue(options, "confidence"),
		Supersedes: actionStringValue(options, "supersedes"),
		ExpiresIn:  actionIntValue(options, "expires_in", "expiresIn"),
	}
	_, output, err := s.remember(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("brain.recall").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "query", Value: "OpenBrain architecture"},
//	core.Option{Key: "top_k", Value: 5},
//
// ))
func (s *DirectSubsystem) handleRecall(ctx context.Context, options core.Options) core.Result {
	input := RecallInput{
		Query:  actionStringValue(options, "query"),
		TopK:   actionIntValue(options, "top_k", "topK"),
		Filter: recallFilterFromOptions(options),
	}
	_, output, err := s.recall(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("brain.forget").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "id", Value: "mem-123"},
//
// ))
func (s *DirectSubsystem) handleForget(ctx context.Context, options core.Options) core.Result {
	input := ForgetInput{
		ID:     actionStringValue(options, "id"),
		Reason: actionStringValue(options, "reason"),
	}
	_, output, err := s.forget(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("brain.list").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "project", Value: "agent"},
//	core.Option{Key: "limit", Value: 10},
//
// ))
func (s *DirectSubsystem) handleList(ctx context.Context, options core.Options) core.Result {
	input := ListInput{
		Org:     actionStringValue(options, "org"),
		Project: actionStringValue(options, "project"),
		Type:    actionStringValue(options, "type"),
		AgentID: actionStringValue(options, "agent_id", "agent"),
		Limit:   actionIntValue(options, "limit"),
	}
	_, output, err := s.list(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("message.send").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "to", Value: "charon"},
//	core.Option{Key: "content", Value: "Deploy complete"},
//
// ))
func (s *DirectSubsystem) handleSend(ctx context.Context, options core.Options) core.Result {
	input := SendInput{
		To:      actionStringValue(options, "to"),
		Content: actionStringValue(options, "content"),
		Subject: actionStringValue(options, "subject"),
	}
	_, output, err := s.sendMessage(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("message.inbox").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "agent", Value: "cladius"},
//
// ))
func (s *DirectSubsystem) handleInbox(ctx context.Context, options core.Options) core.Result {
	input := InboxInput{
		Agent: actionStringValue(options, "agent"),
	}
	_, output, err := s.inbox(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("message.conversation").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "agent", Value: "charon"},
//
// ))
func (s *DirectSubsystem) handleConversation(ctx context.Context, options core.Options) core.Result {
	input := ConversationInput{
		Agent: actionStringValue(options, "agent"),
	}
	_, output, err := s.conversation(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func recallFilterFromOptions(options core.Options) RecallFilter {
	filter := recallFilterValue(actionOptionValue(options, "filter"))
	if filter.Org == "" {
		filter.Org = actionStringValue(options, "org")
	}
	if filter.Project == "" {
		filter.Project = actionStringValue(options, "project")
	}
	if filter.Type == nil {
		filter.Type = actionOptionValue(options, "type")
	}
	if filter.AgentID == "" {
		filter.AgentID = actionStringValue(options, "agent_id", "agent")
	}
	if filter.MinConfidence == 0 {
		filter.MinConfidence = actionFloatValue(options, "min_confidence", "minConfidence")
	}
	return filter
}

func recallFilterValue(value any) RecallFilter {
	switch typed := value.(type) {
	case RecallFilter:
		return typed
	case map[string]any:
		return RecallFilter{
			Project:       actionStringFromAny(typed["project"]),
			Type:          typed["type"],
			AgentID:       actionStringFromAny(typed["agent_id"]),
			Org:           actionStringFromAny(typed["org"]),
			MinConfidence: actionFloatFromAny(typed["min_confidence"]),
		}
	case map[string]string:
		return RecallFilter{
			Project: actionStringFromAny(typed["project"]),
			Type:    typed["type"],
			AgentID: actionStringFromAny(typed["agent_id"]),
			Org:     actionStringFromAny(typed["org"]),
		}
	default:
		if text := actionStringFromAny(value); text != "" {
			return RecallFilter{Type: text}
		}
	}
	return RecallFilter{}
}

func actionOptionValue(options core.Options, keys ...string) any {
	for _, key := range keys {
		result := options.Get(key)
		if result.OK {
			return result.Value
		}
	}
	return nil
}

func actionStringValue(options core.Options, keys ...string) string {
	return actionStringFromAny(actionOptionValue(options, keys...))
}

func actionIntValue(options core.Options, keys ...string) int {
	return actionIntFromAny(actionOptionValue(options, keys...))
}

func actionFloatValue(options core.Options, keys ...string) float64 {
	return actionFloatFromAny(actionOptionValue(options, keys...))
}

func actionStringSliceValue(options core.Options, keys ...string) []string {
	return actionStringSliceFromAny(actionOptionValue(options, keys...))
}

func actionStringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return core.Trim(typed)
	case int:
		return core.Sprint(typed)
	case int64:
		return core.Sprint(typed)
	case float64:
		return core.Sprint(int(typed))
	case bool:
		return core.Sprint(typed)
	}
	return ""
}

func actionIntFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return 0
		}
		var parsed int
		if result := core.JSONUnmarshalString(core.Concat("{\"n\":", trimmed, "}"), &struct {
			N *int `json:"n"`
		}{N: &parsed}); result.OK {
			return parsed
		}
	}
	return 0
}

func actionFloatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return 0
		}
		var parsed float64
		if result := core.JSONUnmarshalString(core.Concat("{\"n\":", trimmed, "}"), &struct {
			N *float64 `json:"n"`
		}{N: &parsed}); result.OK {
			return parsed
		}
	}
	return 0
}

func actionStringSliceFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return cleanActionStrings(typed)
	case []any:
		var values []string
		for _, item := range typed {
			if text := actionStringFromAny(item); text != "" {
				values = append(values, text)
			}
		}
		return cleanActionStrings(values)
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		if core.HasPrefix(trimmed, "[") {
			var values []string
			if result := core.JSONUnmarshalString(trimmed, &values); result.OK {
				return cleanActionStrings(values)
			}
		}
		return cleanActionStrings(core.Split(trimmed, ","))
	default:
		if text := actionStringFromAny(value); text != "" {
			return []string{text}
		}
	}
	return nil
}

func cleanActionStrings(values []string) []string {
	var cleaned []string
	for _, value := range values {
		trimmed := core.Trim(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
