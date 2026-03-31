// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// session := agentic.Session{SessionID: "ses_abc123", AgentType: "codex", Status: "active"}
type Session struct {
	ID             int              `json:"id"`
	SessionID      string           `json:"session_id"`
	Plan           string           `json:"plan,omitempty"`
	PlanSlug       string           `json:"plan_slug,omitempty"`
	AgentType      string           `json:"agent_type"`
	Status         string           `json:"status"`
	ContextSummary map[string]any   `json:"context_summary,omitempty"`
	WorkLog        []map[string]any `json:"work_log,omitempty"`
	Artifacts      []map[string]any `json:"artifacts,omitempty"`
	Handoff        map[string]any   `json:"handoff,omitempty"`
	Summary        string           `json:"summary,omitempty"`
	CreatedAt      string           `json:"created_at,omitempty"`
	UpdatedAt      string           `json:"updated_at,omitempty"`
	EndedAt        string           `json:"ended_at,omitempty"`
}

// input := agentic.SessionStartInput{AgentType: "codex", PlanSlug: "ax-follow-up"}
type SessionStartInput struct {
	PlanSlug  string         `json:"plan_slug,omitempty"`
	AgentType string         `json:"agent_type"`
	Context   map[string]any `json:"context,omitempty"`
}

// input := agentic.SessionGetInput{SessionID: "ses_abc123"}
type SessionGetInput struct {
	SessionID string `json:"session_id"`
}

// input := agentic.SessionListInput{PlanSlug: "ax-follow-up", Status: "active"}
type SessionListInput struct {
	PlanSlug string `json:"plan_slug,omitempty"`
	Status   string `json:"status,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// input := agentic.SessionContinueInput{SessionID: "ses_abc123", AgentType: "codex"}
type SessionContinueInput struct {
	SessionID string           `json:"session_id"`
	AgentType string           `json:"agent_type,omitempty"`
	WorkLog   []map[string]any `json:"work_log,omitempty"`
	Context   map[string]any   `json:"context,omitempty"`
}

// input := agentic.SessionEndInput{SessionID: "ses_abc123", Status: "completed"}
type SessionEndInput struct {
	SessionID string         `json:"session_id"`
	Status    string         `json:"status,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Handoff   map[string]any `json:"handoff,omitempty"`
}

// out := agentic.SessionOutput{Success: true, Session: agentic.Session{SessionID: "ses_abc123"}}
type SessionOutput struct {
	Success bool    `json:"success"`
	Session Session `json:"session"`
}

// out := agentic.SessionListOutput{Success: true, Count: 1, Sessions: []agentic.Session{{SessionID: "ses_abc123"}}}
type SessionListOutput struct {
	Success  bool      `json:"success"`
	Count    int       `json:"count"`
	Sessions []Session `json:"sessions"`
}

// result := c.Action("session.start").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "agent_type", Value: "codex"},
//	core.Option{Key: "plan_slug", Value: "ax-follow-up"},
//
// ))
func (s *PrepSubsystem) handleSessionStart(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sessionStart(ctx, nil, SessionStartInput{
		PlanSlug:  optionStringValue(options, "plan_slug", "plan"),
		AgentType: optionStringValue(options, "agent_type", "agent"),
		Context:   optionAnyMapValue(options, "context"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("session.get").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionGet(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sessionGet(ctx, nil, SessionGetInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("session.list").Run(ctx, core.NewOptions(core.Option{Key: "status", Value: "active"}))
func (s *PrepSubsystem) handleSessionList(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sessionList(ctx, nil, SessionListInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan"),
		Status:   optionStringValue(options, "status"),
		Limit:    optionIntValue(options, "limit"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("session.continue").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionContinue(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sessionContinue(ctx, nil, SessionContinueInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
		AgentType: optionStringValue(options, "agent_type", "agent"),
		WorkLog:   optionAnyMapSliceValue(options, "work_log"),
		Context:   optionAnyMapValue(options, "context"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("session.end").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionEnd(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sessionEnd(ctx, nil, SessionEndInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
		Status:    optionStringValue(options, "status"),
		Summary:   optionStringValue(options, "summary"),
		Handoff:   optionAnyMapValue(options, "handoff"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerSessionTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_start",
		Description: "Start a new agent session for a plan and capture the initial context summary.",
	}, s.sessionStart)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_get",
		Description: "Read a session by session ID, including saved context, work log, and artifacts.",
	}, s.sessionGet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_list",
		Description: "List sessions with optional plan and status filters.",
	}, s.sessionList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_continue",
		Description: "Continue an existing session from its latest saved state.",
	}, s.sessionContinue)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "session_end",
		Description: "End a session with status, summary, and optional handoff notes.",
	}, s.sessionEnd)
}

func (s *PrepSubsystem) sessionStart(ctx context.Context, _ *mcp.CallToolRequest, input SessionStartInput) (*mcp.CallToolResult, SessionOutput, error) {
	if input.AgentType == "" {
		return nil, SessionOutput{}, core.E("sessionStart", "agent_type is required", nil)
	}

	body := map[string]any{
		"agent_type": input.AgentType,
	}
	if input.PlanSlug != "" {
		body["plan_slug"] = input.PlanSlug
	}
	if len(input.Context) > 0 {
		body["context"] = input.Context
	}

	result := s.platformPayload(ctx, "session.start", "POST", "/v1/sessions", body)
	if !result.OK {
		return nil, SessionOutput{}, resultErrorValue("session.start", result)
	}

	return nil, SessionOutput{
		Success: true,
		Session: parseSession(sessionDataMap(result.Value.(map[string]any))),
	}, nil
}

func (s *PrepSubsystem) sessionGet(ctx context.Context, _ *mcp.CallToolRequest, input SessionGetInput) (*mcp.CallToolResult, SessionOutput, error) {
	if input.SessionID == "" {
		return nil, SessionOutput{}, core.E("sessionGet", "session_id is required", nil)
	}

	path := core.Concat("/v1/sessions/", input.SessionID)
	result := s.platformPayload(ctx, "session.get", "GET", path, nil)
	if !result.OK {
		return nil, SessionOutput{}, resultErrorValue("session.get", result)
	}

	return nil, SessionOutput{
		Success: true,
		Session: parseSession(sessionDataMap(result.Value.(map[string]any))),
	}, nil
}

func (s *PrepSubsystem) sessionList(ctx context.Context, _ *mcp.CallToolRequest, input SessionListInput) (*mcp.CallToolResult, SessionListOutput, error) {
	path := "/v1/sessions"
	path = appendQueryParam(path, "plan_slug", input.PlanSlug)
	path = appendQueryParam(path, "status", input.Status)
	if input.Limit > 0 {
		path = appendQueryParam(path, "limit", core.Sprint(input.Limit))
	}

	result := s.platformPayload(ctx, "session.list", "GET", path, nil)
	if !result.OK {
		return nil, SessionListOutput{}, resultErrorValue("session.list", result)
	}

	return nil, parseSessionListOutput(result.Value.(map[string]any)), nil
}

func (s *PrepSubsystem) sessionContinue(ctx context.Context, _ *mcp.CallToolRequest, input SessionContinueInput) (*mcp.CallToolResult, SessionOutput, error) {
	if input.SessionID == "" {
		return nil, SessionOutput{}, core.E("sessionContinue", "session_id is required", nil)
	}

	body := map[string]any{}
	if input.AgentType != "" {
		body["agent_type"] = input.AgentType
	}
	if len(input.WorkLog) > 0 {
		body["work_log"] = input.WorkLog
	}
	if len(input.Context) > 0 {
		body["context"] = input.Context
	}

	path := core.Concat("/v1/sessions/", input.SessionID, "/continue")
	result := s.platformPayload(ctx, "session.continue", "POST", path, body)
	if !result.OK {
		return nil, SessionOutput{}, resultErrorValue("session.continue", result)
	}

	return nil, SessionOutput{
		Success: true,
		Session: parseSession(sessionDataMap(result.Value.(map[string]any))),
	}, nil
}

func (s *PrepSubsystem) sessionEnd(ctx context.Context, _ *mcp.CallToolRequest, input SessionEndInput) (*mcp.CallToolResult, SessionOutput, error) {
	if input.SessionID == "" {
		return nil, SessionOutput{}, core.E("sessionEnd", "session_id is required", nil)
	}

	body := map[string]any{}
	if input.Status != "" {
		body["status"] = input.Status
	}
	if input.Summary != "" {
		body["summary"] = input.Summary
	}
	if len(input.Handoff) > 0 {
		body["handoff"] = input.Handoff
	}

	path := core.Concat("/v1/sessions/", input.SessionID, "/end")
	result := s.platformPayload(ctx, "session.end", "POST", path, body)
	if !result.OK {
		return nil, SessionOutput{}, resultErrorValue("session.end", result)
	}

	return nil, SessionOutput{
		Success: true,
		Session: parseSession(sessionDataMap(result.Value.(map[string]any))),
	}, nil
}

func sessionDataMap(payload map[string]any) map[string]any {
	data := payloadDataMap(payload)
	if len(data) > 0 {
		return data
	}
	return payload
}

func parseSession(values map[string]any) Session {
	planSlug := stringValue(values["plan_slug"])
	if planSlug == "" {
		planSlug = stringValue(values["plan"])
	}

	return Session{
		ID:             intValue(values["id"]),
		SessionID:      stringValue(values["session_id"]),
		Plan:           stringValue(values["plan"]),
		PlanSlug:       planSlug,
		AgentType:      stringValue(values["agent_type"]),
		Status:         stringValue(values["status"]),
		ContextSummary: anyMapValue(values["context_summary"]),
		WorkLog:        anyMapSliceValue(values["work_log"]),
		Artifacts:      anyMapSliceValue(values["artifacts"]),
		Handoff:        anyMapValue(values["handoff"]),
		Summary:        stringValue(values["summary"]),
		CreatedAt:      stringValue(values["created_at"]),
		UpdatedAt:      stringValue(values["updated_at"]),
		EndedAt:        stringValue(values["ended_at"]),
	}
}

func parseSessionListOutput(payload map[string]any) SessionListOutput {
	sessionData := payloadDataSlice(payload)
	sessions := make([]Session, 0, len(sessionData))
	for _, values := range sessionData {
		sessions = append(sessions, parseSession(values))
	}

	count := intValue(payload["count"])
	if count == 0 {
		count = intValue(payload["total"])
	}
	if count == 0 {
		count = len(sessions)
	}

	return SessionListOutput{
		Success:  true,
		Count:    count,
		Sessions: sessions,
	}
}

func resultErrorValue(action string, result core.Result) error {
	if err, ok := result.Value.(error); ok && err != nil {
		return err
	}

	message := stringValue(result.Value)
	if message != "" {
		return core.E(action, message, nil)
	}

	return core.E(action, "request failed", nil)
}
