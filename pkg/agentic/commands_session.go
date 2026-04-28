// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go"
)

func (s *PrepSubsystem) registerSessionCommands() {
	c := s.Core()
	c.Command("session/get", core.Command{Description: "Read a stored session by session ID", Action: s.cmdSessionGet})
	c.Command("agentic:session/get", core.Command{Description: "Read a stored session by session ID", Action: s.cmdSessionGet})
	c.Command("session/list", core.Command{Description: "List stored sessions with optional filters", Action: s.cmdSessionList})
	c.Command("agentic:session/list", core.Command{Description: "List stored sessions with optional filters", Action: s.cmdSessionList})
	c.Command("session/start", core.Command{Description: "Start a stored session for a plan", Action: s.cmdSessionStart})
	c.Command("agentic:session/start", core.Command{Description: "Start a stored session for a plan", Action: s.cmdSessionStart})
	c.Command("session/continue", core.Command{Description: "Continue a stored session from saved context", Action: s.cmdSessionContinue})
	c.Command("agentic:session/continue", core.Command{Description: "Continue a stored session from saved context", Action: s.cmdSessionContinue})
	c.Command("session/handoff", core.Command{Description: "Hand off a stored session with context for the next agent", Action: s.cmdSessionHandoff})
	c.Command("agentic:session/handoff", core.Command{Description: "Hand off a stored session with context for the next agent", Action: s.cmdSessionHandoff})
	c.Command("session/end", core.Command{Description: "End a stored session with status, summary, and handoff notes", Action: s.cmdSessionEnd})
	c.Command("agentic:session/end", core.Command{Description: "End a stored session with status, summary, and handoff notes", Action: s.cmdSessionEnd})
	c.Command("session/complete", core.Command{Description: "Mark a stored session completed with status, summary, and handoff notes", Action: s.cmdSessionEnd})
	c.Command("agentic:session/complete", core.Command{Description: "Mark a stored session completed with status, summary, and handoff notes", Action: s.cmdSessionEnd})
	c.Command("session/log", core.Command{Description: "Add a work log entry to a stored session", Action: s.cmdSessionLog})
	c.Command("agentic:session/log", core.Command{Description: "Add a work log entry to a stored session", Action: s.cmdSessionLog})
	c.Command("session/artifact", core.Command{Description: "Record a created, modified, deleted, or reviewed artifact for a stored session", Action: s.cmdSessionArtifact})
	c.Command("agentic:session/artifact", core.Command{Description: "Record a created, modified, deleted, or reviewed artifact for a stored session", Action: s.cmdSessionArtifact})
	c.Command("session/resume", core.Command{Description: "Resume a paused or handed-off session from local cache", Action: s.cmdSessionResume})
	c.Command("agentic:session/resume", core.Command{Description: "Resume a paused or handed-off session from local cache", Action: s.cmdSessionResume})
	c.Command("session/replay", core.Command{Description: "Build replay context for a stored session", Action: s.cmdSessionReplay})
	c.Command("agentic:session/replay", core.Command{Description: "Build replay context for a stored session", Action: s.cmdSessionReplay})
}

// core-agent session get ses-abc123
func (s *PrepSubsystem) cmdSessionGet(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session get <session-id>")
		return core.Result{Value: core.E("agentic.cmdSessionGet", "session_id is required", nil), OK: false}
	}

	result := s.handleSessionGet(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionGet", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionOutput)
	if !ok {
		err := core.E("agentic.cmdSessionGet", "invalid session get output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", output.Session.SessionID)
	core.Print(nil, "status:  %s", output.Session.Status)
	core.Print(nil, "agent:   %s", output.Session.AgentType)
	if output.Session.PlanSlug != "" {
		core.Print(nil, "plan:    %s", output.Session.PlanSlug)
	}
	if output.Session.Summary != "" {
		core.Print(nil, "summary: %s", output.Session.Summary)
	}
	if output.Session.CreatedAt != "" {
		core.Print(nil, "started: %s", output.Session.CreatedAt)
	}
	if output.Session.UpdatedAt != "" {
		core.Print(nil, "updated: %s", output.Session.UpdatedAt)
	}
	if output.Session.EndedAt != "" {
		core.Print(nil, "ended:   %s", output.Session.EndedAt)
	}
	if len(output.Session.ContextSummary) > 0 {
		core.Print(nil, "context: %d item(s)", len(output.Session.ContextSummary))
	}
	if len(output.Session.WorkLog) > 0 {
		core.Print(nil, "work log: %d item(s)", len(output.Session.WorkLog))
	}
	if len(output.Session.Artifacts) > 0 {
		core.Print(nil, "artifacts: %d item(s)", len(output.Session.Artifacts))
	}
	if len(output.Session.Handoff) > 0 {
		core.Print(nil, "handoff: %d item(s)", len(output.Session.Handoff))
	}

	return core.Result{Value: output, OK: true}
}

// core-agent session list --status=active --plan=ax-follow-up
func (s *PrepSubsystem) cmdSessionList(options core.Options) core.Result {
	result := s.handleSessionList(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: optionStringValue(options, "plan_slug", "plan")},
		core.Option{Key: "agent_type", Value: optionStringValue(options, "agent_type", "agent")},
		core.Option{Key: "status", Value: optionStringValue(options, "status")},
		core.Option{Key: "limit", Value: optionIntValue(options, "limit")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionList", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionListOutput)
	if !ok {
		err := core.E("agentic.cmdSessionList", "invalid session list output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Count == 0 {
		core.Print(nil, "no sessions")
		return core.Result{Value: output, OK: true}
	}

	for _, session := range output.Sessions {
		core.Print(nil, "  %-10s %-10s %-24s %s", session.Status, session.AgentType, session.SessionID, sessionPlanSlug(session))
	}
	core.Print(nil, "%d session(s)", output.Count)
	return core.Result{Value: output, OK: true}
}

// core-agent session start ax-follow-up --agent-type=claude:opus
func (s *PrepSubsystem) cmdSessionStart(options core.Options) core.Result {
	planSlug := optionStringValue(options, "plan_slug", "plan", "_arg")
	agentType := optionStringValue(options, "agent_type", "agent")
	if planSlug == "" {
		core.Print(nil, "usage: core-agent session start <plan-slug> --agent-type=claude:opus [--context='{\"repo\":\"go-io\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionStart", "plan_slug is required", nil), OK: false}
	}
	if agentType == "" {
		core.Print(nil, "usage: core-agent session start <plan-slug> --agent-type=claude:opus [--context='{\"repo\":\"go-io\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionStart", "agent_type is required", nil), OK: false}
	}

	result := s.handleSessionStart(s.commandContext(), core.NewOptions(
		core.Option{Key: "plan_slug", Value: planSlug},
		core.Option{Key: "agent_type", Value: agentType},
		core.Option{Key: "context", Value: optionAnyMapValue(options, "context")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionStart", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionOutput)
	if !ok {
		err := core.E("agentic.cmdSessionStart", "invalid session start output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", output.Session.SessionID)
	core.Print(nil, "plan:    %s", output.Session.PlanSlug)
	core.Print(nil, "agent:   %s", output.Session.AgentType)
	core.Print(nil, "status:  %s", output.Session.Status)
	return core.Result{Value: output, OK: true}
}

// core-agent session continue ses-abc123 --agent-type=codex
func (s *PrepSubsystem) cmdSessionContinue(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	agentType := optionStringValue(options, "agent_type", "agent")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session continue <session-id> [--agent-type=codex] [--work-log='[{\"type\":\"checkpoint\",\"message\":\"...\"}]'] [--context='{\"repo\":\"go-io\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionContinue", "session_id is required", nil), OK: false}
	}

	result := s.handleSessionContinue(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
		core.Option{Key: "agent_type", Value: agentType},
		core.Option{Key: "work_log", Value: optionAnyMapSliceValue(options, "work_log")},
		core.Option{Key: "context", Value: optionAnyMapValue(options, "context")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionContinue", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionOutput)
	if !ok {
		err := core.E("agentic.cmdSessionContinue", "invalid session continue output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", output.Session.SessionID)
	core.Print(nil, "agent:   %s", output.Session.AgentType)
	core.Print(nil, "status:  %s", output.Session.Status)
	if len(output.Session.WorkLog) > 0 {
		core.Print(nil, "work log: %d item(s)", len(output.Session.WorkLog))
	}
	return core.Result{Value: output, OK: true}
}

// core-agent session handoff ses-abc123 --summary="Ready for review" --next-steps="Run the verifier"
func (s *PrepSubsystem) cmdSessionHandoff(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	summary := optionStringValue(options, "summary")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session handoff <session-id> --summary=\"Ready for review\" [--next-steps=\"Run the verifier\"] [--blockers=\"Needs input\"]")
		return core.Result{Value: core.E("agentic.cmdSessionHandoff", "session_id is required", nil), OK: false}
	}
	if summary == "" {
		core.Print(nil, "usage: core-agent session handoff <session-id> --summary=\"Ready for review\" [--next-steps=\"Run the verifier\"] [--blockers=\"Needs input\"]")
		return core.Result{Value: core.E("agentic.cmdSessionHandoff", "summary is required", nil), OK: false}
	}

	result := s.handleSessionHandoff(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
		core.Option{Key: "summary", Value: summary},
		core.Option{Key: "next_steps", Value: optionStringSliceValue(options, "next_steps", "next-steps")},
		core.Option{Key: "blockers", Value: optionStringSliceValue(options, "blockers")},
		core.Option{Key: "context_for_next", Value: optionAnyMapValue(options, "context_for_next", "context-for-next")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionHandoff", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionHandoffOutput)
	if !ok {
		err := core.E("agentic.cmdSessionHandoff", "invalid session handoff output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", sessionID)
	core.Print(nil, "summary: %s", summary)
	if blockers, ok := output.HandoffContext["blockers"].([]string); ok && len(blockers) > 0 {
		core.Print(nil, "blockers: %d", len(blockers))
	}
	if nextSteps, ok := output.HandoffContext["next_steps"].([]string); ok && len(nextSteps) > 0 {
		core.Print(nil, "next steps: %d", len(nextSteps))
	}
	return core.Result{Value: output, OK: true}
}

// core-agent session end ses-abc123 --summary="Ready for review" --status=completed
func (s *PrepSubsystem) cmdSessionEnd(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	summary := optionStringValue(options, "summary")
	status := optionStringValue(options, "status")
	if status == "" {
		status = "completed"
	}
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session end <session-id> --summary=\"Ready for review\" [--status=completed] [--handoff-notes=\"...\"]")
		return core.Result{Value: core.E("agentic.cmdSessionEnd", "session_id is required", nil), OK: false}
	}
	if summary == "" {
		core.Print(nil, "usage: core-agent session end <session-id> --summary=\"Ready for review\" [--status=completed] [--handoff-notes=\"...\"]")
		return core.Result{Value: core.E("agentic.cmdSessionEnd", "summary is required", nil), OK: false}
	}

	result := s.handleSessionEnd(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
		core.Option{Key: "status", Value: status},
		core.Option{Key: "summary", Value: summary},
		core.Option{Key: "handoff_notes", Value: optionAnyMapValue(options, "handoff_notes", "handoff-notes", "handoff")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionEnd", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionOutput)
	if !ok {
		err := core.E("agentic.cmdSessionEnd", "invalid session end output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", output.Session.SessionID)
	core.Print(nil, "status:  %s", output.Session.Status)
	core.Print(nil, "summary: %s", output.Session.Summary)
	if len(output.Session.Handoff) > 0 {
		core.Print(nil, "handoff: %d item(s)", len(output.Session.Handoff))
	}
	return core.Result{Value: output, OK: true}
}

// core-agent session log ses-abc123 --message="Checked build" --type=checkpoint
func (s *PrepSubsystem) cmdSessionLog(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	message := optionStringValue(options, "message")
	entryType := optionStringValue(options, "type")
	if entryType == "" {
		entryType = "info"
	}
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session log <session-id> --message=\"Checked build\" [--type=checkpoint] [--data='{\"key\":\"value\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionLog", "session_id is required", nil), OK: false}
	}
	if message == "" {
		core.Print(nil, "usage: core-agent session log <session-id> --message=\"Checked build\" [--type=checkpoint] [--data='{\"key\":\"value\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionLog", "message is required", nil), OK: false}
	}

	result := s.handleSessionLog(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
		core.Option{Key: "message", Value: message},
		core.Option{Key: "type", Value: entryType},
		core.Option{Key: "data", Value: optionAnyMapValue(options, "data")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionLog", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionLogOutput)
	if !ok {
		err := core.E("agentic.cmdSessionLog", "invalid session log output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", sessionID)
	core.Print(nil, "type:    %s", entryType)
	core.Print(nil, "logged:  %s", output.Logged)
	return core.Result{Value: output, OK: true}
}

// core-agent session artifact ses-abc123 --path="pkg/agentic/session.go" --action=modified --description="Tracked session metadata"
func (s *PrepSubsystem) cmdSessionArtifact(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	path := optionStringValue(options, "path")
	action := optionStringValue(options, "action")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session artifact <session-id> --path=\"pkg/agentic/session.go\" --action=modified [--description=\"...\"] [--metadata='{\"key\":\"value\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionArtifact", "session_id is required", nil), OK: false}
	}
	if path == "" {
		core.Print(nil, "usage: core-agent session artifact <session-id> --path=\"pkg/agentic/session.go\" --action=modified [--description=\"...\"] [--metadata='{\"key\":\"value\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionArtifact", "path is required", nil), OK: false}
	}
	if action == "" {
		core.Print(nil, "usage: core-agent session artifact <session-id> --path=\"pkg/agentic/session.go\" --action=modified [--description=\"...\"] [--metadata='{\"key\":\"value\"}']")
		return core.Result{Value: core.E("agentic.cmdSessionArtifact", "action is required", nil), OK: false}
	}

	result := s.handleSessionArtifact(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
		core.Option{Key: "path", Value: path},
		core.Option{Key: "action", Value: action},
		core.Option{Key: "metadata", Value: optionAnyMapValue(options, "metadata")},
		core.Option{Key: "description", Value: optionStringValue(options, "description")},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionArtifact", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionArtifactOutput)
	if !ok {
		err := core.E("agentic.cmdSessionArtifact", "invalid session artifact output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", sessionID)
	core.Print(nil, "path:    %s", path)
	core.Print(nil, "action:  %s", action)
	core.Print(nil, "artifact: %s", output.Artifact)
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdSessionResume(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session resume <session-id>")
		return core.Result{Value: core.E("agentic.cmdSessionResume", "session_id is required", nil), OK: false}
	}

	result := s.handleSessionResume(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionResume", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionResumeOutput)
	if !ok {
		err := core.E("agentic.cmdSessionResume", "invalid session resume output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session:    %s", output.Session.SessionID)
	core.Print(nil, "status:     %s", output.Session.Status)
	if len(output.HandoffContext) > 0 {
		core.Print(nil, "handoff:    %d item(s)", len(output.HandoffContext))
	}
	if len(output.RecentActions) > 0 {
		core.Print(nil, "recent:     %d action(s)", len(output.RecentActions))
	}
	if len(output.Artifacts) > 0 {
		core.Print(nil, "artifacts:  %d", len(output.Artifacts))
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) cmdSessionReplay(options core.Options) core.Result {
	sessionID := optionStringValue(options, "session_id", "session-id", "id", "_arg")
	if sessionID == "" {
		core.Print(nil, "usage: core-agent session replay <session-id>")
		return core.Result{Value: core.E("agentic.cmdSessionReplay", "session_id is required", nil), OK: false}
	}

	result := s.handleSessionReplay(s.commandContext(), core.NewOptions(
		core.Option{Key: "session_id", Value: sessionID},
	))
	if !result.OK {
		err := commandResultError("agentic.cmdSessionReplay", result)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	output, ok := result.Value.(SessionReplayOutput)
	if !ok {
		err := core.E("agentic.cmdSessionReplay", "invalid session replay output", nil)
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "session: %s", sessionID)
	core.Print(nil, "context items: %d", len(output.ReplayContext))
	return core.Result{Value: output, OK: true}
}
