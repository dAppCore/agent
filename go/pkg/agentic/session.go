// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// session := agentic.Session{SessionID: "ses_abc123", AgentType: "codex", Status: "active"}
type Session struct {
	ID             int              `json:"id"`
	WorkspaceID    int              `json:"workspace_id,omitempty"`
	AgentPlanID    int              `json:"agent_plan_id,omitempty"`
	AgentAPIKeyID  int              `json:"agent_api_key_id,omitempty"`
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

// session := agentic.AgentSession{SessionID: "ses_abc123", AgentType: "codex", Status: "active"}
type AgentSession = Session

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
	PlanSlug  string `json:"plan_slug,omitempty"`
	AgentType string `json:"agent_type,omitempty"`
	Status    string `json:"status,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

// input := agentic.SessionContinueInput{SessionID: "ses_abc123", AgentType: "codex"}
type SessionContinueInput struct {
	SessionID string           `json:"session_id"`
	AgentType string           `json:"agent_type,omitempty"`
	WorkLog   []map[string]any `json:"work_log,omitempty"`
	Context   map[string]any   `json:"context,omitempty"`
}

// input := agentic.SessionEndInput{SessionID: "ses_abc123", Status: "completed", HandoffNotes: map[string]any{"summary": "Ready for review"}}
type SessionEndInput struct {
	SessionID    string         `json:"session_id"`
	Status       string         `json:"status,omitempty"`
	Summary      string         `json:"summary,omitempty"`
	Handoff      map[string]any `json:"handoff,omitempty"`
	HandoffNotes map[string]any `json:"handoff_notes,omitempty"`
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

// input := agentic.SessionLogInput{SessionID: "ses_abc123", Message: "Checked build", Type: "checkpoint"}
type SessionLogInput struct {
	SessionID string         `json:"session_id"`
	Message   string         `json:"message"`
	Type      string         `json:"type,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// out := agentic.SessionLogOutput{Success: true, Logged: "Checked build"}
type SessionLogOutput struct {
	Success bool   `json:"success"`
	Logged  string `json:"logged"`
}

// input := agentic.SessionArtifactInput{SessionID: "ses_abc123", Path: "pkg/agentic/session.go", Action: "modified"}
type SessionArtifactInput struct {
	SessionID   string         `json:"session_id"`
	Path        string         "json:\"path\""
	Action      string         `json:"action"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Description string         `json:"description,omitempty"`
}

// out := agentic.SessionArtifactOutput{Success: true, Artifact: "pkg/agentic/session.go"}
type SessionArtifactOutput struct {
	Success  bool   `json:"success"`
	Artifact string `json:"artifact"`
}

// input := agentic.SessionHandoffInput{SessionID: "ses_abc123", Summary: "Ready for review"}
type SessionHandoffInput struct {
	SessionID      string         `json:"session_id"`
	Summary        string         `json:"summary"`
	NextSteps      []string       `json:"next_steps,omitempty"`
	Blockers       []string       `json:"blockers,omitempty"`
	ContextForNext map[string]any `json:"context_for_next,omitempty"`
}

// out := agentic.SessionHandoffOutput{Success: true}
type SessionHandoffOutput struct {
	Success        bool           `json:"success"`
	HandoffContext map[string]any `json:"handoff_context"`
}

// input := agentic.SessionResumeInput{SessionID: "ses_abc123"}
type SessionResumeInput struct {
	SessionID string `json:"session_id"`
}

// out := agentic.SessionResumeOutput{Success: true, Session: agentic.Session{SessionID: "ses_abc123"}}
type SessionResumeOutput struct {
	Success        bool             `json:"success"`
	Session        Session          `json:"session"`
	HandoffContext map[string]any   `json:"handoff_context,omitempty"`
	RecentActions  []map[string]any `json:"recent_actions,omitempty"`
	Artifacts      []map[string]any `json:"artifacts,omitempty"`
}

// input := agentic.SessionReplayInput{SessionID: "ses_abc123"}
type SessionReplayInput struct {
	SessionID string `json:"session_id"`
}

// out := agentic.SessionReplayOutput{Success: true}
type SessionReplayOutput struct {
	Success       bool           `json:"success"`
	ReplayContext map[string]any `json:"replay_context"`
}

// result := c.Action("session.start").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "agent_type", Value: "codex"},
//	core.Option{Key: "plan_slug", Value: "ax-follow-up"},
//
// ))
func (s *PrepSubsystem) handleSessionStart(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionOutput]("session.start", "invalid session start output", s.sessionStart(ctx, SessionStartInput{
		PlanSlug:  optionStringValue(options, "plan_slug", "plan"),
		AgentType: optionStringValue(options, "agent_type", "agent"),
		Context:   optionAnyMapValue(options, "context"),
	}))
}

// result := c.Action("session.get").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionGet(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionOutput]("session.get", "invalid session get output", s.sessionGet(ctx, SessionGetInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
	}))
}

// result := c.Action("session.list").Run(ctx, core.NewOptions(core.Option{Key: "status", Value: "active"}))
func (s *PrepSubsystem) handleSessionList(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionListOutput]("session.list", "invalid session list output", s.sessionList(ctx, SessionListInput{
		PlanSlug:  optionStringValue(options, "plan_slug", "plan"),
		AgentType: optionStringValue(options, "agent_type", "agent"),
		Status:    optionStringValue(options, "status"),
		Limit:     optionIntValue(options, "limit"),
	}))
}

// result := c.Action("session.continue").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionContinue(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionOutput]("session.continue", "invalid session continue output", s.sessionContinue(ctx, SessionContinueInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
		AgentType: optionStringValue(options, "agent_type", "agent"),
		WorkLog:   optionAnyMapSliceValue(options, "work_log"),
		Context:   optionAnyMapValue(options, "context"),
	}))
}

// result := c.Action("session.end").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionEnd(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionOutput]("session.end", "invalid session end output", s.sessionEnd(ctx, SessionEndInput{
		SessionID:    optionStringValue(options, "session_id", "id", "_arg"),
		Status:       optionStringValue(options, "status"),
		Summary:      optionStringValue(options, "summary"),
		Handoff:      optionAnyMapValue(options, "handoff"),
		HandoffNotes: optionAnyMapValue(options, "handoff_notes", "handoff-notes"),
	}))
}

// result := c.Action("session.log").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "session_id", Value: "ses_abc123"},
//	core.Option{Key: "message", Value: "Checked build"},
//
// ))
func (s *PrepSubsystem) handleSessionLog(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionLogOutput]("session.log", "invalid session log output", s.sessionLog(ctx, SessionLogInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
		Message:   optionStringValue(options, "message"),
		Type:      optionStringValue(options, "type"),
		Data:      optionAnyMapValue(options, "data"),
	}))
}

// result := c.Action("session.artifact").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "session_id", Value: "ses_abc123"},
//	core.Option{Key: `path`, Value: "pkg/agentic/session.go"},
//
// ))
func (s *PrepSubsystem) handleSessionArtifact(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionArtifactOutput]("session.artifact", "invalid session artifact output", s.sessionArtifact(ctx, SessionArtifactInput{
		SessionID:   optionStringValue(options, "session_id", "id", "_arg"),
		Path:        optionStringValue(options, `path`),
		Action:      optionStringValue(options, "action"),
		Metadata:    optionAnyMapValue(options, "metadata"),
		Description: optionStringValue(options, "description"),
	}))
}

// result := c.Action("session.handoff").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "session_id", Value: "ses_abc123"},
//	core.Option{Key: "summary", Value: "Ready for review"},
//
// ))
func (s *PrepSubsystem) handleSessionHandoff(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionHandoffOutput]("session.handoff", "invalid session handoff output", s.sessionHandoff(ctx, SessionHandoffInput{
		SessionID:      optionStringValue(options, "session_id", "id", "_arg"),
		Summary:        optionStringValue(options, "summary"),
		NextSteps:      optionStringSliceValue(options, "next_steps", "next-steps"),
		Blockers:       optionStringSliceValue(options, "blockers"),
		ContextForNext: optionAnyMapValue(options, "context_for_next", "context-for-next"),
	}))
}

// result := c.Action("session.resume").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionResume(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionResumeOutput]("session.resume", "invalid session resume output", s.sessionResume(ctx, SessionResumeInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
	}))
}

// result := c.Action("session.replay").Run(ctx, core.NewOptions(core.Option{Key: "session_id", Value: "ses_abc123"}))
func (s *PrepSubsystem) handleSessionReplay(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[SessionReplayOutput]("session.replay", "invalid session replay output", s.sessionReplay(ctx, SessionReplayInput{
		SessionID: optionStringValue(options, "session_id", "id", "_arg"),
	}))
}

func (s *PrepSubsystem) registerSessionTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_start",
		Description: "Start a new agent session for a plan and capture the initial context summary.",
	}, toolHandlerFor[SessionStartInput, SessionOutput]("session.start", "invalid session start output", s.sessionStart))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_start",
		Description: "Start a new agent session for a plan and capture the initial context summary.",
	}, toolHandlerFor[SessionStartInput, SessionOutput]("session.start", "invalid session start output", s.sessionStart))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_get",
		Description: "Read a session by session ID, including saved context, work log, and artifacts.",
	}, toolHandlerFor[SessionGetInput, SessionOutput]("session.get", "invalid session get output", s.sessionGet))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_get",
		Description: "Read a session by session ID, including saved context, work log, and artifacts.",
	}, toolHandlerFor[SessionGetInput, SessionOutput]("session.get", "invalid session get output", s.sessionGet))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_list",
		Description: "List sessions with optional plan and status filters.",
	}, toolHandlerFor[SessionListInput, SessionListOutput]("session.list", "invalid session list output", s.sessionList))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_list",
		Description: "List sessions with optional plan and status filters.",
	}, toolHandlerFor[SessionListInput, SessionListOutput]("session.list", "invalid session list output", s.sessionList))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_continue",
		Description: "Continue an existing session from its latest saved state.",
	}, toolHandlerFor[SessionContinueInput, SessionOutput]("session.continue", "invalid session continue output", s.sessionContinue))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_continue",
		Description: "Continue an existing session from its latest saved state.",
	}, toolHandlerFor[SessionContinueInput, SessionOutput]("session.continue", "invalid session continue output", s.sessionContinue))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_end",
		Description: "End a session with status, summary, and optional handoff notes.",
	}, toolHandlerFor[SessionEndInput, SessionOutput]("session.end", "invalid session end output", s.sessionEnd))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_end",
		Description: "End a session with status, summary, and optional handoff notes.",
	}, toolHandlerFor[SessionEndInput, SessionOutput]("session.end", "invalid session end output", s.sessionEnd))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_complete",
		Description: "Mark a session completed with status, summary, and optional handoff notes.",
	}, toolHandlerFor[SessionEndInput, SessionOutput]("session.end", "invalid session end output", s.sessionEnd))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_complete",
		Description: "Mark a session completed with status, summary, and optional handoff notes.",
	}, toolHandlerFor[SessionEndInput, SessionOutput]("session.end", "invalid session end output", s.sessionEnd))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_log",
		Description: "Add a typed work log entry to a stored session.",
	}, toolHandlerFor[SessionLogInput, SessionLogOutput]("session.log", "invalid session log output", s.sessionLog))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_log",
		Description: "Add a typed work log entry to a stored session.",
	}, toolHandlerFor[SessionLogInput, SessionLogOutput]("session.log", "invalid session log output", s.sessionLog))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_artifact",
		Description: "Record a created, modified, deleted, or reviewed artifact for a stored session.",
	}, toolHandlerFor[SessionArtifactInput, SessionArtifactOutput]("session.artifact", "invalid session artifact output", s.sessionArtifact))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_artifact",
		Description: "Record a created, modified, deleted, or reviewed artifact for a stored session.",
	}, toolHandlerFor[SessionArtifactInput, SessionArtifactOutput]("session.artifact", "invalid session artifact output", s.sessionArtifact))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_handoff",
		Description: "Prepare a stored session for handoff and mark it handed_off with summary, blockers, and next-step context.",
	}, toolHandlerFor[SessionHandoffInput, SessionHandoffOutput]("session.handoff", "invalid session handoff output", s.sessionHandoff))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_handoff",
		Description: "Prepare a stored session for handoff and mark it handed_off with summary, blockers, and next-step context.",
	}, toolHandlerFor[SessionHandoffInput, SessionHandoffOutput]("session.handoff", "invalid session handoff output", s.sessionHandoff))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_resume",
		Description: "Resume a paused or handed-off stored session and return handoff context.",
	}, toolHandlerFor[SessionResumeInput, SessionResumeOutput]("session.resume", "invalid session resume output", s.sessionResume))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_resume",
		Description: "Resume a paused or handed-off stored session and return handoff context.",
	}, toolHandlerFor[SessionResumeInput, SessionResumeOutput]("session.resume", "invalid session resume output", s.sessionResume))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "session_replay",
		Description: "Build replay context for a stored session from its work log, checkpoints, errors, and artifacts.",
	}, toolHandlerFor[SessionReplayInput, SessionReplayOutput]("session.replay", "invalid session replay output", s.sessionReplay))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_session_replay",
		Description: "Build replay context for a stored session from its work log, checkpoints, errors, and artifacts.",
	}, toolHandlerFor[SessionReplayInput, SessionReplayOutput]("session.replay", "invalid session replay output", s.sessionReplay))
}

func (s *PrepSubsystem) sessionStart(ctx context.Context, input SessionStartInput) core.Result {
	if input.AgentType == "" {
		return core.Fail(core.E("sessionStart", "agent_type is required", nil))
	}
	normalisedAgentType, ok := normaliseSessionAgentType(input.AgentType)
	if !ok {
		return core.Fail(core.E("sessionStart", "agent_type must be opus, sonnet, haiku, or claude:opus|claude:sonnet|claude:haiku", nil))
	}

	body := map[string]any{
		"agent_type": normalisedAgentType,
	}
	if input.PlanSlug != "" {
		body["plan_slug"] = input.PlanSlug
	}
	if len(input.Context) > 0 {
		body["context"] = input.Context
	}

	result := s.platformPayload(ctx, "session.start", "POST", "/v1/sessions", body)
	if !result.OK {
		return failureResult("session.start", "request failed", result)
	}

	return core.Ok(SessionOutput{
		Success: true,
		Session: s.storeSession(sessionFromInput(parseSession(sessionDataMap(result.Value.(map[string]any))), input)),
	})
}

func (s *PrepSubsystem) sessionGet(ctx context.Context, input SessionGetInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionGet", "session_id is required", nil))
	}

	path := core.Concat("/v1/sessions/", input.SessionID)
	result := s.platformPayload(ctx, "session.get", "GET", path, nil)
	if !result.OK {
		return failureResult("session.get", "request failed", result)
	}

	return core.Ok(SessionOutput{
		Success: true,
		Session: s.storeSession(parseSession(sessionDataMap(result.Value.(map[string]any)))),
	})
}

func (s *PrepSubsystem) sessionList(ctx context.Context, input SessionListInput) core.Result {
	path := "/v1/sessions"
	path = appendQueryParam(path, "plan_slug", input.PlanSlug)
	if agentType, ok := normaliseSessionAgentType(input.AgentType); ok {
		path = appendQueryParam(path, "agent_type", agentType)
	} else {
		path = appendQueryParam(path, "agent_type", input.AgentType)
	}
	path = appendQueryParam(path, "status", input.Status)
	if input.Limit > 0 {
		path = appendQueryParam(path, "limit", core.Sprint(input.Limit))
	}

	result := s.platformPayload(ctx, "session.list", "GET", path, nil)
	if !result.OK {
		return failureResult("session.list", "request failed", result)
	}

	output := parseSessionListOutput(result.Value.(map[string]any))
	for i := range output.Sessions {
		output.Sessions[i] = s.storeSession(output.Sessions[i])
	}

	return core.Ok(output)
}

func (s *PrepSubsystem) sessionContinue(ctx context.Context, input SessionContinueInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionContinue", "session_id is required", nil))
	}

	body := map[string]any{}
	if agentType, ok := normaliseSessionAgentType(input.AgentType); ok {
		body["agent_type"] = agentType
	} else if input.AgentType != "" {
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
		return failureResult("session.continue", "request failed", result)
	}

	return core.Ok(SessionOutput{
		Success: true,
		Session: s.storeSession(parseSession(sessionDataMap(result.Value.(map[string]any)))),
	})
}

func (s *PrepSubsystem) sessionEnd(ctx context.Context, input SessionEndInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionEnd", "session_id is required", nil))
	}

	body := map[string]any{}
	if input.Status != "" {
		body["status"] = input.Status
	}
	if input.Summary != "" {
		body["summary"] = input.Summary
	}
	handoff := mergeSessionHandoff(input.Handoff, input.HandoffNotes)
	if len(handoff) > 0 {
		body["handoff"] = handoff
		body["handoff_notes"] = handoff
	}

	path := core.Concat("/v1/sessions/", input.SessionID, "/end")
	result := s.platformPayload(ctx, "session.end", "POST", path, body)
	if !result.OK {
		return failureResult("session.end", "request failed", result)
	}

	return core.Ok(SessionOutput{
		Success: true,
		Session: func() Session {
			session := s.storeSession(sessionEndFromInput(parseSession(sessionDataMap(result.Value.(map[string]any))), input))
			s.persistSessionHandoffMemory(ctx, session)
			return session
		}(),
	})
}

func (s *PrepSubsystem) sessionLog(ctx context.Context, input SessionLogInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionLog", "session_id is required", nil))
	}
	if input.Message == "" {
		return core.Fail(core.E("sessionLog", "message is required", nil))
	}

	session, err := loadSession(s, ctx, input.SessionID)
	if err != nil {
		return core.Fail(err)
	}

	entryType := input.Type
	if entryType == "" {
		entryType = "info"
	}

	entry := map[string]any{
		"message":   input.Message,
		"type":      entryType,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if len(input.Data) > 0 {
		entry["data"] = input.Data
	}

	session.WorkLog = append(session.WorkLog, entry)
	session.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := writeSessionCache(&session); err != nil {
		return core.Fail(err)
	}

	return core.Ok(SessionLogOutput{
		Success: true,
		Logged:  input.Message,
	})
}

func (s *PrepSubsystem) sessionArtifact(ctx context.Context, input SessionArtifactInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionArtifact", "session_id is required", nil))
	}
	if input.Path == "" {
		return core.Fail(core.E("sessionArtifact", "path is required", nil))
	}
	if input.Action == "" {
		return core.Fail(core.E("sessionArtifact", "action is required", nil))
	}

	session, err := loadSession(s, ctx, input.SessionID)
	if err != nil {
		return core.Fail(err)
	}

	artifact := map[string]any{
		`path`:      input.Path,
		"action":    input.Action,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	if input.Description != "" {
		metadata["description"] = input.Description
	}
	if len(metadata) > 0 {
		artifact["metadata"] = metadata
	}

	session.Artifacts = append(session.Artifacts, artifact)
	session.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := writeSessionCache(&session); err != nil {
		return core.Fail(err)
	}

	return core.Ok(SessionArtifactOutput{
		Success:  true,
		Artifact: input.Path,
	})
}

func (s *PrepSubsystem) sessionHandoff(ctx context.Context, input SessionHandoffInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionHandoff", "session_id is required", nil))
	}
	if input.Summary == "" {
		return core.Fail(core.E("sessionHandoff", "summary is required", nil))
	}

	session, err := loadSession(s, ctx, input.SessionID)
	if err != nil {
		return core.Fail(err)
	}

	session.Handoff = map[string]any{
		"summary":          input.Summary,
		"next_steps":       cleanStrings(input.NextSteps),
		"blockers":         cleanStrings(input.Blockers),
		"context_for_next": input.ContextForNext,
	}
	session.Status = "handed_off"
	session.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := writeSessionCache(&session); err != nil {
		return core.Fail(err)
	}
	s.persistSessionHandoffMemory(ctx, session)

	return core.Ok(SessionHandoffOutput{
		Success:        true,
		HandoffContext: sessionHandoffContext(session),
	})
}

func (s *PrepSubsystem) sessionResume(ctx context.Context, input SessionResumeInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionResume", "session_id is required", nil))
	}

	session, err := loadSession(s, ctx, input.SessionID)
	if err != nil {
		return core.Fail(err)
	}

	if session.Status == "" || session.Status == "paused" || session.Status == "handed_off" {
		session.Status = "active"
	}
	session.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := writeSessionCache(&session); err != nil {
		return core.Fail(err)
	}

	return core.Ok(SessionResumeOutput{
		Success:        true,
		Session:        session,
		HandoffContext: sessionHandoffContext(session),
		RecentActions:  recentSessionActions(session.WorkLog, 20),
		Artifacts:      session.Artifacts,
	})
}

func (s *PrepSubsystem) sessionReplay(ctx context.Context, input SessionReplayInput) core.Result {
	if input.SessionID == "" {
		return core.Fail(core.E("sessionReplay", "session_id is required", nil))
	}

	session, err := loadSession(s, ctx, input.SessionID)
	if err != nil {
		return core.Fail(err)
	}

	return core.Ok(SessionReplayOutput{
		Success:       true,
		ReplayContext: sessionReplayContext(session),
	})
}

func sessionDataMap(payload map[string]any) map[string]any {
	data := payloadResourceMap(payload, "session")
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
		WorkspaceID:    intValue(values["workspace_id"]),
		AgentPlanID:    intValue(values["agent_plan_id"]),
		AgentAPIKeyID:  intValue(values["agent_api_key_id"]),
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

var loadSession = func(s *PrepSubsystem, ctx context.Context, sessionID string) (Session, error) {
	if cached, err := readSessionCache(sessionID); err == nil && cached != nil {
		return *cached, nil
	}

	result := s.sessionGet(ctx, SessionGetInput{SessionID: sessionID})
	typed := typedResultValue[SessionOutput]("loadSession", "invalid session get output", result)
	if !typed.OK {
		return Session{}, resultErrorValue("loadSession", typed)
	}

	return typed.Value.(SessionOutput).Session, nil
}

func (s *PrepSubsystem) storeSession(session Session) Session {
	merged, err := mergeSessionCache(session)
	if err != nil {
		return session
	}
	if err := writeSessionCache(&merged); err != nil {
		return merged
	}
	return merged
}

func sessionFromInput(session Session, input SessionStartInput) Session {
	if session.PlanSlug == "" {
		session.PlanSlug = input.PlanSlug
	}
	if session.Plan == "" {
		session.Plan = input.PlanSlug
	}
	if session.AgentType == "" {
		session.AgentType = input.AgentType
	}
	if len(session.ContextSummary) == 0 && len(input.Context) > 0 {
		session.ContextSummary = input.Context
	}
	return session
}

func sessionEndFromInput(session Session, input SessionEndInput) Session {
	if session.SessionID == "" {
		session.SessionID = input.SessionID
	}
	if session.Status == "" {
		session.Status = input.Status
	}
	if session.Summary == "" {
		session.Summary = input.Summary
	}
	if len(session.Handoff) == 0 && len(input.Handoff) > 0 {
		session.Handoff = mergeSessionHandoff(input.Handoff, input.HandoffNotes)
	}
	if len(session.Handoff) == 0 && len(input.HandoffNotes) > 0 {
		session.Handoff = mergeSessionHandoff(input.HandoffNotes, nil)
	}
	if session.Status == "completed" || session.Status == "failed" || session.Status == "handed_off" {
		if session.EndedAt == "" {
			session.EndedAt = time.Now().Format(time.RFC3339)
		}
	}
	return session
}

func (s *PrepSubsystem) persistSessionHandoffMemory(ctx context.Context, session Session) {
	if s == nil || s.brainKey == "" || session.SessionID == "" {
		return
	}
	if len(session.Handoff) == 0 {
		return
	}

	summary := stringValue(session.Handoff["summary"])
	nextSteps := stringSliceValue(session.Handoff["next_steps"])
	blockers := stringSliceValue(session.Handoff["blockers"])
	contextForNext := anyMapValue(session.Handoff["context_for_next"])

	if summary == "" && len(nextSteps) == 0 && len(blockers) == 0 && len(contextForNext) == 0 {
		return
	}

	body := map[string]any{
		"content":    sessionHandoffMemoryContent(session, summary, nextSteps, blockers, contextForNext),
		"agent_id":   session.AgentType,
		"type":       "observation",
		"tags":       sessionHandoffMemoryTags(session),
		"confidence": 0.7,
	}
	if project := sessionBrainProject(session, contextForNext); project != "" {
		body["project"] = project
	}

	result := s.brainCall(ctx, "POST", "/v1/brain/remember", session.AgentType, body)
	if !result.OK {
		core.Warn("session handoff memory persist failed", "session_id", session.SessionID, "reason", result.Value)
	}
}

func sessionHandoffMemoryContent(session Session, summary string, nextSteps, blockers []string, contextForNext map[string]any) string {
	builder := core.NewBuilder()
	builder.WriteString(core.Concat("Session handoff: ", session.SessionID, "\n"))
	if session.PlanSlug != "" {
		builder.WriteString(core.Concat("Plan: ", session.PlanSlug, "\n"))
	}
	if session.AgentType != "" {
		builder.WriteString(core.Concat("Agent: ", session.AgentType, "\n"))
	}
	if session.Status != "" {
		builder.WriteString(core.Concat("Status: ", session.Status, "\n"))
	}

	if summary != "" {
		builder.WriteString("\nSummary:\n")
		builder.WriteString(summary)
		builder.WriteString("\n")
	}
	if len(nextSteps) > 0 {
		builder.WriteString("\nNext steps:\n")
		for _, nextStep := range nextSteps {
			builder.WriteString(core.Concat("- ", nextStep, "\n"))
		}
	}
	if len(blockers) > 0 {
		builder.WriteString("\nBlockers:\n")
		for _, blocker := range blockers {
			builder.WriteString(core.Concat("- ", blocker, "\n"))
		}
	}
	if len(contextForNext) > 0 {
		builder.WriteString("\nContext for next:\n")
		builder.WriteString(core.JSONMarshalString(contextForNext))
		builder.WriteString("\n")
	}

	return core.Trim(builder.String())
}

func sessionHandoffMemoryTags(session Session) []string {
	return cleanStrings([]string{"session", "handoff", session.AgentType, sessionPlanSlug(session)})
}

func sessionBrainProject(session Session, contextForNext map[string]any) string {
	if project := stringValue(session.ContextSummary["repo"]); project != "" {
		return project
	}
	if project := stringValue(contextForNext["repo"]); project != "" {
		return project
	}
	return ""
}

func mergeSessionHandoff(primary, fallback map[string]any) map[string]any {
	if len(primary) == 0 && len(fallback) == 0 {
		return nil
	}

	merged := map[string]any{}
	for key, value := range primary {
		if value != nil {
			merged[key] = value
		}
	}
	for key, value := range fallback {
		if value != nil {
			merged[key] = value
		}
	}
	return merged
}

func sessionCacheRoot() string {
	return core.JoinPath(CoreRoot(), "sessions")
}

func sessionCachePath(sessionID string) string {
	return core.JoinPath(sessionCacheRoot(), core.Concat(pathKey(sessionID), ".json"))
}

var readSessionCache = func(sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, core.E("readSessionCache", "session_id is required", nil)
	}

	result := fs.Read(sessionCachePath(sessionID))
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			return nil, core.E("readSessionCache", core.Concat("session not found: ", sessionID), nil)
		}
		return nil, core.E("readSessionCache", core.Concat("session not found: ", sessionID), err)
	}

	var session Session
	if parseResult := core.JSONUnmarshalString(result.Value.(string), &session); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return nil, core.E("readSessionCache", "failed to parse session cache", err)
	}

	return &session, nil
}

var writeSessionCache = func(session *Session) error {
	if session == nil {
		return core.E("writeSessionCache", "session is required", nil)
	}
	if session.SessionID == "" {
		return core.E("writeSessionCache", "session_id is required", nil)
	}

	now := time.Now().Format(time.RFC3339)
	if session.CreatedAt == "" {
		session.CreatedAt = now
	}
	if session.UpdatedAt == "" {
		session.UpdatedAt = now
	}

	if ensureDirResult := fs.EnsureDir(sessionCacheRoot()); !ensureDirResult.OK {
		err, _ := ensureDirResult.Value.(error)
		return core.E("writeSessionCache", "failed to create session cache directory", err)
	}

	if writeResult := fs.WriteAtomic(sessionCachePath(session.SessionID), core.JSONMarshalString(session)); !writeResult.OK {
		err, _ := writeResult.Value.(error)
		return core.E("writeSessionCache", "failed to write session cache", err)
	}

	return nil
}

var mergeSessionCache = func(session Session) (Session, error) {
	existing, err := readSessionCache(session.SessionID)
	if err == nil && existing != nil {
		if session.ID == 0 {
			session.ID = existing.ID
		}
		if session.WorkspaceID == 0 {
			session.WorkspaceID = existing.WorkspaceID
		}
		if session.AgentPlanID == 0 {
			session.AgentPlanID = existing.AgentPlanID
		}
		if session.AgentAPIKeyID == 0 {
			session.AgentAPIKeyID = existing.AgentAPIKeyID
		}
		if session.Plan == "" {
			session.Plan = existing.Plan
		}
		if session.PlanSlug == "" {
			session.PlanSlug = existing.PlanSlug
		}
		if session.AgentType == "" {
			session.AgentType = existing.AgentType
		}
		if session.Status == "" {
			session.Status = existing.Status
		}
		if len(session.ContextSummary) == 0 {
			session.ContextSummary = existing.ContextSummary
		}
		if len(session.WorkLog) == 0 {
			session.WorkLog = existing.WorkLog
		}
		if len(session.Artifacts) == 0 {
			session.Artifacts = existing.Artifacts
		}
		if len(session.Handoff) == 0 {
			session.Handoff = existing.Handoff
		}
		if session.Summary == "" {
			session.Summary = existing.Summary
		}
		if session.CreatedAt == "" {
			session.CreatedAt = existing.CreatedAt
		}
		if session.EndedAt == "" {
			session.EndedAt = existing.EndedAt
		}
	}

	if session.SessionID == "" {
		return session, core.E("mergeSessionCache", "session_id is required", nil)
	}

	if session.CreatedAt == "" {
		session.CreatedAt = time.Now().Format(time.RFC3339)
	}
	session.UpdatedAt = time.Now().Format(time.RFC3339)

	return session, nil
}

func sessionHandoffContext(session Session) map[string]any {
	context := map[string]any{
		"session_id":      session.SessionID,
		"agent_type":      session.AgentType,
		"context_summary": session.ContextSummary,
		"recent_actions":  recentSessionActions(session.WorkLog, 20),
		"artifacts":       session.Artifacts,
		"handoff_notes":   session.Handoff,
	}
	if session.PlanSlug != "" || session.Plan != "" {
		context["plan"] = map[string]any{
			"slug": sessionPlanSlug(session),
		}
	}
	if session.CreatedAt != "" {
		context["started_at"] = session.CreatedAt
	}
	if session.UpdatedAt != "" {
		context["last_active_at"] = session.UpdatedAt
	}
	return context
}

func sessionReplayContext(session Session) map[string]any {
	checkpoints := filterSessionEntries(session.WorkLog, "checkpoint")
	decisions := filterSessionEntries(session.WorkLog, "decision")
	errors := filterSessionEntries(session.WorkLog, "error")
	lastCheckpoint := map[string]any(nil)
	if len(checkpoints) > 0 {
		lastCheckpoint = checkpoints[len(checkpoints)-1]
	}

	return map[string]any{
		"session_id":       session.SessionID,
		"status":           session.Status,
		"agent_type":       session.AgentType,
		"plan":             map[string]any{"slug": sessionPlanSlug(session)},
		"started_at":       session.CreatedAt,
		"last_active_at":   session.UpdatedAt,
		"context_summary":  session.ContextSummary,
		"progress_summary": sessionProgressSummary(session.WorkLog),
		"work_log":         session.WorkLog,
		"last_checkpoint":  lastCheckpoint,
		"checkpoints":      checkpoints,
		"decisions":        decisions,
		`errors`:           errors,
		"work_log_by_type": map[string]any{
			"checkpoint": checkpoints,
			"decision":   decisions,
			"error":      errors,
		},
		"artifacts": session.Artifacts,
		"artifacts_by_action": map[string]any{
			"created":  filterArtifactsByAction(session.Artifacts, "created"),
			"modified": filterArtifactsByAction(session.Artifacts, "modified"),
			"deleted":  filterArtifactsByAction(session.Artifacts, "deleted"),
			"reviewed": filterArtifactsByAction(session.Artifacts, "reviewed"),
		},
		"recent_actions": recentSessionActions(session.WorkLog, 20),
		"total_actions":  len(session.WorkLog),
		"handoff_notes":  session.Handoff,
		"final_summary":  session.Summary,
	}
}

func filterSessionEntries(entries []map[string]any, entryType string) []map[string]any {
	filtered := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if stringValue(entry["type"]) == entryType {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func recentSessionActions(entries []map[string]any, limit int) []map[string]any {
	if len(entries) == 0 || limit <= 0 {
		return nil
	}

	recent := make([]map[string]any, 0, limit)
	for i := len(entries) - 1; i >= 0 && len(recent) < limit; i-- {
		recent = append(recent, entries[i])
	}
	return recent
}

func sessionProgressSummary(entries []map[string]any) map[string]any {
	if len(entries) == 0 {
		return map[string]any{
			"completed_steps": 0,
			"last_action":     nil,
			"summary":         "No work recorded",
		}
	}

	lastEntry := entries[len(entries)-1]
	checkpointCount := len(filterSessionEntries(entries, "checkpoint"))
	errorCount := len(filterSessionEntries(entries, "error"))
	lastAction := stringValue(lastEntry["action"])
	if lastAction == "" {
		lastAction = stringValue(lastEntry["message"])
	}
	if lastAction == "" {
		lastAction = "Unknown"
	}

	return map[string]any{
		"completed_steps":  len(entries),
		"checkpoint_count": checkpointCount,
		"error_count":      errorCount,
		"last_action":      lastAction,
		"last_action_at":   stringValue(lastEntry["timestamp"]),
		"summary": core.Sprintf(
			"%d actions recorded, %d checkpoints, %d errors",
			len(entries),
			checkpointCount,
			errorCount,
		),
	}
}

func filterArtifactsByAction(artifacts []map[string]any, action string) []map[string]any {
	filtered := make([]map[string]any, 0, len(artifacts))
	for _, artifact := range artifacts {
		if stringValue(artifact["action"]) == action {
			filtered = append(filtered, artifact)
		}
	}
	return filtered
}

func sessionPlanSlug(session Session) string {
	if session.PlanSlug != "" {
		return session.PlanSlug
	}
	return session.Plan
}

func parseSessionListOutput(payload map[string]any) SessionListOutput {
	sessionData := payloadDataSlice(payload, "sessions")
	sessions := make([]Session, 0, len(sessionData))
	for _, values := range sessionData {
		sessions = append(sessions, parseSession(values))
	}

	count := mapIntValue(payload, "count", "total")
	if count == 0 {
		count = mapIntValue(payloadDataMap(payload), "count", "total")
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

var resultErrorValue = func(action string, result core.Result) error {
	if err, ok := result.Value.(error); ok && err != nil {
		return err
	}

	message := stringValue(result.Value)
	if message != "" {
		return core.E(action, message, nil)
	}

	return core.E(action, "request failed", nil)
}

func normaliseSessionAgentType(agentType string) (string, bool) {
	trimmed := core.Lower(core.Trim(agentType))
	if trimmed == "" {
		return "", false
	}

	switch trimmed {
	case "claude":
		return "opus", true
	case "opus", "sonnet", "haiku":
		return trimmed, true
	}

	parts := core.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return "", false
	}
	if parts[0] != "claude" {
		return "", false
	}
	switch core.Lower(core.Trim(agentType)) {
	case "claude:opus":
		return "opus", true
	case "claude:sonnet":
		return "sonnet", true
	case "claude:haiku":
		return "haiku", true
	default:
		return "", false
	}
}
