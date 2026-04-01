// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	core "dappco.re/go/core"
)

func (s *PrepSubsystem) registerSessionCommands() {
	c := s.Core()
	c.Command("session/handoff", core.Command{Description: "Pause a stored session with handoff context", Action: s.cmdSessionHandoff})
	c.Command("agentic:session/handoff", core.Command{Description: "Pause a stored session with handoff context", Action: s.cmdSessionHandoff})
	c.Command("session/end", core.Command{Description: "End a stored session with status, summary, and handoff notes", Action: s.cmdSessionEnd})
	c.Command("agentic:session/end", core.Command{Description: "End a stored session with status, summary, and handoff notes", Action: s.cmdSessionEnd})
	c.Command("session/resume", core.Command{Description: "Resume a paused or handed-off session from local cache", Action: s.cmdSessionResume})
	c.Command("agentic:session/resume", core.Command{Description: "Resume a paused or handed-off session from local cache", Action: s.cmdSessionResume})
	c.Command("session/replay", core.Command{Description: "Build replay context for a stored session", Action: s.cmdSessionReplay})
	c.Command("agentic:session/replay", core.Command{Description: "Build replay context for a stored session", Action: s.cmdSessionReplay})
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
